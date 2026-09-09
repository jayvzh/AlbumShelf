<script setup lang="ts">
// 帧内容渲染：单页/双页统一等高布局（UniformHeight，垂直居中，SPREAD_ENGINE.md §3.4 / UI_DESIGN.md §3）
// 尺寸链路：每张 img onload 上报 natural 尺寸 → 全部就绪后按最大高 baseH 等比缩放，
// 计算帧宽 = Σ(w_i·baseH/h_i) + GAP·(n−1)，emit('layout') 供 ImageViewer setContentSize + Fit
import { ref, watch } from 'vue'
import type { ImageFile } from '../../types/file'
import type { SpreadFrame } from '../../types/spread'
import { SPREAD_FRAME_GAP } from '../../utils/spread'
import { buildImageUrl } from '../../services/image.service'

const props = defineProps<{
  frame: SpreadFrame
  // 加载变体：preview 预览图 / original 原图
  variant: 'preview' | 'original'
}>()

const emit = defineEmits<{
  // 帧布局尺寸就绪（帧内全部图片尺寸已知）
  layout: [size: { width: number; height: number }]
  // 单张图片加载失败（帧内其余图片继续渲染）
  imageError: [image: ImageFile]
}>()

// 每张图的自然尺寸，按 frame.images 下标追踪
const naturalSizes = ref<Array<{ width: number; height: number } | null>>([])
// 等高布局结果；null 表示尚未就绪（img 先以自然尺寸显示，就绪后统一收缩到 baseH）
const frameSize = ref<{ width: number; height: number } | null>(null)

// 换帧 / 切换预览原图：重置尺寸追踪，等待重新 onload
watch(
  () => [props.frame, props.variant] as const,
  () => {
    naturalSizes.value = props.frame.images.map(() => null)
    frameSize.value = null
  },
  { immediate: true },
)

// 登记一张图尺寸；全部就绪后一次性计算等高布局并上报
function settleSize(index: number, width: number, height: number) {
  if (naturalSizes.value[index]) return
  naturalSizes.value[index] = { width, height }
  if (naturalSizes.value.some((s) => s === null)) return
  const baseH = Math.max(...naturalSizes.value.map((s) => s!.height))
  const fitWidth =
    naturalSizes.value.reduce((sum, s) => sum + (s!.width * baseH) / s!.height, 0) +
    SPREAD_FRAME_GAP * (props.frame.images.length - 1)
  frameSize.value = { width: fitWidth, height: baseH }
  emit('layout', frameSize.value)
}

function onImgLoad(index: number, e: Event) {
  const img = e.target as HTMLImageElement
  settleSize(index, img.naturalWidth, img.naturalHeight)
}

// 加载失败：用目录元数据兜底（异常占位比 400×600），保证 layout 链不断
function onImgError(index: number) {
  const image = props.frame.images[index]
  if (!image) return
  emit('imageError', image)
  settleSize(index, image.width ?? 400, image.height ?? 600)
}
</script>

<template>
  <div class="flex select-none items-center" :style="{ gap: `${SPREAD_FRAME_GAP}px` }">
    <img
      v-for="(image, i) in frame.images"
      :key="image.path"
      :src="buildImageUrl(image, variant)"
      :alt="image.name"
      class="block max-w-none"
      draggable="false"
      :style="frameSize ? { height: `${frameSize.height}px` } : undefined"
      @load="onImgLoad(i, $event)"
      @error="onImgError(i)"
    />
  </div>
</template>
