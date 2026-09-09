import { ref } from 'vue'
import type { UseZoomReturn } from './useZoom'

// 拖拽平移：pointer events 统一处理鼠标/触摸（容器需 touch-action: none）
export function usePan(zoom: UseZoomReturn) {
  const dragging = ref(false)
  let startX = 0
  let startY = 0
  let startPanX = 0
  let startPanY = 0

  function onPointerDown(e: PointerEvent) {
    // 鼠标仅响应左键；触摸/笔不受限
    if (e.pointerType === 'mouse' && e.button !== 0) return
    dragging.value = true
    startX = e.clientX
    startY = e.clientY
    startPanX = zoom.panX.value
    startPanY = zoom.panY.value
    // 捕获指针：拖出容器后仍持续接收 move/up
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }

  function onPointerMove(e: PointerEvent) {
    if (!dragging.value) return
    zoom.panX.value = startPanX + (e.clientX - startX)
    zoom.panY.value = startPanY + (e.clientY - startY)
  }

  function endDrag() {
    dragging.value = false
  }

  // 绑定到容器元素（ViewerCanvas onMounted 时调用一次）
  function bind(el: HTMLElement) {
    el.addEventListener('pointerdown', onPointerDown)
    el.addEventListener('pointermove', onPointerMove)
    el.addEventListener('pointerup', endDrag)
    el.addEventListener('pointercancel', endDrag)
    el.addEventListener('pointerleave', endDrag)
    el.style.touchAction = 'none'
  }

  function unbind(el: HTMLElement) {
    el.removeEventListener('pointerdown', onPointerDown)
    el.removeEventListener('pointermove', onPointerMove)
    el.removeEventListener('pointerup', endDrag)
    el.removeEventListener('pointercancel', endDrag)
    el.removeEventListener('pointerleave', endDrag)
  }

  return { dragging, bind, unbind }
}

export type UsePanReturn = ReturnType<typeof usePan>
