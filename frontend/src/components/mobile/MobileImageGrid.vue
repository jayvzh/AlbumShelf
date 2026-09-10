<script setup lang="ts">
// 移动端封面网格：minmax(110px,1fr) 自适应列数（375px ≈ 3 列）
// 卡片 = 方形缩略图 + 文件名 + 尺寸·大小；收藏角标 36px 热区（移动端无 hover，常显）
import { useFavoritesStore } from '../../stores/favorites'
import { buildThumbnailUrl } from '../../services/image.service'
import { formatFileSize } from '../../utils/format'
import type { ImageFile } from '../../types/file'

defineProps<{
  images: ImageFile[]
}>()

const emit = defineEmits<{ open: [index: number] }>()

const favorites = useFavoritesStore()

// 收藏/取消收藏：失败已回滚，静默（同桌面）
async function toggleFavorite(image: ImageFile) {
  try {
    await favorites.toggle(image.path, image)
  } catch {
    /* 忽略：favorites store 已回滚乐观状态 */
  }
}
</script>

<template>
  <div class="grid grid-cols-[repeat(auto-fill,minmax(110px,1fr))] gap-2 p-2">
    <div
      v-for="(image, index) in images"
      :key="image.path"
      class="relative cursor-pointer overflow-hidden rounded-lg border border-line bg-panel transition-colors active:bg-elevated"
      @click="emit('open', index)"
    >
      <img
        :src="buildThumbnailUrl(image, 300)"
        :alt="image.name"
        loading="lazy"
        decoding="async"
        class="aspect-square w-full bg-black/20 object-cover"
      />
      <!-- 收藏角标：36px 热区，压暗圆底保证深浅封面下可见 -->
      <button
        v-if="favorites.available"
        type="button"
        :title="favorites.has(image.path) ? '取消收藏' : '收藏'"
        class="absolute right-0.5 top-0.5 flex h-9 w-9 items-center justify-center rounded-full bg-black/35"
        @click.stop="toggleFavorite(image)"
      >
        <svg
          class="h-4 w-4"
          :class="favorites.has(image.path) ? 'text-red-400' : 'text-white/85'"
          :fill="favorites.has(image.path) ? 'currentColor' : 'none'"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          viewBox="0 0 24 24"
        >
          <path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z" />
        </svg>
      </button>
      <!-- 元信息：文件名 + 尺寸·大小 -->
      <div class="p-1.5">
        <p class="truncate text-xs text-ink">{{ image.name }}</p>
        <p class="mt-0.5 truncate text-[10px] leading-4 text-muted">
          {{ image.width !== null && image.height !== null ? `${image.width}×${image.height}` : '—' }} · {{ formatFileSize(image.size) }}
        </p>
      </div>
    </div>
  </div>
</template>
