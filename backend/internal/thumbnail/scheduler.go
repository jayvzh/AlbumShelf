package thumbnail

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrQueueFull 队列深度达到上限时 Submit 返回的哨兵错误。
var ErrQueueFull = errors.New("thumbnail: 调度队列已满")

// maxQueueDepth 高低优先级队列总深度上限：防止异常客户端无限堆积任务
// 占用等待 goroutine（正常浏览远达不到该量级，前端预热已窗口化）。
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
// 队列总深度达到 maxQueueDepth 时立即返回 ErrQueueFull，不入队。
// run panic 被 exec 兜底 recover，Submit 返回包含 panic 值的错误而非悬挂。
func (s *Scheduler) Submit(ctx context.Context, prio Priority, run func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if prio != PriorityHigh {
		prio = PriorityLow
	}
	j := &job{prio: prio, ctx: ctx, run: run, done: make(chan struct{})}
	s.mu.Lock()
	if len(s.queues[PriorityHigh])+len(s.queues[PriorityLow]) >= maxQueueDepth {
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
