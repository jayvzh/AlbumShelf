package thumbnail

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// ErrQueueFull 对应优先级队列深度达到上限时 Enqueue/Submit 返回的哨兵错误。
var ErrQueueFull = errors.New("thumbnail: 调度队列已满")

// maxQueueDepth 单条优先级队列的深度上限（高/低各自独立计数）：
// 独立上限保证后台预热（low）永远挤不掉用户当前查看（high）的入队资格；
// 防止异常客户端无限堆积任务占用等待 goroutine（正常浏览远达不到该量级，
// 前端预热已窗口化）。
const maxQueueDepth = 256

// Priority 描述生成任务的调度优先级：
// PriorityHigh 用于用户正在查看的图（当前帧、翻页邻帧），PriorityLow 用于后台空闲预热。
type Priority uint8

const (
	PriorityHigh Priority = iota
	PriorityLow
)

// job 是一次待执行的任务单元。ran 标记 run 是否真正执行过：
// 排队期间被取消的任务会被 take() 丢弃（close(done) 但 ran=false），
// 提交方据此区分"已执行"与"排队中被取消"。
type job struct {
	prio Priority
	ctx  context.Context
	run  func()
	done chan struct{}
	ran  bool
	err  error // 仅在 run panic 被 recover 时非 nil（exec defer 兜底写入）
}

// Scheduler 是全局生成调度器：限制同时执行的 libvips 任务数（避免大图解码
// 并发踩踏把 NAS 打满），双优先级 FIFO 出队，排队中的任务随 ctx 取消即丢弃。
// 已开工的生成不可中断（libvips 同步调用），跑完落盘的结果对后续等待者依然有效。
type Scheduler struct {
	slots  chan struct{}
	mu     sync.Mutex
	queues [2][]*job

	// lastHighMicros 最近一次高优先级任务入队的时刻（UnixMicro），供后台预热器
	// 判定前台是否空闲（FrontendIdleFor）：优先级只决定排队顺序，挡不住在飞
	// 任务与 IO 竞争，预热器必须额外等前台静默后再推进。
	// 前台持续产生缩略图/预览图请求 = 正在浏览。
	lastHighMicros atomic.Int64

	// lastWarmMicros 最近一次收到前端低优先级（X-Load-Priority: low）HTTP 请求的
	// 时刻（UnixMicro）。与 lastHighMicros 区分：前者是用户可见请求（浏览中），
	// 本者是前端预加载器的闲时预热流量（WARMUP 滑窗 / DIRWARM 目录预热）。
	// 后台预热器据此（FrontendWarmIdleFor）在前端预热未完成时不推进，实现
	// "目录预热完成后才轮到全库后台构建"。打点只能来自 handler 层——若挂在
	// Submit(PriorityLow) 上，预热器自身提交的低优任务会把自己挡死。
	lastWarmMicros atomic.Int64
}

// schedulerConcurrency 记录最近一次 NewScheduler 的并发上限，供
// EffectiveVipsConcurrency 做"调度器并发 × vips 线程 ≈ 可用核数"的默认配平。
// app 组装必然先于首个生成请求，写入天然先于读取；仅影响性能默认值，不影响正确性。
var schedulerConcurrency = 2

// NewScheduler 创建并发上限为 concurrency 的调度器，小于 1 时按 1 处理。
func NewScheduler(concurrency int) *Scheduler {
	if concurrency < 1 {
		concurrency = 1
	}
	schedulerConcurrency = concurrency
	return &Scheduler{slots: make(chan struct{}, concurrency)}
}

// Submit 入队并在任务完成后返回。ctx 取消时：
//   - 尚未出队 → 任务被丢弃，返回 ctx.Err() 且 run 不执行；
//   - 已开工 → 等待本次 run 执行完毕（无法中断），返回 nil。
//
// 对应优先级队列达到 maxQueueDepth 时立即返回 ErrQueueFull，不入队。
// run panic 被 exec 兜底 recover，返回包含 panic 值的错误而非悬挂。
func (s *Scheduler) Submit(ctx context.Context, prio Priority, run func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if prio != PriorityHigh {
		prio = PriorityLow
	} else {
		s.lastHighMicros.Store(time.Now().UnixMicro())
	}
	j := &job{prio: prio, ctx: ctx, run: run, done: make(chan struct{})}
	s.mu.Lock()
	if len(s.queues[prio]) >= maxQueueDepth {
		s.mu.Unlock()
		return ErrQueueFull
	}
	s.queues[prio] = append(s.queues[prio], j)
	s.pump()
	s.mu.Unlock()

	<-j.done
	if !j.ran {
		return ctx.Err()
	}
	return j.err
}

// FrontendIdleFor 返回距最近一次高优先级任务入队已过去的时长。
// 从未有前台任务时返回最大时长（视为一直空闲）——后台预热器据此放心启动。
func (s *Scheduler) FrontendIdleFor() time.Duration {
	last := s.lastHighMicros.Load()
	if last == 0 {
		return time.Duration(math.MaxInt64)
	}
	return time.Since(time.UnixMicro(last))
}

// NoteFrontendWarmTraffic 记录一次前端低优先级（X-Load-Priority: low）HTTP 请求
// 到达。仅 handler 层调用（见 lastWarmMicros 注释）。
func (s *Scheduler) NoteFrontendWarmTraffic() {
	s.lastWarmMicros.Store(time.Now().UnixMicro())
}

// FrontendWarmIdleFor 返回距最近一次前端低优先级请求已过去的时长。
// 从未有过此类请求时返回最大时长（视为一直空闲）。
func (s *Scheduler) FrontendWarmIdleFor() time.Duration {
	last := s.lastWarmMicros.Load()
	if last == 0 {
		return time.Duration(math.MaxInt64)
	}
	return time.Since(time.UnixMicro(last))
}

// pump 尝试占用空闲槽位并派发队首任务；无任务则归还槽位。
// 调用方必须持有 s.mu。
func (s *Scheduler) pump() {
	for {
		select {
		case s.slots <- struct{}{}:
		default:
			return // 槽位已满
		}
		j := s.take()
		if j == nil {
			<-s.slots
			return
		}
		go s.exec(j)
	}
}

// take 按 高优先 → 低优先 弹出第一个未取消的任务，途中已取消的任务
// close(done) 丢弃（其等待者随即返回 ctx.Err()）。调用方必须持有 s.mu。
func (s *Scheduler) take() *job {
	for prio := PriorityHigh; prio <= PriorityLow; prio++ {
		for len(s.queues[prio]) > 0 {
			j := s.queues[prio][0]
			s.queues[prio] = s.queues[prio][1:]
			if j.ctx.Err() != nil {
				close(j.done)
				continue
			}
			return j
		}
	}
	return nil
}

// exec 执行任务，执行完先归还槽位再关闭 done，最后补位派发后续任务。
// defer 兜底：run panic 时 recover 并把包装错误带给等待者（Submit 返回而非悬挂），
// 槽位归还与 close(done) 依然必定执行，避免泄漏槽位与悬挂等待 goroutine。
func (s *Scheduler) exec(j *job) {
	j.ran = true
	defer func() {
		if r := recover(); r != nil {
			j.err = fmt.Errorf("thumbnail: 调度任务 panic: %v", r)
		}
		<-s.slots
		close(j.done)
		s.mu.Lock()
		s.pump()
		s.mu.Unlock()
	}()
	j.run()
}
