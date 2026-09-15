// 图片预加载优先级调度器（模块级单例，非 Pinia store——遵守 store 白名单铁律）
// 职责：跳转时相邻帧预览图抢占置顶（取消/暂停在飞的低优先级任务腾出连接），
// 其余帧预览图低并发空闲预热；用户停留在目录时以最低档预热当前目录前缀的
// 预览图+缩略图（DIRWARM）。预取经 fetch 发出，响应带 immutable 头可完整落入
// HTTP 缓存，之后 <img> 同 URL 直接命中缓存不再发请求。当前帧自身不经此调度器：
// 由 SpreadFrame 的 <img> 原生加载（浏览器最高优先级渲染资源，不可取消）。
import { buildImageUrl, buildThumbnailUrl } from '../services/image.service'
import type { ImageFile } from '../types/file'

// 优先级：数值小者先加载；ADJACENT = 跳转目标相邻帧，WARMUP = 查看器滑窗空闲预热，
// DIRWARM = 目录空闲预热（最低档，仅在更高档无排队时才获得连接）
export const PRELOAD_PRIORITY = { ADJACENT: 1, WARMUP: 2, DIRWARM: 3 } as const
type Priority = (typeof PRELOAD_PRIORITY)[keyof typeof PRELOAD_PRIORITY]

// 并发上限 4：LAN 主场景下服务端生成为瓶颈，4 并发仍在浏览器每主机 6 连接限制内，
// 余量留给 <img> / 缩略图
const MAX_CONCURRENCY = 4

// WARMUP 滑动窗口：当前帧向前预热 40 帧、向后保留 4 帧（覆盖回看场景），随换帧滑动
const WARMUP_FORWARD = 40
const WARMUP_BACKWARD = 4
// 超过 50MP 的图片不进 WARMUP/DIRWARM preview 队列：预热收益低于服务端生成/传输成本
// （ADJACENT 与当前帧不受限）
const MAX_WARMUP_PIXELS = 50_000_000

// DIRWARM 目录预热默认前缀上限：防千张万张目录一次全量预热打满算力/IO；
// 解锁（用户查看第 101 张及以后，索引 ≥ DIR_UNLOCK_THRESHOLD）后放开为全量
const DIR_PREVIEW_LIMIT = 100
const DIR_THUMB_LIMIT = 100
export const DIR_UNLOCK_THRESHOLD = 100

// RequestInit.priority 为较新标准字段（fetch 请求优先级），部分 TS lib 版本尚未收录，最小扩展保持类型安全
type FetchInit = RequestInit & { priority?: 'high' | 'low' | 'auto' }

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
// 已成功预取的 URL：查看器反复换帧会重复经过同一帧，enqueue 前据此跳过，不再重复 fetch；
// cancelAll（查看器关闭）时清空，重新打开可全量预热
const completed = new Set<string>()
let running = 0

function queuedIndex(url: string): number {
  return queue.findIndex((t) => t.url === url)
}

// 入队（去重：已成功预取、已在队列或在飞的不重复）；toFront 用于抢占置顶
function enqueue(url: string, priority: Priority, toFront: boolean) {
  if (completed.has(url) || inflight.has(url) || queuedIndex(url) !== -1) return
  const task: Task = { url, priority, controller: new AbortController(), preempted: false }
  if (toFront) queue.unshift(task)
  else queue.push(task)
}

// 出队：按优先级值最小者先执行（同值取先出现者，保持档内 FIFO）。
// 三档混排后（高档重建 filter 后 append、抢占任务重排队尾）数组不再全局有序，
// 纯 FIFO shift 会让 DIRWARM 排到新 WARMUP 之前，必须按优先级显式选队首。
function pump() {
  while (running < MAX_CONCURRENCY && queue.length > 0) {
    let best = 0
    for (let i = 1; i < queue.length; i++) {
      if (queue[i].priority < queue[best].priority) best = i
    }
    const task = queue.splice(best, 1)[0]
    running++
    inflight.set(task.url, task)
    void run(task)
  }
}

