// imagePreloader 单元测试：mock global.fetch，覆盖 WARMUP 滑动窗口边界、已完成集合去重、
// cancelAll 清空后重取、>50MP 过滤与在飞并发上限 4；以及 DIRWARM 目录预热档
// （前 100 上限、解锁全量、让位高档、抢占不丢任务、低优请求头）。
// 全部异步均为 microtask 级联，flush 以一个宏任务边界排空，不依赖真实时间
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { buildImageUrl, buildThumbnailUrl } from '../services/image.service'
import type { ImageFile } from '../types/file'
import { cancelAll, markDirWarmUnlocked, syncDirWarm, syncViewerPreload } from './imagePreloader'

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

// thumb URL 期望值与生产同源：同用 buildThumbnailUrl 默认 dpr 桶，环境无关
const thumbUrlOf = (index0Based: number, w: number | null = 400, h: number | null = 600) =>
  buildThumbnailUrl(img(index0Based + 1, w, h))

/** 平铺图片列表（目录预热输入），n 张、可按 0 基索引定制宽高 */
function flatImages(n: number, size?: (i: number) => [number | null, number | null]): ImageFile[] {
  return Array.from({ length: n }, (_, i) => {
    const [w, h] = size ? size(i) : [400, 600]
    return img(i + 1, w, h)
  })
}

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

describe('syncDirWarm（DIRWARM 目录空闲预热）', () => {
  it('默认预热前 100 张 preview + 前 100 张 thumb；>50MP 仅跳过 preview', async () => {
    const images = flatImages(150, (i) => (i === 50 ? [10000, 6000] : [400, 600]))
    syncDirWarm(images, '/dir-limit')
    await flush()
    const fetched = new Set(calls.map((c) => c.url))
    for (let i = 0; i < 100; i++) {
      const [w, h] = i === 50 ? [10000, 6000] : [400, 600]
      if (i === 50) {
        expect(fetched.has(previewUrlOf(i, w, h))).toBe(false) // 60MP：preview 跳过
      } else {
        expect(fetched.has(previewUrlOf(i))).toBe(true)
      }
      expect(fetched.has(thumbUrlOf(i, w, h))).toBe(true) // thumb 不做像素过滤
    }
    // 100 名之后不预热（未解锁）
    expect(fetched.has(previewUrlOf(120))).toBe(false)
    expect(fetched.has(thumbUrlOf(120))).toBe(false)
    expect(calls.length).toBe(99 + 100) // 99 preview + 100 thumb
  })

  it('解锁后 preview 与 thumb 均扩展为全量', async () => {
    const images = flatImages(120)
    markDirWarmUnlocked('/dir-unlocked')
    syncDirWarm(images, '/dir-unlocked')
    await flush()
    const fetched = new Set(calls.map((c) => c.url))
    for (let i = 0; i < 120; i++) {
      expect(fetched.has(previewUrlOf(i))).toBe(true)
      expect(fetched.has(thumbUrlOf(i))).toBe(true)
    }
  })

  it('重建档去重不重发：同目录重复 sync 不产生新 fetch；换目录只发新目录任务', async () => {
    const images = flatImages(10)
    syncDirWarm(images, '/dir-rebuild')
    await flush()
    const afterFirst = calls.length
    expect(afterFirst).toBe(20) // 10 preview + 10 thumb
    syncDirWarm(images, '/dir-rebuild')
    await flush()
    expect(calls.length).toBe(afterFirst) // completed 去重

    const other = flatImages(6).map(
      (im, i) => ({ ...im, path: `/other/${String(i + 1).padStart(3, '0')}.jpg` }) as ImageFile,
    )
    syncDirWarm(other, '/dir-other')
    await flush()
    expect(calls.length).toBe(afterFirst + 12) // 旧档被清空，仅新目录 6 preview + 6 thumb
  })

  it('DIRWARM 让位：查看器 ADJACENT/WARMUP 全部完成后才开始更低档任务', async () => {
    const images = flatImages(60)
    const frames = images.map((im) => ({ images: [im] }))
    syncDirWarm(images, '/dir-yield')
    syncViewerPreload(frames, 9)
    await flush()
    // 窗口（含相邻帧）全部取完后，才出现第一个 thumb 请求（thumb 只属于 DIRWARM）
    const windowUrls = expectedWarmupSet(9, 60)
    let lastWindowIdx = -1
    let firstThumbIdx = calls.length
    calls.forEach((c, idx) => {
      if (windowUrls.has(c.url)) lastWindowIdx = Math.max(lastWindowIdx, idx)
      if (c.url.includes('/api/v1/thumbnail')) firstThumbIdx = Math.min(firstThumbIdx, idx)
    })
    expect(firstThumbIdx).toBeGreaterThan(lastWindowIdx)
  })

  it('bumpToTop 抢占在飞 DIRWARM：abort 后重排队尾，任务不丢失', async () => {
    const images = flatImages(12)
    syncDirWarm(images, '/dir-preempt')
    expect(calls.length).toBe(4) // 4 个 DIRWARM preview 在飞
    const inflightUrls = [...calls.map((c) => c.url)]
    syncViewerPreload(images.map((im) => ({ images: [im] })), 5) // 换帧抢占
    await flush()
    const fetched = new Set(calls.map((c) => c.url))
    for (const url of inflightUrls) expect(fetched.has(url)).toBe(true) // 被抢占的重排队尾后重取
    for (let i = 0; i < 12; i++) {
      expect(fetched.has(previewUrlOf(i))).toBe(true)
      expect(fetched.has(thumbUrlOf(i))).toBe(true)
    }
  })

  it('DIRWARM 请求（preview 与 thumb）均携带 X-Load-Priority: low', async () => {
    syncDirWarm(flatImages(6), '/dir-header')
    await flush()
    expect(calls.filter((c) => c.url.includes('/api/v1/thumbnail')).length).toBe(6)
    expect(calls.length).toBe(12)
    const headerOf = (c: { init?: FetchInit }) =>
      (c.init?.headers as Record<string, string> | undefined)?.['X-Load-Priority']
    expect(calls.every((c) => headerOf(c) === 'low')).toBe(true)
  })

  it('空列表（收藏夹虚拟视图/空目录）仅清空 DIRWARM 档', async () => {
    syncDirWarm(flatImages(6), '/dir-empty')
    await flush()
    expect(calls.length).toBe(12)
    syncDirWarm([], '/dir-empty')
    await flush()
    expect(calls.length).toBe(12) // 无新任务
  })
})
