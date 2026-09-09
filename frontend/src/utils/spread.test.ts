// SpreadEngine 单元测试（覆盖 docs/SPREAD_ENGINE.md §6 七项要求 + 补充边界）
import { describe, expect, it } from 'vitest'
import type { ImageFile } from '../types/file'
import type { SpreadFrame, SpreadOptions } from '../types/spread'
import { DEFAULT_SPREAD_OPTIONS, buildSpreadFrames, isLandscape } from './spread'

/** 构造测试图片，编号写入 name 便于断言分组 */
function img(index: number, width: number | null, height: number | null): ImageFile {
  const id = String(index).padStart(2, '0')
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

/** n 张竖图（400×600，编号 1..n） */
function portraits(n: number): ImageFile[] {
  return Array.from({ length: n }, (_, i) => img(i + 1, 400, 600))
}

// 配对行为基线：显式关闭封面单页（DEFAULT.singleFirstPage=true 会让首图落单，封面行为由专门用例覆盖）
const spread = (overrides: Partial<SpreadOptions> = {}): SpreadOptions => ({
  ...DEFAULT_SPREAD_OPTIONS,
  pageMode: 'spread',
  singleFirstPage: false,
  ...overrides,
})

/** 提取各帧图片编号，便于断言分组 */
function groups(frames: SpreadFrame[]): number[][] {
  return frames.map((f) => f.images.map((image) => Number(image.name.slice(0, 2))))
}

describe('isLandscape', () => {
  it('尺寸缺失（null）或非法值按竖图处理', () => {
    expect(isLandscape(null, null, 1.0)).toBe(false)
    expect(isLandscape(0, 0, 1.0)).toBe(false)
  })
})

describe('buildSpreadFrames', () => {
  it('全竖图序列两两配对 [1,2],[3,4],[5,6]', () => {
    const { frames } = buildSpreadFrames(portraits(6), spread())
    expect(groups(frames)).toEqual([[1, 2], [3, 4], [5, 6]])
    expect(frames.every((f) => f.type === 'spread')).toBe(true)
  })

  it('横图打断：竖,竖,横,竖,竖 → [1,2],[3],[4,5]', () => {
    const images = [img(1, 400, 600), img(2, 400, 600), img(3, 1200, 600), img(4, 400, 600), img(5, 400, 600)]
    const { frames } = buildSpreadFrames(images, spread())
    expect(groups(frames)).toEqual([[1, 2], [3], [4, 5]])
    expect(frames[1].type).toBe('single')
  })

  it('末尾落单竖图独占一帧', () => {
    const { frames } = buildSpreadFrames(portraits(3), spread())
    expect(groups(frames)).toEqual([[1, 2], [3]])
    expect(frames[1].type).toBe('single')
  })

  it('封面单页开启：第一张竖图不与第二张拼页', () => {
    const { frames } = buildSpreadFrames(portraits(4), spread({ singleFirstPage: true }))
    expect(groups(frames)).toEqual([[1], [2, 3], [4]])

    // 对照：关闭后恢复两两配对
    const off = buildSpreadFrames(portraits(4), spread({ singleFirstPage: false }))
    expect(groups(off.frames)).toEqual([[1, 2], [3, 4]])
  })

  it('封底单页开启：最后一张竖图不拼页', () => {
    const { frames } = buildSpreadFrames(portraits(4), spread({ singleLastPage: true }))
    expect(groups(frames)).toEqual([[1, 2], [3], [4]])
  })

  it('WideRatio=1.2 正方形判竖图；WideRatio=0.8 判横图', () => {
    expect(isLandscape(100, 100, 1.2)).toBe(false)
    expect(isLandscape(100, 100, 0.8)).toBe(true)

    const images = [img(1, 100, 100), img(2, 400, 600)]
    expect(groups(buildSpreadFrames(images, spread({ wideRatio: 1.2 })).frames)).toEqual([[1, 2]])
    expect(groups(buildSpreadFrames(images, spread({ wideRatio: 0.8 })).frames)).toEqual([[1], [2]])
  })

  it('right_to_left：双页帧内图片顺序反转（第一页在右）', () => {
    const { frames } = buildSpreadFrames(portraits(4), spread({ readOrder: 'right_to_left' }))
    expect(groups(frames)).toEqual([[2, 1], [4, 3]])
  })

  it('imageToFrame 映射正确', () => {
    const images = [img(1, 400, 600), img(2, 400, 600), img(3, 1200, 600), img(4, 400, 600), img(5, 400, 600)]
    const { frames, imageToFrame } = buildSpreadFrames(images, spread())
    expect(frames).toHaveLength(3)
    expect(imageToFrame).toEqual([0, 0, 1, 2, 2])
  })

  it('模式切换（spread → single）后当前图片不跳变：映射反查定位', () => {
    const images = [img(1, 400, 600), img(2, 400, 600), img(3, 1200, 600), img(4, 400, 600), img(5, 400, 600)]
    // spread 模式下用户正看着第 3 张（横图，frameIndex=1）
    const spreadLayout = buildSpreadFrames(images, spread())
    expect(spreadLayout.imageToFrame[2]).toBe(1)

    // 切到 single 模式：按"当前图片"反查新帧号，不按旧帧号平移
    const currentImageIndex = 2
    const singleLayout = buildSpreadFrames(images, { ...DEFAULT_SPREAD_OPTIONS })
    expect(singleLayout.imageToFrame[currentImageIndex]).toBe(2)
    expect(singleLayout.frames[2].images[0]).toBe(images[2])
  })

  it('pageMode=single 为每图一帧的退化情形', () => {
    const { frames, imageToFrame } = buildSpreadFrames(portraits(3), { ...DEFAULT_SPREAD_OPTIONS })
    expect(frames).toHaveLength(3)
    expect(frames.every((f) => f.type === 'single' && f.images.length === 1)).toBe(true)
    expect(imageToFrame).toEqual([0, 1, 2])
  })

  it('空目录返回空帧', () => {
    const { frames, imageToFrame } = buildSpreadFrames([], spread())
    expect(frames).toEqual([])
    expect(imageToFrame).toEqual([])
  })
})
