<script setup lang="ts">
// 图片元信息浮动面板：经 service 请求 /image/info（组件不直接 fetch）
// 双页帧传入两张图时上下分块展示「左页 / 右页」，各自独立请求与 loading/error 状态；
// 单图帧外观与原单面板一致（不显示左右页标签）
import { ref, watch } from 'vue'
import { getImageInfo } from '../../services/image.service'
import { formatFileSize } from '../../utils/format'
import type { ImageFile } from '../../types/file'
import type { ImageInfo as ImageInfoData } from '../../types/image'

const props = defineProps<{ images: ImageFile[] }>()

interface Block {
  image: ImageFile
  info: ImageInfoData | null
  loading: boolean
  error: string | null
}

const blocks = ref<Block[]>([])

// 帧守卫：快速翻帧时丢弃过期帧的全部响应
let seq = 0

watch(
  () => props.images.map((i) => i.path).join('|'),
  () => {
    const imgs = props.images
    const id = ++seq
    blocks.value = imgs.map((image) => ({ image, info: null, loading: true, error: null }))
    imgs.forEach(async (image, idx) => {
      try {
        const res = await getImageInfo(image.path)
        if (id !== seq) return
        blocks.value[idx] = { image, info: res, loading: false, error: null }
      } catch (e) {
        if (id !== seq) return
        blocks.value[idx] = {
          image,
          info: null,
          loading: false,
          error: e instanceof Error ? e.message : '加载图片信息失败',
        }
      }
    })
  },
  { immediate: true },
)
</script>

<template>
  <div
    v-if="blocks.length"
    class="absolute bottom-3 right-3 z-20 max-h-[calc(100%-2rem)] w-72 max-w-[90%] overflow-y-auto rounded-lg border border-line bg-panel/80 p-3 text-xs text-body"
  >
    <div
      v-for="(b, i) in blocks"
      :key="b.image.path"
      class="relative"
      :class="i > 0 ? 'mt-2 border-t border-line pt-2' : ''"
    >
      <!-- 双页帧右下角大号水印 L/R 区分左右页；负 z 沉于文字之下，文字可直接压在上面 -->
      <span
        v-if="blocks.length > 1"
        class="pointer-events-none absolute bottom-0 right-0 -z-10 select-none text-4xl font-light leading-none text-ink/10"
      >
        {{ i === 0 ? 'L' : 'R' }}
      </span>
      <p class="max-w-full truncate text-sm text-ink" :title="b.info?.name ?? b.image.name">
        {{ b.info?.name ?? b.image.name }}
      </p>
      <p v-if="b.loading" class="mt-2 text-faint">加载中…</p>
      <p v-else-if="b.error" class="mt-2 text-red-500">{{ b.error }}</p>
      <template v-else-if="b.info">
        <p class="mt-2">
          分辨率：{{ b.info.resolution.width !== null && b.info.resolution.height !== null ? `${b.info.resolution.width}×${b.info.resolution.height}` : '—' }}
        </p>
        <p>大小：{{ formatFileSize(b.info.size) }}</p>
        <p>修改时间：{{ new Date(b.info.modified_at).toLocaleString() }}</p>
      </template>
    </div>
  </div>
</template>
