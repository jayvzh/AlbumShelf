<script setup lang="ts">
// Filmstrip 单个缩略图项：固定高度 THUMB_H、宽度由父级按宽高比算好传入（等高不等宽）
import { ref } from 'vue'
import type { ImageFile } from '../../types/file'
import { buildThumbnailUrl, withCacheBust } from '../../services/image.service'
import { THUMB_H } from '../../utils/filmstrip'

const props = defineProps<{
  image: ImageFile
  // 当前查看中的项：高亮边框（主题 accent-focus，深浅主题各配亮/深琥珀，见 styles/main.css）
  active: boolean
  // 显示宽度（px），由 Filmstrip 布局计算得出
  width: number
}>()

// 点击整项通知父级；父级持有索引，故不传参
const emit = defineEmits<{ select: [] }>()

// 加载失败：以 cache-bust URL 一次性重试（绕过可能已损坏的浏览器缓存条目），仍失败保持原样
const bustedSrc = ref<string | null>(null)
function onImgError() {
  if (bustedSrc.value) return
  bustedSrc.value = withCacheBust(buildThumbnailUrl(props.image))
}
</script>

<template>
  <button
    type="button"
    class="block shrink-0 overflow-hidden rounded-md border bg-panel transition"
    :class="active ? 'border-accent-focus ring-2 ring-accent-focus/50' : 'border-line hover:border-line-hover'"
    :style="{ height: `${THUMB_H}px`, width: `${width}px` }"
    @click="emit('select')"
  >
    <!-- 缩略图懒加载：进入视口才请求；object-cover 填充、禁止拖拽；加载前露面板色占位底 -->
    <img
      :src="bustedSrc ?? buildThumbnailUrl(image)"
      :alt="image.name"
      loading="lazy"
      draggable="false"
      class="h-full w-full select-none object-cover"
      @error="onImgError"
    />
  </button>
</template>
