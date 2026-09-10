<script setup lang="ts">
// 侧栏：目录树容器；右缘可拖拽调宽（min/max 限制），localStorage 持久化（键 albumshelf:sidebar-width）
import { ref } from 'vue'
import FolderTree from '../folder/FolderTree.vue'

const WIDTH_KEY = 'albumshelf:sidebar-width'
const MIN_WIDTH = 180
const MAX_WIDTH = 480
const DEFAULT_WIDTH = 256

function clampWidth(w: number): number {
  return Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, Math.round(w)))
}

// 初始宽度：读 localStorage，缺省/非法/越界值回退默认
const width = ref(DEFAULT_WIDTH)
try {
  const raw = Number(localStorage.getItem(WIDTH_KEY))
  if (Number.isFinite(raw) && raw > 0) width.value = clampWidth(raw)
} catch {
  /* 存储不可用时本次会话用默认宽度 */
}

// 拖拽状态：pointer capture 保证移出热区后仍持续追踪
const dragging = ref(false)

function onDragStart(e: PointerEvent) {
  dragging.value = true
  ;(e.currentTarget as Element).setPointerCapture(e.pointerId)
  e.preventDefault()
  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'col-resize'
}

function onDragMove(e: PointerEvent) {
  // 侧栏贴视口左缘，clientX 即目标宽度
  if (dragging.value) width.value = clampWidth(e.clientX)
}

function onDragEnd() {
  if (!dragging.value) return
  dragging.value = false
  document.body.style.userSelect = ''
  document.body.style.cursor = ''
  try {
    localStorage.setItem(WIDTH_KEY, String(width.value))
  } catch {
    /* 存储不可用时仅本次会话生效 */
  }
}
</script>

<template>
  <!-- 外层不滚动：拖拽手柄钉在右缘，不随目录树滚动 -->
  <div class="relative shrink-0" :style="{ width: `${width}px` }">
    <aside class="h-full overflow-y-auto border-r border-line bg-base p-2">
      <FolderTree />
    </aside>
    <!-- 拖拽手柄：右缘 6px 热区，hover/拖拽时高亮 -->
    <div
      class="absolute inset-y-0 right-0 w-1.5 cursor-col-resize touch-none"
      :class="dragging ? 'bg-accent-text/40' : 'hover:bg-line-strong'"
      @pointerdown="onDragStart"
      @pointermove="onDragMove"
      @pointerup="onDragEnd"
      @pointercancel="onDragEnd"
    />
  </div>
</template>
