<script setup lang="ts">
// 底部缩略图条：等高不等宽布局 + 前缀和偏移 + 二分可见区间的虚拟渲染
// （布局数学在 utils/filmstrip.ts，状态接线在 composables/useFilmstrip.ts，见 docs/ARCHITECTURE.md §4.6）
import { computed, nextTick, watch } from 'vue'
import type { ImageFile } from '../../types/file'
import { useFilmstrip } from '../../composables/useFilmstrip'
import { THUMB_H } from '../../utils/filmstrip'
import ThumbnailItem from './ThumbnailItem.vue'

const props = defineProps<{
  images: ImageFile[]
  // 当前帧包含的所有图片索引（双页帧两图同时高亮）；查看器未打开时为空数组
  activeIndexes: number[]
}>()

const emit = defineEmits<{ select: [index: number] }>()

// 布局（宽度/偏移/总宽）、容器测量与可见区间全部来自 useFilmstrip
const { containerRef, layout, visibleRange, onScroll, scrollToCenter } = useFilmstrip(
  computed(() => props.images),
)

// 仅渲染可见区间（± overscan）内的项：携带全局索引、绝对偏移与项宽
// 可见数量由容器宽度与各项宽度动态决定，禁止硬编码（docs/UI_DESIGN.md §6）
const visibleItems = computed(() => {
  const { start, end } = visibleRange.value
  if (start < 0 || end < 0) return []
  const items: { image: ImageFile; index: number; offset: number; width: number }[] = []
  for (let i = start; i <= end; i++) {
    items.push({
      image: props.images[i],
      index: i,
      offset: layout.value.offsets[i],
      width: layout.value.widths[i],
    })
  }
  return items
})

// 打开查看器 / 翻帧联动：当前帧首图自动平滑滚动到视口中部（双页两图相邻，首图居中即基本可见）
watch(
  () => props.activeIndexes.join(','),
  (key) => {
    if (!key) return
    const first = Math.min(...props.activeIndexes)
    // 等 DOM 更新（含虚拟渲染切片变化）后再滚动
    nextTick(() => scrollToCenter(first))
  },
)

// 垂直滚轮转横向滚动（docs/UI_DESIGN.md §5：Filmstrip 滚轮横向滚动缩略图条）
function onWheel(e: WheelEvent) {
  if (containerRef.value) containerRef.value.scrollLeft += e.deltaY
}
</script>

<template>
  <!-- 目录为空不渲染整条 -->
  <div
    v-if="images.length > 0"
    ref="containerRef"
    class="filmstrip-scroll overflow-x-auto overflow-y-hidden border-t border-line bg-base py-2"
    @scroll="onScroll"
    @wheel.prevent="onWheel"
  >
    <!-- 占位层：撑起真实内容总宽，内部项按绝对偏移定位，滚动条长度与实际一致 -->
    <div class="relative" :style="{ width: `${layout.totalWidth}px`, height: `${THUMB_H}px` }">
      <ThumbnailItem
        v-for="item in visibleItems"
        :key="item.image.path"
        :image="item.image"
        :active="activeIndexes.includes(item.index)"
        :width="item.width"
        class="absolute left-0 top-0"
        :style="{ transform: `translateX(${item.offset}px)` }"
        @select="emit('select', item.index)"
      />
    </div>
  </div>
</template>

<style scoped>
/* 细滚动条：滑块颜色随主题（--app-scrollbar），避免原生粗滚动条突兀（纯 CSS，不引插件） */
.filmstrip-scroll {
  scrollbar-width: thin;
  scrollbar-color: var(--app-scrollbar) transparent;
}
.filmstrip-scroll::-webkit-scrollbar {
  height: 6px;
}
.filmstrip-scroll::-webkit-scrollbar-thumb {
  border-radius: 3px;
  background: var(--app-scrollbar);
}
.filmstrip-scroll::-webkit-scrollbar-track {
  background: transparent;
}
</style>
