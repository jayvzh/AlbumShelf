package thumbnail

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

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
