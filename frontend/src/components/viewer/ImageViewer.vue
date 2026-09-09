<script setup lang="ts">
// 大图浏览（单页/双页帧）：单实例双形态——图片模式原地嵌入主内容区（Teleport disabled），
// 全图模式 Teleport 全屏覆盖；组合 Toolbar / Canvas / SpreadFrame / Loading / Info / Filmstrip
// 状态链路：键盘/按钮 → action（store + viewer composable），不直接操作 DOM
import { computed, onUnmounted, ref, watch } from 'vue'
import { useViewerStore } from '../../stores/viewer'
import { useSpreadStore } from '../../stores/spread'
import { useFavoritesStore } from '../../stores/favorites'
import { useViewer } from '../../composables/useViewer'
import { useKeyboard } from '../../composables/useKeyboard'
import { buildImageUrl } from '../../services/image.service'
import type { ImageFile } from '../../types/file'
import ViewerToolbar from './ViewerToolbar.vue'
import ViewerCanvas from './ViewerCanvas.vue'
import SpreadFrame from './SpreadFrame.vue'
import LoadingOverlay from './LoadingOverlay.vue'
import ImageInfo from './ImageInfo.vue'
import Filmstrip from '../filmstrip/Filmstrip.vue'
import AppButton from '../common/AppButton.vue'

const viewerStore = useViewerStore()
const spreadStore = useSpreadStore()
const favorites = useFavoritesStore()
const viewer = useViewer()

const showInfo = ref(false)
const loading = ref(false)
const loadError = ref(false)

const currentFrame = computed(() => viewerStore.currentFrame)
const scale = computed(() => viewer.scale.value)
const isFullscreen = computed(() => viewer.isFullscreen.value)

// 根容器双形态：全图模式全屏覆盖；图片模式填满主内容区（relative 锚定 absolute 子元素）。
// 注意 fixed 与 relative 不能同挂一个元素（Tailwind 中 .relative 排序在后会覆盖 .fixed，导致覆盖层跌回文档流）
const rootClass = computed(() =>
  viewerStore.mode === 'full'
    ? 'fixed inset-0 z-50 bg-base'
    : 'relative h-full w-full bg-base',
)

// 页码标签：单图帧 "N / total"，双图帧 "N–M / total"（翻页按帧推进，docs/UI_DESIGN.md §3）
const pageLabel = computed(() => {
  const total = viewerStore.images.length
  const indexes = viewerStore.activeImageIndexes
  if (indexes.length === 0) return `- / ${total}`
  if (indexes.length === 1) return `${indexes[0] + 1} / ${total}`
  return `${Math.min(...indexes) + 1}–${Math.max(...indexes) + 1} / ${total}`
})

// 收藏/取消收藏当前帧第 index 张图（视觉左→右即 images 下标序）；失败已回滚，静默不打断阅读
async function toggleFrameFavorite(index: number) {
  const image = currentFrame.value?.images[index]
  if (!image || !favorites.available) return
  try {
    await favorites.toggle(image.path, image)
  } catch {
    /* 忽略：favorites store 已回滚乐观状态 */
  }
}

// 键盘 → action：翻页走 store，缩放/旋转/全屏走 viewer，Esc 逐级返回（全图→图片→文件）；
// S 收藏当前图（双页帧收藏视觉左图，即 images[0]）
const keyboard = useKeyboard({
  previous: () => viewerStore.previous(),
  next: () => viewerStore.next(),
  toggleFullscreen: () => viewer.toggleFullscreen(),
  fit: () => viewer.fit(),
  setScale100: () => viewer.setScale100(),
  rotate: () => viewer.rotate(),
  close: () => viewerStore.stepBack(),
  toggleFavorite: () => toggleFrameFavorite(0),
})

// immediate 覆盖持久化模式刷新首屏（isOpen 初始即 true 的场景）
watch(
  () => viewerStore.isOpen,
  (open) => (open ? keyboard.bind() : keyboard.unbind()),
  { immediate: true },
)
// 卸载时必须解绑：onKeyDown 是 useKeyboard 每次调用创建的闭包（引用不同），
// HMR 或 v-if/v-else 重挂后旧监听残留会导致一次 Esc 触发两次 stepBack（全图→文件跳级）
onUnmounted(() => keyboard.unbind())

// 预载下一帧全部图片（preview），提升翻帧流畅度（不预载原图）
function preloadNextFrame() {
  const next = spreadStore.frames[viewerStore.currentFrameIndex + 1]
  if (!next) return
  for (const image of next.images) {
    new Image().src = buildImageUrl(image, 'preview')
  }
}

// 换帧（帧内图片集合变化）或切换预览/原图（variant 变化）时：Loading、重置旋转、预载下一帧；
// 布局尺寸由 SpreadFrame 全部 onload 后上报（onFrameLayout）统一 Fit
watch(
  () =>
    `${viewerStore.currentFrame?.images.map((i) => i.path).join('|') ?? ''}|${viewerStore.variant}`,
  (key) => {
    if (!key.split('|')[0]) return
    loading.value = true
    loadError.value = false
    viewer.resetRotation()
    preloadNextFrame()
  },
)

