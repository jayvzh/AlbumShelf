import { ref, onUnmounted } from 'vue'
import { useZoom } from './useZoom'
import { usePan, type TapDetail } from './usePan'

// 顺时针旋转角度（度）
export type Rotation = 0 | 90 | 180 | 270

export interface UseViewerOptions {
  // tap 手势回调（透传 usePan，仅 touch/pen 触发；不传则无 tap 行为）
  onTap?: (detail: TapDetail) => void
}

// Viewer UI 运行态组合：Zoom / Pan / Fit / 100% / Rotate / Fullscreen（不进 Pinia store）
export function useViewer(options: UseViewerOptions = {}) {
  const zoom = useZoom()
  const pan = usePan(zoom, { onTap: options.onTap })

  // 画布容器（ViewerCanvas 外层 div，经 :ref 注入）
  const containerRef = ref<HTMLElement | null>(null)
  const isFullscreen = ref(false)

  // 内容原始尺寸（外部上报：单图为 img natural 尺寸，双页帧为等高布局尺寸）
  const contentWidth = ref(0)
  const contentHeight = ref(0)

  // 顺时针旋转角度（运行态，换帧时由 ImageViewer 调 resetRotation 归零）
  const rotation = ref<Rotation>(0)

  // 左右镜像（运行态，仅 CSS 翻转显示不改源文件；页面刷新即重置，换帧保留）
  const mirrored = ref(false)

  function setContentSize(width: number, height: number) {
    contentWidth.value = width
    contentHeight.value = height
  }

  // 按当前 rotation 与 scale 计算居中平移。
  // transform = translate(tx,ty) scale(s) rotate(r)（origin 0 0），CSS rotate 正角顺时针：90° 时 (x,y)→(−y,x)。
  // 设缩放后未旋转尺寸 w=W·s、h=H·s，各角度旋转后内容包围盒与居中平移：
  //   0°:   [0,w]×[0,h]   → tx=(cw−w)/2, ty=(ch−h)/2
  //   90°:  [−h,0]×[0,w]  → tx=(cw+h)/2, ty=(ch−w)/2
  //   180°: [−w,0]×[−h,0] → tx=(cw+w)/2, ty=(ch+h)/2
  //   270°: [0,h]×[−w,0]  → tx=(cw−h)/2, ty=(ch+w)/2
  function applyCentered() {
    const el = containerRef.value
    if (!el) return
    const w = contentWidth.value * zoom.scale.value
    const h = contentHeight.value * zoom.scale.value
    const cw = el.clientWidth
    const ch = el.clientHeight
    switch (rotation.value) {
      case 90:
        zoom.panX.value = (cw + h) / 2
        zoom.panY.value = (ch - w) / 2
        break
      case 180:
        zoom.panX.value = (cw + w) / 2
        zoom.panY.value = (ch + h) / 2
        break
      case 270:
        zoom.panX.value = (cw - h) / 2
        zoom.panY.value = (ch + w) / 2
        break
      default:
        zoom.panX.value = (cw - w) / 2
        zoom.panY.value = (ch - h) / 2
    }
  }

  // Fit：等比缩放至刚好放进容器并居中（旋转 90°/270° 时有效投影宽高互换；防 0 除：尺寸未知跳过）
  function fit() {
    const el = containerRef.value
    if (!el || contentWidth.value <= 0 || contentHeight.value <= 0) return
    const swap = rotation.value === 90 || rotation.value === 270
    const effW = swap ? contentHeight.value : contentWidth.value
    const effH = swap ? contentWidth.value : contentHeight.value
    zoom.scale.value = Math.min(el.clientWidth / effW, el.clientHeight / effH)
    applyCentered()
  }

  // 100%：scale=1，保持居中
  function setScale100() {
    if (!containerRef.value) return
    zoom.scale.value = 1
    applyCentered()
  }

  // 顺时针旋转 90° 并重新 Fit
  function rotate() {
    rotation.value = ((rotation.value + 90) % 360) as Rotation
    fit()
  }

  // 切换左右镜像（显示翻转，bbox 不变，无需重算 Fit）
  function toggleMirror() {
    mirrored.value = !mirrored.value
  }

  // 重置旋转并重新 Fit（换帧时调用；未旋转时跳过避免多余布局）
  function resetRotation() {
    if (rotation.value === 0) return
    rotation.value = 0
    fit()
  }

  // 重置 = Fit
  function reset() {
    fit()
  }

  // 全屏元素读取 / 状态变更事件：兼容 iOS Safari 等仅支持 webkit 前缀的内核
  function getFullscreenElement(): Element | null {
    return document.fullscreenElement
      ?? (document as unknown as { webkitFullscreenElement?: Element }).webkitFullscreenElement
      ?? null
  }

  // 网页全屏（等效 F11）：对 documentElement 请求，页面布局不变（顶栏/目录树/缩略图条保留），
  // 仅浏览器窗口全屏；全图模式覆盖层本就 fixed inset-0 铺满，同样适用。
  // 返回是否成功进入/保持全屏（请求被拒绝时为 false，调用方据此回滚联动状态）
  async function toggleFullscreen(): Promise<boolean> {
    try {
      if (getFullscreenElement()) {
        if (document.exitFullscreen) {
          await document.exitFullscreen()
        } else {
          await (document as unknown as { webkitExitFullscreen?: () => void }).webkitExitFullscreen?.()
        }
        return false
      }
      const el = document.documentElement as HTMLElement & {
        webkitRequestFullscreen?: () => Promise<void> | void
      }
      if (el.requestFullscreen) {
        await el.requestFullscreen()
      } else if (el.webkitRequestFullscreen) {
        await el.webkitRequestFullscreen()
      } else {
        // 当前环境完全不支持全屏 API
        return false
      }
      return true
    } catch {
      // 全屏请求失败（如非用户手势）静默忽略
      return false
    }
  }

  function onFullscreenChange() {
    isFullscreen.value = getFullscreenElement() != null
  }

  document.addEventListener('fullscreenchange', onFullscreenChange)
  // iOS Safari / 旧 WebKit 仅派发前缀事件
  document.addEventListener('webkitfullscreenchange', onFullscreenChange)
  onUnmounted(() => {
    document.removeEventListener('fullscreenchange', onFullscreenChange)
    document.removeEventListener('webkitfullscreenchange', onFullscreenChange)
  })

  return {
    scale: zoom.scale,
    panX: zoom.panX,
    panY: zoom.panY,
    // 锚点缩放：rotate ≠ 0 时锚点坐标含旋转偏移，为近似缩放（接受，docs/UI_DESIGN.md rotation 属运行态）
    zoomAt: zoom.zoomAt,
    dragging: pan.dragging,
    bindPan: pan.bind,
    unbindPan: pan.unbind,
    containerRef,
    isFullscreen,
    contentWidth,
    contentHeight,
    rotation,
    mirrored,
    setContentSize,
    fit,
    setScale100,
    rotate,
    toggleMirror,
    resetRotation,
    reset,
    toggleFullscreen,
  }
}

export type UseViewerReturn = ReturnType<typeof useViewer>
