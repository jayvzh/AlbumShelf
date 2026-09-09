<script setup lang="ts">
// Filmstrip 单个缩略图项：固定高度 THUMB_H、宽度由父级按宽高比算好传入（等高不等宽）
import type { ImageFile } from '../../types/file'
import { buildThumbnailUrl } from '../../services/image.service'
import { THUMB_H } from '../../utils/filmstrip'

defineProps<{
  image: ImageFile
  // 当前查看中的项：高亮边框（amber accent，风格对齐 ViewerToolbar）
  active: boolean
  // 显示宽度（px），由 Filmstrip 布局计算得出
  width: number
}>()

// 点击整项通知父级；父级持有索引，故不传参
const emit = defineEmits<{ select: [] }>()
</script>

<template>
  <button
    type="button"
    class="block shrink-0 overflow-hidden rounded-md border bg-panel transition-colors"
    :class="active ? 'border-amber-400/70' : 'border-line hover:border-line-hover'"
    :style="{ height: `${THUMB_H}px`, width: `${width}px` }"
    @click="emit('select')"
  >
    <!-- 缩略图懒加载：进入视口才请求；object-cover 填充、禁止拖拽；加载前露面板色占位底 -->
    <img
      :src="buildThumbnailUrl(image)"
      :alt="image.name"
      loading="lazy"
      draggable="false"
      class="h-full w-full select-none object-cover"
    />
  </button>
</template>