// 帧布局尺寸就绪：设定内容尺寸并 Fit 居中，解除 Loading
function onFrameLayout(size: { width: number; height: number }) {
  viewer.setContentSize(size.width, size.height)
  viewer.fit()
  loading.value = false
  loadError.value = false
}

// 帧内某图加载失败：解除 Loading 并提示（帧内其余图片继续显示）
function onFrameImageError(_image: ImageFile) {
  loading.value = false
  loadError.value = true
}

// 工具栏缩放按钮：以容器中心为锚点
function zoomAtCenter(factor: number) {
  const el = viewer.containerRef.value
  const cx = el ? el.clientWidth / 2 : 0
  const cy = el ? el.clientHeight / 2 : 0
  viewer.zoomAt(factor, cx, cy)
}

// 容器尺寸变化（图片↔全图、缩略图条显隐、浏览器全屏、窗口缩放）后重新 Fit 的职责
// 由 ViewerCanvas 内的 ResizeObserver 承担（布局完成后触发，无时序问题）；
// 换帧的 Fit 由 onFrameLayout（布局上报）负责
</script>

<template>
  <Teleport to="body" :disabled="viewerStore.mode !== 'full'">
    <div
      v-if="viewerStore.isOpen"
      class="flex flex-col overflow-hidden"
      :class="rootClass"
    >
      <ViewerToolbar
        :scale="scale"
        :is-fullscreen="isFullscreen"
        :show-info="showInfo"
        :page-label="pageLabel"
        :variant="viewerStore.variant"
        :mode="viewerStore.mode"
        :filmstrip-visible="viewerStore.filmstripVisible"
        @fit="viewer.fit()"
        @scale100="viewer.setScale100()"
        @zoom-in="zoomAtCenter(1.2)"
        @zoom-out="zoomAtCenter(0.8)"
        @rotate="viewer.rotate()"
        @toggle-fullscreen="viewer.toggleFullscreen()"
        @toggle-info="showInfo = !showInfo"
        @toggle-variant="viewerStore.setVariant(viewerStore.variant === 'preview' ? 'original' : 'preview')"
        @toggle-filmstrip="viewerStore.toggleFilmstrip()"
        @go-first="viewerStore.select(0)"
      />

      <!-- 画布区：占满剩余高度（缩略图条在下方占据自然高度） -->
      <div class="relative min-h-0 flex-1">
        <ViewerCanvas :viewer="viewer">
          <SpreadFrame
            v-if="currentFrame"
            :frame="currentFrame"
            :variant="viewerStore.variant"
            @layout="onFrameLayout"
            @image-error="onFrameImageError"
          />
        </ViewerCanvas>

        <!-- 加载失败提示（独立于画布 transform） -->
        <div
          v-if="loadError"
          class="pointer-events-none absolute inset-0 z-20 flex items-center justify-center text-sm text-muted"
        >
          图片加载失败
        </div>
      </div>

      <!-- 全图模式底部缩略图条：可显隐；图片模式由 BrowserPage 底部常驻提供 -->
      <Filmstrip
        v-if="viewerStore.mode === 'full' && viewerStore.filmstripVisible"
        :images="viewerStore.images"
        :active-indexes="viewerStore.activeImageIndexes"
        @select="viewerStore.select($event)"
      />

      <LoadingOverlay :visible="loading" />

      <ImageInfo v-if="showInfo" :images="currentFrame?.images ?? []" />

      <AppButton
        variant="icon"
        title="关闭 (Esc)"
        class="absolute right-3 top-3 z-10 opacity-80"
        @click="viewerStore.stepBack()"
      >
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
          <path d="M6 6l12 12M18 6L6 18" />
        </svg>
      </AppButton>

      <!-- 收藏浮动按钮组：✕ 正下方竖排，低调（半透明）不影响浏览；已收藏柔和红。
           单图帧 1 个按钮；双页帧 2 个并带 L/R 标记（位于心形左侧），按帧内 images 下标映射视觉左/右 -->
      <div
        v-if="favorites.available && currentFrame"
        class="absolute right-3 top-14 z-10 flex flex-col items-center gap-1"
      >
        <div
          v-for="(image, i) in currentFrame.images"
          :key="image.path"
          class="flex items-center gap-0.5 opacity-50 transition-opacity hover:opacity-90"
        >
          <span
            v-if="currentFrame.images.length > 1"
            class="text-[10px] leading-none"
            :class="favorites.has(image.path) ? 'text-red-400' : 'text-ink'"
          >{{ i === 0 ? 'L' : 'R' }}</span>
          <button
            type="button"
            :title="favorites.has(image.path) ? '取消收藏 (S)' : '收藏 (S)'"
            class="flex h-7 w-7 items-center justify-center"
            :class="favorites.has(image.path) ? 'text-red-400' : 'text-ink'"
            @click="toggleFrameFavorite(i)"
          >
            <svg
              class="h-[18px] w-[18px]"
              viewBox="0 0 24 24"
              :fill="favorites.has(image.path) ? 'currentColor' : 'none'"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
