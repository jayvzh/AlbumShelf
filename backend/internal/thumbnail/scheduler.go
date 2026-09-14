package thumbnail

import (
	"context"
	"sync"
)

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
}

// Scheduler 是全局生成调度器：限制同时执行的 libvips 任务数（避免大图解码
// 并发踩踏把 NAS 打满），双优先级 FIFO 出队，排队中的任务随 ctx 取消即丢弃。
// 已开工的生成不可中断（libvips 同步调用），跑完落盘的结果对后续等待者依然有效。
type Scheduler struct {
	slots  chan struct{}
	mu     sync.Mutex
	queues [2][]*job
}

// NewScheduler 创建并发上限为 concurrency 的调度器，小于 1 时按 1 处理。
func NewScheduler(concurrency int) *Scheduler {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Scheduler{slots: make(chan struct{}, concurrency)}
}

// Submit 入队并在任务完成后返回。ctx 取消时：
//   - 尚未出队 → 任务被丢弃，返回 ctx.Err() 且 run 不执行；
//   - 已开工 → 等待本次 run 执行完毕（无法中断），返回 nil。
func (s *Scheduler) Submit(ctx context.Context, prio Priority, run func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if prio != PriorityHigh {
		prio = PriorityLow
	}
	j := &job{prio: prio, ctx: ctx, run: run, done: make(chan struct{})}
	s.mu.Lock()
	s.queues[prio] = append(s.queues[prio], j)
	s.pump()
	s.mu.Unlock()

	<-j.done
	if !j.ran {
		return ctx.Err()
	}
	return nil
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
func (s *Scheduler) exec(j *job) {
	j.ran = true
	j.run()
	<-s.slots
	close(j.done)
	s.mu.Lock()
	s.pump()
	s.mu.Unlock()
}