async function run(task: Task) {
  try {
    // 完整读入响应体，确保资源完整落入 HTTP 缓存。
    // ADJACENT 携带 priority: 'high'（浏览器 fetch 请求优先级，抢在普通资源之前）；
    // WARMUP/DIRWARM 带低优先级请求头：服务端生成调度器把它排在用户正在查看的图之后；
    // 同源自定义头不改变 URL，<img> 后续同 URL 命中缓存不受影响
    const init: FetchInit = {
      signal: task.controller.signal,
      priority: task.priority === PRELOAD_PRIORITY.ADJACENT ? 'high' : undefined,
      headers:
        task.priority > PRELOAD_PRIORITY.ADJACENT ? { 'X-Load-Priority': 'low' } : undefined,
    }
    const res = await fetch(task.url, init)
    await res.arrayBuffer()
    // 仅成功响应记入已完成集合（非 2xx / 取消 / 网络失败不记，允许后续重试）
    if (res.ok) completed.add(task.url)
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

// 取消全部（查看器关闭/卸载）：清空队列与已完成集合（重开可全量预热），
// abort 在飞任务且不重排
export function cancelAll() {
  queue = []
  completed.clear()
  for (const task of inflight.values()) {
    task.preempted = false
    task.controller.abort()
  }
}

// 查看器换帧统一入口（桌面 ImageViewer / 移动 MobileReader 复用）：
// 1) 跳转优先：当前帧左右相邻帧的预览图抢占置顶加载；
// 2) 重建 WARMUP：滑动窗口 [currentIndex-WARMUP_BACKWARD, currentIndex+WARMUP_FORWARD] 内的
//    帧预览图低并发空闲预热（500 帧目录、当前第 10 帧 → 第 6~50 帧）
export function syncViewerPreload(frames: Array<{ images: ImageFile[] }>, currentIndex: number) {
  const previewUrlsAt = (i: number) => frames[i]?.images.map((img) => buildImageUrl(img, 'preview')) ?? []
  bumpToTop([...previewUrlsAt(currentIndex - 1), ...previewUrlsAt(currentIndex + 1)])
  const warmup: string[] = []
  const lo = Math.max(0, currentIndex - WARMUP_BACKWARD)
  const hi = Math.min(frames.length - 1, currentIndex + WARMUP_FORWARD)
  for (let i = lo; i <= hi; i++) {
    if (i === currentIndex) continue // 当前帧由 <img> 原生加载，不经调度器重复预取
    for (const img of frames[i]?.images ?? []) {
      // 超大图跳过；width/height 元数据缺失（null）或为 0 时不过滤
      if (img.width && img.height && img.width * img.height > MAX_WARMUP_PIXELS) continue
      warmup.push(buildImageUrl(img, 'preview'))
    }
  }
  // 窗口内已被 ADJACENT/在飞/已成功预取覆盖的 URL 由 enqueue 去重自然跳过
  preload(warmup, PRELOAD_PRIORITY.WARMUP)
}

// DIRWARM 解锁目录（会话级，内存 Set）：用户在本目录手动查看第 101 张及以后
// （索引 ≥ DIR_UNLOCK_THRESHOLD）后解锁——前缀限制放开为全量 preview + 全量 thumb。
// 不落 localStorage：库内容会变化，持久化解锁易过期失真；重新深翻一次即可再解锁。
const unlockedDirs = new Set<string>()

// 标记目录解锁（幂等）；生效由调用方随后重调 syncDirWarm 重建全量清单完成。
export function markDirWarmUnlocked(dirPath: string) {
  unlockedDirs.add(dirPath)
}

// 目录空闲预热统一入口（桌面/移动浏览页调用）：重建 DIRWARM 档——
// 阶段 A 先 preview 前 DIR_PREVIEW_LIMIT 张（解锁后全量；preview 是打开即看的
// 瓶颈资源，且后续 thumb 可从新鲜 preview 派生，每图只解码一次原图）；
// 阶段 B 再 thumb 前 DIR_THUMB_LIMIT 张（解锁后全量；与胶片条/网格同 dpr 桶，
// 浏览器 HTTP 缓存天然去重，不做像素过滤）。
// 重建语义同 WARMUP：丢弃旧未开始任务按新清单重排；在飞任务不打断（enqueue
// 去重跳过）。收藏夹虚拟视图/空目录传空数组即清空该档。
// 重复任务语义 = 提权：用户点击的图若已在低档队列，bumpToTop 会移除后插队首；
// 在飞同 URL 不打断（服务端 singleflight + 高撞低双跑兜底）。
export function syncDirWarm(images: ImageFile[], dirPath: string) {
  queue = queue.filter((t) => t.priority !== PRELOAD_PRIORITY.DIRWARM)
  const limit = unlockedDirs.has(dirPath) ? images.length : 0
  const previewCount = Math.max(limit, Math.min(DIR_PREVIEW_LIMIT, images.length))
  const thumbCount = Math.max(limit, Math.min(DIR_THUMB_LIMIT, images.length))
  for (let i = 0; i < previewCount; i++) {
    const img = images[i]
    // 超大图跳过（同 WARMUP 成本收益判断）；宽高元数据缺失（null）或为 0 时不过滤
    if (img.width && img.height && img.width * img.height > MAX_WARMUP_PIXELS) continue
    enqueue(buildImageUrl(img, 'preview'), PRELOAD_PRIORITY.DIRWARM, false)
  }
  for (let i = 0; i < thumbCount; i++) {
    enqueue(buildThumbnailUrl(images[i]), PRELOAD_PRIORITY.DIRWARM, false)
  }
  pump()
}
