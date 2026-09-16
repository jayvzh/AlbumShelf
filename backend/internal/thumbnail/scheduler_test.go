package thumbnail

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// VIPS_CONCURRENCY 显式设置（≥1）优先；未设置或非法时按"可用核数 ÷ 调度器并发"配平（至少 1）。
func TestEffectiveVipsConcurrency(t *testing.T) {
	t.Setenv("VIPS_CONCURRENCY", "3")
	if got := EffectiveVipsConcurrency(); got != 3 {
		t.Fatalf("显式设置应直接采用 3: %d", got)
	}

	t.Setenv("VIPS_CONCURRENCY", "0") // 非法值视为未设置，走配平
	NewScheduler(2)
	want := runtime.GOMAXPROCS(0) / 2
	if want < 1 {
		want = 1
	}
	if got := EffectiveVipsConcurrency(); got != want {
		t.Fatalf("未设置时应配平为 %d: %d", want, got)
	}
}

// 并发上限：8 个任务、上限 2，运行峰值不得超过 2。
func TestSchedulerConcurrencyLimit(t *testing.T) {
	s := NewScheduler(2)
	var mu sync.Mutex
	running, peak := 0, 0
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Submit(context.Background(), PriorityLow, func() {
				mu.Lock()
				running++
				if running > peak {
					peak = running
				}
				mu.Unlock()
				time.Sleep(20 * time.Millisecond)
				mu.Lock()
				running--
				mu.Unlock()
			}); err != nil {
				t.Errorf("Submit 不应报错: %v", err)
			}
		}()
	}
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if peak > 2 {
		t.Fatalf("峰值并发 %d 超过上限 2", peak)
	}
	if peak < 2 {
		t.Fatalf("峰值并发 %d 未达到上限 2，调度器可能串行", peak)
	}
}

// 优先级：占用唯一槽位后依次入队 low、high，释放后执行顺序应为 first → high → low。
func TestSchedulerPriorityOrder(t *testing.T) {
	s := NewScheduler(1)
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	var order []string

	firstErr := make(chan error, 1)
	go func() {
		firstErr <- s.Submit(context.Background(), PriorityHigh, func() {
			close(started)
			<-release
			mu.Lock()
			order = append(order, "first")
			mu.Unlock()
		})
	}()
	<-started // 确认槽位已被 first 占用

	lowErr := make(chan error, 1)
	highErr := make(chan error, 1)
	go func() {
		lowErr <- s.Submit(context.Background(), PriorityLow, func() {
			mu.Lock()
			order = append(order, "low")
			mu.Unlock()
		})
	}()
	go func() {
		highErr <- s.Submit(context.Background(), PriorityHigh, func() {
			mu.Lock()
			order = append(order, "high")
			mu.Unlock()
		})
	}()

	// 同包测试直接轮询内部队列，确定性地等待两任务都已入队
	//（不能靠 sleep：close(release) 若早于 high 入队，pump 会先派发 low）
	deadline := time.Now().Add(2 * time.Second)
	for {
		s.mu.Lock()
		enqueued := len(s.queues[PriorityLow]) == 1 && len(s.queues[PriorityHigh]) == 1
		s.mu.Unlock()
		if enqueued {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("low/high 任务未能按时入队")
		}
		time.Sleep(time.Millisecond)
	}

	close(release)
	if err := <-firstErr; err != nil {
		t.Fatalf("first Submit: %v", err)
	}
	if err := <-highErr; err != nil {
		t.Fatalf("high Submit: %v", err)
	}
	if err := <-lowErr; err != nil {
		t.Fatalf("low Submit: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 || order[0] != "first" || order[1] != "high" || order[2] != "low" {
		t.Fatalf("执行顺序 %v，期望 [first high low]", order)
	}
}

// 排队中取消：任务被丢弃（run 不执行），Submit 返回 context.Canceled。
func TestSchedulerCancelWhileQueued(t *testing.T) {
	s := NewScheduler(1)
	started := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = s.Submit(context.Background(), PriorityHigh, func() {
			close(started)
			<-release
		})
	}()
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	executed := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Submit(ctx, PriorityLow, func() { close(executed) })
	}()
	cancel()

	close(release) // 释放槽位触发 pump，丢弃已取消的排队任务
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("期望 context.Canceled，得到 %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Submit 未在释放槽位后返回")
	}
	select {
	case <-executed:
		t.Fatal("已取消的任务不应执行")
	default:
	}
}

