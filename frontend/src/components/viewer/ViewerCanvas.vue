<script setup lang="ts">
// 通用画布：对整个内容区执行 zoom/pan transform（slot 注入帧内容，单图/双页复用，docs/ARCHITECTURE.md §4.5）
import { computed, onMounted, onUnmounted } from 'vue'
import type { UseViewerReturn } from '../../composables/useViewer'

const props = defineProps<{ viewer: UseViewerReturn }>()

// 内容 transform：translate + scale + rotate，origin 0 0 与 zoomAt 锚点数学一致
// （rotate 在 scale 内层，useViewer.applyCentered 的平移分支与其配套）
const contentStyle = computed(() => ({
  transform: `translate(${props.viewer.panX.value}px, ${props.viewer.panY.value}px) scale(${props.viewer.scale.value}) rotate(${props.viewer.rotation.value}deg)`,
}))

const cursor = computed(() => (props.viewer.dragging.value ? 'grabbing' : 'grab'))

// 滚轮缩放：以鼠标在容器内位置为锚点（clientX - rect，规避子元素 offsetX 坐标系差异）
function onWheel(e: WheelEvent) {
  e.preventDefault()
  const factor = e.deltaY < 0 ? 1.2 : 0.8
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  props.viewer.zoomAt(factor, e.clientX - rect.left, e.clientY - rect.top)
}

// 卸载时模板 ref 已被置空，onMounted 缓存元素用于解绑
let boundEl: HTMLElement | null = null
// 容器尺寸变化（图片↔全图、缩略图条显隐、浏览器全屏、窗口缩放）后重新 Fit；
// ResizeObserver 在布局完成后回调，天然规避 Fullscreen API 等异步切换的时序问题
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  const el = props.viewer.containerRef.value
  if (!el) return
  boundEl = el
  el.addEventListener('wheel', onWheel, { passive: false })
  props.viewer.bindPan(el)
  resizeObserver = new ResizeObserver(() => props.viewer.fit())
  resizeObserver.observe(el)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
  if (!boundEl) return
  boundEl.removeEventListener('wheel', onWheel)
  props.viewer.unbindPan(boundEl)
  boundEl = null
})
</script>

<template>
  <div
    :ref="viewer.containerRef"
    class="relative h-full w-full select-none overflow-hidden"
    :style="{ cursor }"
  >
    <div class="origin-top-left" :style="contentStyle">
      <slot />
    </div>
  </div>
</template>
