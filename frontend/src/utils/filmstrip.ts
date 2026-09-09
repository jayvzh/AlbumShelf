// Filmstrip 布局纯函数：等高不等宽缩略图条的宽度 / 前缀和偏移 / 可见区间二分（算法见 docs/ARCHITECTURE.md §4.6）
// 禁止引入 DOM / Vue 依赖，保证可被单测直接调用；响应式接线见 composables/useFilmstrip.ts

// 显示高度（px）：与后端缩略图生成尺寸（200px）解耦
export const THUMB_H = 96
// 项与项之间的固定间距（px）
export const GAP = 8
// 项宽 clamp 下限（px）：竖图不低于此宽
export const MIN_W = 48
// 项宽 clamp 上限（px）：横图不超过此宽
export const MAX_W = 288
// 可见区间两端额外渲染的项数（虚拟渲染缓冲）
export const OVERSCAN = 3

// 宽或高缺失（width/height 为 null）时按竖图 2:3 兜底比例计算宽度
const FALLBACK_RATIO = 2 / 3

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

// 单项显示宽度：THUMB_H × (w/h) 后 clamp 到 [MIN_W, MAX_W]；宽或高为 null 时按竖图比例兜底
export function computeItemWidth(width: number | null, height: number | null): number {
  const ratio = width !== null && height !== null && height > 0 ? width / height : FALLBACK_RATIO
  return clamp(THUMB_H * ratio, MIN_W, MAX_W)
}

// 前缀和偏移数组：offsets[0] = 0，offsets[i] = offsets[i-1] + widths[i-1] + GAP，与输入等长
// 构建 O(n)，此后任意索引定位 O(1)
export function computeOffsets(widths: number[]): number[] {
  const offsets = new Array<number>(widths.length)
  let acc = 0
  for (let i = 0; i < widths.length; i++) {
    offsets[i] = acc
    acc += widths[i] + GAP
  }
  return offsets
}

// 内容总宽度：全部项宽之和 + GAP × (n-1)；n = 0 返回 0
export function computeTotalWidth(widths: number[]): number {
  if (widths.length === 0) return 0
  const sum = widths.reduce((total, w) => total + w, 0)
  return sum + GAP * (widths.length - 1)
}

// 可见区间（闭区间索引）：视口 [scrollLeft, scrollLeft + viewportWidth] 覆盖的项，两端各扩 overscan 项
// 约定：n = 0，或视口完全滚出内容范围（滚过末尾）时返回 { start: -1, end: -1 } 表示空区间
export function findVisibleRange(
  offsets: number[],
  widths: number[],
  scrollLeft: number,
  viewportWidth: number,
  overscan = OVERSCAN,
): { start: number; end: number } {
  const n = offsets.length
  if (n === 0) return { start: -1, end: -1 }
  const right = scrollLeft + viewportWidth

  // 二分找首个右边界（offset + width）> scrollLeft 的项 = 可见起点
  let lo = 0
  let hi = n - 1
  let first = -1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    if (offsets[mid] + widths[mid] > scrollLeft) {
      first = mid
      hi = mid - 1
    } else {
      lo = mid + 1
    }
  }
  // 所有项右边界都 ≤ scrollLeft：已滚过末尾，视口内无内容
  if (first === -1) return { start: -1, end: -1 }
  const start = clamp(first - overscan, 0, n - 1)

  // 二分找首个 offset ≥ right 的项（exclusive 终点），终点前一项即最后可见项，再前进 overscan
  let end = n
  lo = 0
  hi = n
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (offsets[mid] >= right) {
      end = mid
      hi = mid
    } else {
      lo = mid + 1
    }
  }
  return { start, end: clamp(end - 1 + overscan, 0, n - 1) }
}

// 自动居中的滚动目标：使第 i 项位于视口水平中点，clamp 到可滚动范围 [0, totalWidth - viewportWidth]
export function centerScrollLeft(
  offset: number,
  itemWidth: number,
  viewportWidth: number,
  totalWidth: number,
): number {
  const max = Math.max(0, totalWidth - viewportWidth)
  return clamp(offset - (viewportWidth - itemWidth) / 2, 0, max)
}
