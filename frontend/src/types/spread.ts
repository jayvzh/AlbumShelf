// 双页 Spread 类型契约（见 docs/API.md §4 / docs/SPREAD_ENGINE.md）
import type { ImageFile } from './file'

/** 显示模式：单页 / 双页拼页 */
export type PageMode = 'single' | 'spread'

/** 阅读方向：左起（普通）/ 右起（日漫） */
export type ReadOrder = 'left_to_right' | 'right_to_left'

/** 双页阅读选项（对应 folder_settings 的 spread 字段，可保存复用） */
export interface SpreadOptions {
  pageMode: PageMode
  /** 横图判定阈值：width > height × wideRatio 判为横图，默认 1.0 */
  wideRatio: number
  readOrder: ReadOrder
  /** 封面（第一张）单页显示，默认 true */
  singleFirstPage: boolean
  /** 封底（最后一张）单页显示，默认 false */
  singleLastPage: boolean
}

/** 一"帧" = 屏幕上一次呈现的内容（1 张单页/横图独占，或 2 张竖图拼页） */
export interface SpreadFrame {
  images: ImageFile[]
  type: 'single' | 'spread'
}

/** 拼帧结果：帧列表 + imageIndex → frameIndex 映射（供 Filmstrip / 翻页 / 模式切换反查） */
export interface SpreadLayout {
  frames: SpreadFrame[]
  imageToFrame: number[]
}
