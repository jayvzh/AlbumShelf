import { ref } from 'vue'

// 缩放范围
export const MIN_SCALE = 0.05
export const MAX_SCALE = 8

// 缩放状态：scale + 内容平移像素（panX/panY 语义为"内容平移像素"，配合 transform-origin: 0 0）
export function useZoom() {
  const scale = ref(1)
  const panX = ref(0)
  const panY = ref(0)

  function clampScale(value: number): number {
    return Math.min(Math.max(value, MIN_SCALE), MAX_SCALE)
  }

  // 以容器坐标 (cx, cy) 为锚点缩放：内容点在锚点处视觉不动
  // 推导：pan' = anchor - (anchor - pan) * (newScale / scale)
  function zoomAt(factor: number, cx: number, cy: number) {
    const newScale = clampScale(scale.value * factor)
    if (newScale === scale.value) return
    panX.value = cx - (cx - panX.value) * (newScale / scale.value)
    panY.value = cy - (cy - panY.value) * (newScale / scale.value)
    scale.value = newScale
  }

  return { scale, panX, panY, zoomAt }
}

export type UseZoomReturn = ReturnType<typeof useZoom>
