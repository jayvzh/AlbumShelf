// SpreadEngine：双页拼帧纯函数（算法依据 docs/SPREAD_ENGINE.md §3.3，无副作用、可单测）
import type { ImageFile } from '../types/file'
import type { SpreadFrame, SpreadLayout, SpreadOptions } from '../types/spread'

/** 帧内两图间距（px） */
export const SPREAD_FRAME_GAP = 8 // TODO(Phase 2): 页间距可配置（PRD F007）

/** 默认阅读选项（SPREAD_ENGINE.md §3.2） */
export const DEFAULT_SPREAD_OPTIONS: SpreadOptions = {
  pageMode: 'single',
  wideRatio: 1.0,
  readOrder: 'left_to_right',
  singleFirstPage: true,
  singleLastPage: false,
}

/**
 * 横图判定：width > height × wideRatio（SPREAD_ENGINE.md §2.2）。
 * 尺寸缺失或非法值按竖图处理，保证布局不中断（§4）。
 */
export function isLandscape(width: number | null, height: number | null, wideRatio: number): boolean {
  if (width == null || height == null || width <= 0 || height <= 0) return false
  return width > height * wideRatio
}

/**
 * 动态拼帧（SPREAD_ENGINE.md §3.3 伪代码 MVP）：
 * 横图独占 / 竖图两两拼页 / 下一张横图则独占 / 封面封底单页 / 末尾落单。
 * pageMode === 'single' 时为"每图一帧"的退化情形。
 */
export function buildSpreadFrames(images: ImageFile[], options: SpreadOptions): SpreadLayout {
  if (options.pageMode === 'single') {
    return {
      frames: images.map((img) => ({ images: [img], type: 'single' })),
      imageToFrame: images.map((_, index) => index),
    }
  }

  const frames: SpreadFrame[] = []
  const imageToFrame: number[] = new Array(images.length)
  const last = images.length - 1
  let i = 0
  while (i < images.length) {
    const page = images[i]
    const frameIndex = frames.length
    // 本页独占一帧并登记映射
    const takeSingle = () => {
      frames.push({ images: [page], type: 'single' })
      imageToFrame[i] = frameIndex
      i += 1
    }

    if (isLandscape(page.width, page.height, options.wideRatio)) {
      takeSingle() // 横图独占
      continue
    }

    const next = images[i + 1]
    if (i + 1 > last) {
      takeSingle() // 末尾落单（Phase 2 可插 Dummy 补位）
    } else if (isLandscape(next.width, next.height, options.wideRatio)) {
      takeSingle() // 下一页是横图，本页独占
    } else if ((i === 0 && options.singleFirstPage) || (i + 1 === last && options.singleLastPage)) {
      takeSingle() // 封面/封底单页（i+1 === last 即 next 为最后一页、自身无配对）
    } else {
      // 双页帧；right_to_left 时帧内反转（第一页在右）
      frames.push({
        images: options.readOrder === 'right_to_left' ? [next, page] : [page, next],
        type: 'spread',
      })
      imageToFrame[i] = frameIndex
      imageToFrame[i + 1] = frameIndex
      i += 2
    }
  }
  return { frames, imageToFrame }
}
