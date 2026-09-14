// imagePreloader 单元测试：mock global.fetch，覆盖 WARMUP 滑动窗口边界、已完成集合去重、
// cancelAll 清空后重取、>50MP 过滤与在飞并发上限 4。
// 全部异步均为 microtask 级联，flush 以一个宏任务边界排空，不依赖真实时间
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { buildImageUrl } from '../services/image.service'
import type { ImageFile } from '../types/file'
import { cancelAll, syncViewerPreload } from './imagePreloader'

// RequestInit.priority 为较新标准字段，与被测模块同步做最小类型扩展
type FetchInit = RequestInit & { priority?: 'high' | 'low' | 'auto' }

/** 构造测试图片，编号写入 path 便于按帧断言 */
function img(index: number, width: number | null = 400, height: number | null = 600): ImageFile {
  const id = String(index).padStart(3, '0')
  return {
    name: `${id}.jpg`,
    path: `/test/${id}.jpg`,
    extension: 'jpg',
    size: 1024,
    width,
    height,
    modified_at: '2026-01-01T00:00:00Z',
  }
}

/** n 帧、每帧 1 张图的目录；size 可按 0-based 帧号定制宽高（null 表示元数据缺失） */
function catalog(n: number, size?: (i: number) => [number | null, number | null]) {
  return Array.from({ length: n }, (_, i) => {
    const [w, h] = size ? size(i) : [400, 600]
    return { images: [img(i + 1, w, h)] }
  })
}

const previewUrlOf = (frameIndex0Based: number, w: number | null = 400, h: number | null = 600) =>
  buildImageUrl(img(frameIndex0Based + 1, w, h), 'preview')

/** 窗口内应被预取的帧号集合（0-based，跳过当前帧） */
function expectedWarmupSet(current: number, n: number): Set<string> {
  const lo = Math.max(0, current - 4)
  const hi = Math.min(n - 1, current + 40)
  const set = new Set<string>()
  for (let i = lo; i <= hi; i++) if (i !== current) set.add(previewUrlOf(i))
  return set
}

let calls: Array<{ url: string; init?: FetchInit }>
let concurrent = 0
let maxConcurrent = 0

beforeEach(() => {
  calls = []
  concurrent = 0
  maxConcurrent = 0
  // fetch mock：立即 resolve 的最小 Response，监听 signal abort 以兑现取消语义
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, init?: FetchInit) => {
      calls.push({ url, init })
      const signal = init?.signal
      return new Promise((resolve, reject) => {
        let settled = false
        const settle = (fail: boolean) => {
          if (settled) return
          settled = true
          concurrent--
          if (fail) reject(new DOMException('Aborted', 'AbortError'))
          else resolve({ ok: true, arrayBuffer: async () => new ArrayBuffer(8) })
        }
        concurrent++
        maxConcurrent = Math.max(maxConcurrent, concurrent)
        signal?.addEventListener('abort', () => settle(true), { once: true })
        queueMicrotask(() => settle(false))
      })
    }),
  )
})

afterEach(() => {
  // 重置模块单例状态（清空 queue/inflight/completed）并还原 fetch
  cancelAll()
  vi.unstubAllGlobals()
})

/** 排空全部任务链：microtask 级联在下一个宏任务前全部执行完 */
const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0))

describe('syncViewerPreload', () => {
  it('滑动窗口边界：n=500、current=10 只预热第 6~50 帧（跳过当前帧自身）', async () => {
    syncViewerPreload(catalog(500), 9)
    await flush()
    expect(new Set(calls.map((c) => c.url))).toEqual(expectedWarmupSet(9, 500))
  })

  it('已完成集合去重：成功预取后再次 sync 不重复 fetch', async () => {
    syncViewerPreload(catalog(60), 9)
    await flush()
    const afterFirst = calls.length
    expect(afterFirst).toBeGreaterThan(0)
    syncViewerPreload(catalog(60), 9)
    await flush()
    expect(calls.length).toBe(afterFirst)
  })

  it('cancelAll 清空已完成集合与队列后可重新预取（含被 abort 的在飞任务）', async () => {
    syncViewerPreload(catalog(60), 9)
    expect(calls.length).toBe(4) // 此刻在飞：2 ADJACENT + 2 WARMUP
    cancelAll()
    await flush()
    expect(calls.length).toBe(4) // 在飞任务被 abort，未完成的不再继续
    syncViewerPreload(catalog(60), 9)
    await flush()
    // 全部窗口 URL（含被 abort 的 4 个）都重新发起了 fetch
    const fetched = new Set(calls.map((c) => c.url))
    expect(fetched).toEqual(expectedWarmupSet(9, 60))
    expect(calls.length).toBeGreaterThan(4)
  })

  it('>50MP 图片不进 WARMUP；元数据缺失（null）或为 0 时不过滤', async () => {
    // current=9 → 窗口 [5,19]；帧 11 为 60MP、帧 12 缺失、帧 13 为 0
    const frames = catalog(20, (i) => {
      if (i === 11) return [10000, 6000]
      if (i === 12) return [null, null]
      if (i === 13) return [0, 0]
      return [400, 600]
    })
    syncViewerPreload(frames, 9)
    await flush()
    const fetched = new Set(calls.map((c) => c.url))
    expect(fetched.has(previewUrlOf(11, 10000, 6000))).toBe(false)
    expect(fetched.has(previewUrlOf(12, null, null))).toBe(true)
    expect(fetched.has(previewUrlOf(13, 0, 0))).toBe(true)
    const expected = expectedWarmupSet(9, 20)
    expected.delete(previewUrlOf(11, 10000, 6000))
    expect(fetched).toEqual(expected)
  })

  it('在飞并发上限为 4（MAX_CONCURRENCY）', async () => {
    syncViewerPreload(catalog(60), 9)
    expect(calls.length).toBe(4) // pump 同步启动：2 ADJACENT + 2 WARMUP
    await flush()
    expect(maxConcurrent).toBe(4)
  })

  it('ADJACENT 抢占 fetch 携带 priority: high，WARMUP 不携带', async () => {
    syncViewerPreload(catalog(12), 5)
    const adjacent = calls.filter((c) => c.init?.priority === 'high').map((c) => c.url)
    expect(adjacent.sort()).toEqual([previewUrlOf(4), previewUrlOf(6)].sort())
    await flush()
    expect(calls.every((c) => c.init?.priority === 'high' || c.init?.priority === undefined)).toBe(true)
  })
})
