// buildThumbnailUrl 缩略图桶选择测试：默认按 devicePixelRatio 自适应，显式 width 不被覆盖
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ImageFile } from '../types/file'
import { buildThumbnailUrl } from './image.service'

const image: ImageFile = {
  name: '01.jpg',
  path: '/test/01.jpg',
  extension: 'jpg',
  size: 1024,
  width: 400,
  height: 600,
  modified_at: '2026-01-01T00:00:00Z',
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('buildThumbnailUrl 桶选择', () => {
  it('dpr=1/2/3 分别命中 200/300/500 桶', () => {
    vi.stubGlobal('window', { devicePixelRatio: 1 })
    expect(buildThumbnailUrl(image)).toContain('&width=200&')
    vi.stubGlobal('window', { devicePixelRatio: 2 })
    expect(buildThumbnailUrl(image)).toContain('&width=300&')
    vi.stubGlobal('window', { devicePixelRatio: 3 })
    expect(buildThumbnailUrl(image)).toContain('&width=500&')
  })

  it('无 window（SSR/测试环境）回退 300 桶', () => {
    expect(buildThumbnailUrl(image)).toContain('&width=300&')
  })

  it('显式 width 传参不被 dpr 覆盖', () => {
    vi.stubGlobal('window', { devicePixelRatio: 3 })
    expect(buildThumbnailUrl(image, 300)).toContain('&width=300&')
    expect(buildThumbnailUrl(image, 200)).toContain('&width=200&')
  })
})
