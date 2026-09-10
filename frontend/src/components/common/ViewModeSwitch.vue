<script setup lang="ts">
// 视图模式三段式切换：文件网格 / 嵌入大图（含缩略图条）/ 全屏大图（缩略图条可显隐）
// 直接读写 viewerStore（同 SortMenu 直接用 store 的既有模式），AppHeader 与 ViewerToolbar 复用
// compact 模式：顶栏窄宽时文案压缩为单字（文/图/全），保留 title 完整说明
import { useViewerStore } from '../../stores/viewer'
import type { ViewMode } from '../../stores/viewer'

withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })

const viewerStore = useViewerStore()

const MODES: { value: ViewMode; label: string; short: string }[] = [
  { value: 'file', label: '文件', short: '文' },
  { value: 'image', label: '图片', short: '图' },
  { value: 'full', label: '全图', short: '全' },
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
      :title="mode.label"
      class="px-2"
      :class="segClass(viewerStore.mode === mode.value)"
      @click="viewerStore.setMode(mode.value)"
    >
      {{ compact ? mode.short : mode.label }}
    </button>
  </div>
</template>
