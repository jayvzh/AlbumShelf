// 图片预加载优先级调度器（模块级单例，非 Pinia store——遵守 store 白名单铁律）
// 职责：跳转时相邻帧预览图抢占置顶（取消/暂停在飞的低优先级任务腾出连接），
// 其余帧预览图低并发空闲预热。预取经 fetch 发出，响应带 immutable 头可完整落入 HTTP 缓存，
// 之后 <img> 同 URL 直接命中缓存不再发请求。当前帧自身不经此调度器：
// 由 SpreadFrame 的 <img> 原生加载（浏览器最高优先级渲染资源，不可取消）。
import { buildImageUrl } from '../services/image.service'
import type { ImageFile } from '../types/file'

// 优先级：数值小者先加载；ADJACENT = 跳转目标相邻帧，WARMUP = 空闲预热
export const PRELOAD_PRIORITY = { ADJACENT: 1, WARMUP: 2 } as const
type Priority = (typeof PRELOAD_PRIORITY)[keyof typeof PRELOAD_PRIORITY]

// 并发上限 2：浏览器每主机 6 连接，余量留给 <img> / 缩略图
const MAX_CONCURRENCY = 2

interface Task {
  url: string
  priority: Priority
  controller: AbortController
  // 被抢占（bumpToTop 让出连接）→ 结束后按原优先级重排队尾；
  // cancelAll 会清掉该标记，不重排
  preempted: boolean
}

let queue: Task[] = []
const inflight = new Map<string, Task>()
let running = 0

function queuedIndex(url: string): number {
  return queue.findIndex((t) => t.url === url)
}

// 入队（去重：已在队列或在飞的不重复）；toFront 用于抢占置顶
function enqueue(url: string, priority: Priority, toFront: boolean) {
  if (inflight.has(url) || queuedIndex(url) !== -1) return
  const task: Task = { url, priority, controller: new AbortController(), preempted: false }
  if (toFront) queue.unshift(task)
  else queue.push(task)
}

function pump() {
  while (running < MAX_CONCURRENCY && queue.length > 0) {
    const task = queue.shift()!
    running++
    inflight.set(task.url, task)
    void run(task)
  }
}

async function run(task: Task) {
  try {
    // 完整读入响应体，确保资源完整落入 HTTP 缓存。
    // WARMUP 带低优先级请求头：服务端生成调度器把它排在用户正在查看的图之后；
    // 同源自定义头不改变 URL，<img> 后续同 URL 命中缓存不受影响
    const res = await fetch(task.url, {
      signal: task.controller.signal,
      headers:
        task.priority === PRELOAD_PRIORITY.WARMUP ? { 'X-Load-Priority': 'low' } : undefined,
    })
    await res.arrayBuffer()
  } catch {
    /* 取消（被抢占 / 整体取消）或网络失败：静默 */
  } finally {
    running--
    inflight.delete(task.url)
    // 被抢占任务按原优先级重排队尾（服务端 singleflight 保证已开工的生成不白做）
    if (task.preempted && queuedIndex(task.url) === -1) {
      queue.push({ url: task.url, priority: task.priority, controller: new AbortController(), preempted: false })
    }
    pump()
  }
}

// 批量预取：priority 为 WARMUP 时重建 WARMUP 队列（丢弃旧未开始任务，按新顺序重排；
// 在飞任务不打断，enqueue 去重自然跳过）
export function preload(urls: string[], priority: Priority) {
  if (priority === PRELOAD_PRIORITY.WARMUP) {
    queue = queue.filter((t) => t.priority !== PRELOAD_PRIORITY.WARMUP)
  }
  for (const url of urls) enqueue(url, priority, false)
  pump()
}

// 跳转抢占：urls（按优先序）以 ADJACENT 插队首；在飞的 WARMUP 任务 abort 让出连接，
// 标记 preempted 后自动按原优先级重排队尾
export function bumpToTop(urls: string[]) {
  const wanted = [...new Set(urls)]
  for (const task of inflight.values()) {
    if (task.priority > PRELOAD_PRIORITY.ADJACENT && !wanted.includes(task.url)) {
      task.preempted = true
      task.controller.abort()
    }
  }
  for (let i = wanted.length - 1; i >= 0; i--) {
    const idx = queuedIndex(wanted[i])
    if (idx !== -1) queue.splice(idx, 1)
    enqueue(wanted[i], PRELOAD_PRIORITY.ADJACENT, true)
  }
  pump()
}

// 取消全部（查看器关闭/卸载）：清空队列，abort 在飞任务且不重排
export function cancelAll() {
  queue = []
  for (const task of inflight.values()) {
    task.preempted = false
    task.controller.abort()
  }
}

// 查看器换帧统一入口（桌面 ImageViewer / 移动 MobileReader 复用）：
// 1) 跳转优先：当前帧左右相邻帧的预览图抢占置顶加载；
// 2) 重建 WARMUP：其余帧预览图从当前帧+2 起按目录顺序低并发空闲预热
export function syncViewerPreload(frames: Array<{ images: ImageFile[] }>, currentIndex: number) {
  const previewUrlsAt = (i: number) => frames[i]?.images.map((img) => buildImageUrl(img, 'preview')) ?? []
  bumpToTop([...previewUrlsAt(currentIndex - 1), ...previewUrlsAt(currentIndex + 1)])
  const warmup: string[] = []
  for (let i = currentIndex + 2; i < frames.length; i++) warmup.push(...previewUrlsAt(i))
  preload(warmup, PRELOAD_PRIORITY.WARMUP)
}
