<script setup lang="ts">
// 视图模式三段式切换：文件网格 / 嵌入大图（含缩略图条）/ 全屏大图（缩略图条可显隐）
// 直接读写 viewerStore（同 SortMenu 直接用 store 的既有模式），AppHeader 与 ViewerToolbar 复用
import { useViewerStore } from '../../stores/viewer'
import type { ViewMode } from '../../stores/viewer'

const viewerStore = useViewerStore()

const MODES: { value: ViewMode; label: string }[] = [
  { value: 'file', label: '文件' },
  { value: 'image', label: '图片' },
  { value: 'full', label: '全图' },
]

// 分段按钮样式：选中态仅左右边框 + 提亮文字（弱化色块，避免 amber 过于显眼）；
// 未选中态以透明 border-x 占位，避免切换时内容宽度跳动（高度由外层 items-stretch 撑满）
function segClass(active: boolean): string {
  return active
    ? 'rounded-[3px] border-x border-line-strong bg-elevated text-ink transition-colors'
    : 'rounded-[3px] border-x border-transparent text-body transition-colors hover:bg-elevated'
}
</script>

<template>
  <div class="flex h-8 shrink-0 items-stretch gap-0.5 rounded border border-line-strong bg-panel p-0.5 text-sm">
    <button
      v-for="mode in MODES"
      :key="mode.value"
      type="button"
      class="px-2"
      :class="segClass(viewerStore.mode === mode.value)"
      @click="viewerStore.setMode(mode.value)"
    >
      {{ mode.label }}
    </button>
  </div>
</template>
