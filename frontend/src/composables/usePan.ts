import { ref } from 'vue'
import type { UseZoomReturn } from './useZoom'

// tap 手势信息（容器相对坐标 + 触点类型）
export interface TapDetail {
  x: number
  y: number
  width: number
  height: number
  pointerType: string
}

export interface UsePanOptions {
  // tap 手势回调（仅 touch/pen 触发；鼠标不触发）
  onTap?: (detail: TapDetail) => void
}

// tap 判定阈值：位移 <10px 且时长 <300ms
const TAP_MAX_DISTANCE = 10
const TAP_MAX_DURATION = 300

interface PointerState {
  x: number
  y: number
}

// 平移 + 双指缩放手势（pointer events 统一鼠标/触摸/笔，容器需 touch-action: none）
// - 单指/鼠标：拖拽平移（基线式）
// - 双指 pinch：锚=两指中点、factor=距离比，经 zoomAt 锚点缩放；中点位移整体平移
// - 指数切换（双指→单指、单指→双指）重建基线，无跳变
// - tap（可选）：单指按下后位移与时长均在阈值内抬起 → onTap 回调（透出容器相对坐标）
export function usePan(zoom: UseZoomReturn, options: UsePanOptions = {}) {
  const dragging = ref(false)

  // 活动指针集合（多指追踪）
  const pointers = new Map<number, PointerState>()

  // 容器视口矩形缓存（pointerdown 时刷新；pinch 锚点/tap 坐标换算用）
  let rect = { left: 0, top: 0, width: 0, height: 0 }

  // 单指拖拽基线
  let startX = 0
  let startY = 0
  let startPanX = 0
  let startPanY = 0

  // pinch 上一帧状态
  let pinchDist = 0
  let pinchMidX = 0
  let pinchMidY = 0

  // tap 判定基线
  let tapActive = false
  let tapPointerId: number | null = null
  let tapStartX = 0
  let tapStartY = 0
  let tapStartTime = 0

  function updateRect(el: HTMLElement) {
    const r = el.getBoundingClientRect()
    rect = { left: r.left, top: r.top, width: r.width, height: r.height }
  }

  function onPointerDown(e: PointerEvent) {
    // 鼠标仅响应左键；触摸/笔不受限
    if (e.pointerType === 'mouse' && e.button !== 0) return
    const el = e.currentTarget as HTMLElement
    updateRect(el)
    pointers.set(e.pointerId, { x: e.clientX, y: e.clientY })
    // 捕获指针：拖出容器后仍持续接收 move/up（指针已释放或合成事件时可能抛错，忽略即可）
    try {
      el.setPointerCapture(e.pointerId)
    } catch {
      // 忽略
    }

    if (pointers.size === 1) {
      dragging.value = true
      startX = e.clientX
      startY = e.clientY
      startPanX = zoom.panX.value
      startPanY = zoom.panY.value
      // tap 基线（仅 touch/pen；鼠标不触发 tap）
      if (e.pointerType !== 'mouse') {
        tapActive = true
        tapPointerId = e.pointerId
        tapStartX = e.clientX
        tapStartY = e.clientY
        tapStartTime = performance.now()
      }
    } else if (pointers.size === 2) {
      // 进入 pinch：以当前双指状态重建基线（第二指落下无跳变），多指取消 tap
      const [a, b] = [...pointers.values()]
      pinchDist = Math.hypot(a.x - b.x, a.y - b.y)
      pinchMidX = (a.x + b.x) / 2
      pinchMidY = (a.y + b.y) / 2
      tapActive = false
    }
  }

  function onPointerMove(e: PointerEvent) {
    const p = pointers.get(e.pointerId)
    if (!p) return
    p.x = e.clientX
    p.y = e.clientY

    // tap 位移超阈值取消
    if (tapActive && e.pointerId === tapPointerId) {
      if (Math.hypot(e.clientX - tapStartX, e.clientY - tapStartY) > TAP_MAX_DISTANCE) {
        tapActive = false
      }
    }

    if (pointers.size >= 2) {
      // 双指 pinch：距离比缩放（锚=中点，容器相对坐标）+ 中点位移平移
      const [a, b] = [...pointers.values()]
      const dist = Math.hypot(a.x - b.x, a.y - b.y)
      const midX = (a.x + b.x) / 2
      const midY = (a.y + b.y) / 2
      if (pinchDist > 0 && dist > 0) {
        zoom.zoomAt(dist / pinchDist, midX - rect.left, midY - rect.top)
      }
      zoom.panX.value += midX - pinchMidX
      zoom.panY.value += midY - pinchMidY
      pinchDist = dist
      pinchMidX = midX
      pinchMidY = midY
    } else if (dragging.value) {
      // 单指拖拽平移
      zoom.panX.value = startPanX + (e.clientX - startX)
      zoom.panY.value = startPanY + (e.clientY - startY)
    }
  }

  // 指针抬起/取消：删除指针并按剩余指数重建基线
  function onPointerEnd(e: PointerEvent, allowTap: boolean) {
    const wasCount = pointers.size
    if (!pointers.delete(e.pointerId)) return

    // tap 判定：仅 touch/pen、目标指针、全程单指、时长在阈值内（位移阈值在 move 中判定）
    if (
      allowTap &&
      tapActive &&
      e.pointerId === tapPointerId &&
      wasCount === 1 &&
      performance.now() - tapStartTime <= TAP_MAX_DURATION
    ) {
      options.onTap?.({
        x: e.clientX - rect.left,
        y: e.clientY - rect.top,
        width: rect.width,
        height: rect.height,
        pointerType: e.pointerType,
      })
    }
    tapActive = false
    tapPointerId = null

    if (pointers.size === 1) {
      // 双指退单指：以剩余指重建拖拽基线（无跳变）
      const [p] = [...pointers.values()]
      dragging.value = true
      startX = p.x
      startY = p.y
      startPanX = zoom.panX.value
      startPanY = zoom.panY.value
    } else if (pointers.size === 0) {
      dragging.value = false
    }
  }

  // 指针抬起：允许 tap 判定
  function onPointerUp(e: PointerEvent) {
    onPointerEnd(e, true)
  }

  // 指针被系统取消（如浏览器接管手势）：不做 tap 判定
  function onPointerCancel(e: PointerEvent) {
    onPointerEnd(e, false)
  }

  // 鼠标移出容器（未捕获场景）：终止拖拽。触摸/笔有 capture 且与 up/cancel 成对，忽略
  function onPointerLeave(e: PointerEvent) {
    if (e.pointerType !== 'mouse') return
    pointers.clear()
    tapActive = false
    tapPointerId = null
    dragging.value = false
  }

  // 绑定到容器元素（ViewerCanvas / MobileReader onMounted 时调用一次）
  function bind(el: HTMLElement) {
    el.addEventListener('pointerdown', onPointerDown)
    el.addEventListener('pointermove', onPointerMove)
    el.addEventListener('pointerup', onPointerUp)
    el.addEventListener('pointercancel', onPointerCancel)
    el.addEventListener('pointerleave', onPointerLeave)
    el.style.touchAction = 'none'
  }

  function unbind(el: HTMLElement) {
    el.removeEventListener('pointerdown', onPointerDown)
    el.removeEventListener('pointermove', onPointerMove)
    el.removeEventListener('pointerup', onPointerUp)
    el.removeEventListener('pointercancel', onPointerCancel)
    el.removeEventListener('pointerleave', onPointerLeave)
  }

  return { dragging, bind, unbind }
}

export type UsePanReturn = ReturnType<typeof usePan>