// 提交时 ctx 已取消：立即返回错误且不执行。
func TestSchedulerSubmitWithCancelledContext(t *testing.T) {
	s := NewScheduler(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ran := false
	err := s.Submit(ctx, PriorityHigh, func() { ran = true })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("期望 context.Canceled，得到 %v", err)
	}
	if ran {
		t.Fatal("任务不应执行")
	}
}

// 队列上限：唯一槽位被阻塞任务占用、队列填满后，Submit 立即返回 ErrQueueFull。
func TestSchedulerQueueFull(t *testing.T) {
	s := NewScheduler(1)
	started := make(chan struct{})
	release := make(chan struct{})
	blockerDone := make(chan error, 1)
	go func() {
		blockerDone <- s.Submit(context.Background(), PriorityHigh, func() {
			close(started)
			<-release
		})
	}()
	<-started // 确保槽位已被阻塞任务占用

	// 同包测试直接填充 low 队列到上限（起 256 个 Submit goroutine 既慢又无必要）
	s.mu.Lock()
	for len(s.queues[PriorityLow]) < maxQueueDepth {
		s.queues[PriorityLow] = append(s.queues[PriorityLow],
			&job{prio: PriorityLow, ctx: context.Background(), run: func() {}, done: make(chan struct{})})
	}
	s.mu.Unlock()

	// low 队列已满：同优先级提交立即拒绝
	if err := s.Submit(context.Background(), PriorityLow, func() {}); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("期望 ErrQueueFull，得到 %v", err)
	}

	// 独立上限：low 塞满不得挤掉 high 的入队资格——high 应正常入队并在释放后执行
	highRan := make(chan struct{})
	highDone := make(chan error, 1)
	go func() {
		highDone <- s.Submit(context.Background(), PriorityHigh, func() { close(highRan) })
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		s.mu.Lock()
		enqueued := len(s.queues[PriorityHigh]) == 1
		s.mu.Unlock()
		if enqueued {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("high 任务未能按时入队（被 low 队列挤占）")
		}
		time.Sleep(time.Millisecond)
	}

	// 清理：丢弃 low 填充任务并释放阻塞槽位
	s.mu.Lock()
	for _, j := range s.queues[PriorityLow] {
		close(j.done)
	}
	s.queues[PriorityLow] = nil
	s.mu.Unlock()
	close(release)

	select {
	case <-highRan:
	case <-time.After(2 * time.Second):
		t.Fatal("high 任务未执行")
	}
	if err := <-highDone; err != nil {
		t.Fatalf("high Submit: %v", err)
	}
	if err := <-blockerDone; err != nil {
		t.Fatalf("阻塞任务 Submit: %v", err)
	}
}

// 前端低优流量打点：从未打点视为一直空闲；打点后 FrontendWarmIdleFor 接近零；
// 与高优先级活跃度（FrontendIdleFor）互不影响——两条判定服务于不同的让路语义。
func TestSchedulerFrontendWarmTrafficTracking(t *testing.T) {
	s := NewScheduler(1)

	if s.FrontendWarmIdleFor() < time.Hour {
		t.Fatal("从未打点应视为一直空闲（返回最大时长）")
	}

	s.NoteFrontendWarmTraffic()
	if idle := s.FrontendWarmIdleFor(); idle > time.Second {
		t.Fatalf("刚打点后空闲时长应接近零: %v", idle)
	}
	if s.FrontendIdleFor() < time.Hour {
		t.Fatal("低优打点不应影响高优先级活跃判定")
	}

	// 反向：高优先级 Submit 不触碰低优流量打点
	if err := s.Submit(context.Background(), PriorityHigh, func() {}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if s.FrontendIdleFor() > time.Second {
		t.Fatal("高优提交后 FrontendIdleFor 应接近零")
	}
}

// panic 防御：run panic 时 Submit 返回包含 panic 值的错误而非悬挂，
// 且槽位被归还，后续任务仍可正常执行。
func TestSchedulerPanicRecovery(t *testing.T) {
	s := NewScheduler(1)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Submit(context.Background(), PriorityHigh, func() { panic("boom") })
	}()

	var err error
	select {
	case err = <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("panic 任务导致 Submit 悬挂")
	}
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("期望包含 panic 值的错误，得到 %v", err)
	}

	// 槽位未泄漏：后续任务仍能执行
	ran := make(chan struct{})
	if err := s.Submit(context.Background(), PriorityHigh, func() { close(ran) }); err != nil {
		t.Fatalf("panic 后 Submit 失败: %v", err)
	}
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("panic 后槽位泄漏，任务未执行")
	}
}
