<script setup lang="ts">
// 视图模式图标式切换（主流文件管理器风格）：列表图标 = 文件网格 / 图片预览图标 = 嵌入大图
// 全图（覆盖浏览）入口在查看器右上角最大化按钮；全图模式下图片图标仍为激活态，点击即还原
// 直接读写 viewerStore（同 SortMenu 直接用 store 的既有模式）
import { useViewerStore } from '../../stores/viewer'

const viewerStore = useViewerStore()

// 分段按钮样式：激活态提亮底色，未激活态透明占位（与原文字分段风格一致，纯图标无宽度跳动）
function segClass(active: boolean): string {
  return active
    ? 'rounded-[3px] bg-elevated text-ink transition-colors'
    : 'rounded-[3px] text-body transition-colors hover:bg-elevated'
}
</script>

<template>
  <div class="flex h-8 shrink-0 items-stretch gap-0.5 rounded border border-line-strong bg-panel p-0.5">
    <button
      type="button"
      title="文件列表"
      class="flex w-8 items-center justify-center"
      :class="segClass(viewerStore.mode === 'file')"
      @click="viewerStore.setMode('file')"
    >
      <svg
        class="h-[18px] w-[18px]"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01" />
      </svg>
    </button>
    <button
      type="button"
      title="图片预览"
      class="flex w-8 items-center justify-center"
      :class="segClass(viewerStore.mode !== 'file')"
      @click="viewerStore.setMode('image')"
    >
      <svg
        class="h-[18px] w-[18px]"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <rect x="3" y="3" width="18" height="18" rx="2" />
        <circle cx="9" cy="9" r="2" />
        <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21" />
      </svg>
    </button>
  </div>
</template>
