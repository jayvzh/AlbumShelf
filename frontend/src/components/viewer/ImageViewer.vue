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
import { useIsMobile, useMediaQuery } from '../../composables/useMediaQuery'
import { cancelAll, syncViewerPreload } from '../../utils/imagePreloader'
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

// 全屏与模式联动路径标记：图片模式进入全屏时自动切全图，退出全屏后自动回到图片模式；
// 全图模式进入全屏仅切换浏览器全屏，退出后保持全图
const enteredFullFromImage = ref(false)

async function handleToggleFullscreen() {
  if (viewer.isFullscreen.value) {
    await viewer.toggleFullscreen()
    return
  }
  if (viewerStore.mode === 'image') {
    enteredFullFromImage.value = true
    viewerStore.setMode('full')
  }
  // 以 requestFullscreen 的 promise 结果为准（此时 fullscreenchange 事件可能尚未派发，不能读 isFullscreen）
  const entered = await viewer.toggleFullscreen()
  if (enteredFullFromImage.value && !entered) {
    enteredFullFromImage.value = false
    viewerStore.setMode('image')
  }
}

// 退出全屏时回退联动进入的全图模式（用户已手动点还原则模式已是 image，跳过）
watch(
  () => viewer.isFullscreen.value,
  (fullscreen) => {
    if (!fullscreen && enteredFullFromImage.value) {
      enteredFullFromImage.value = false
      if (viewerStore.mode === 'full') viewerStore.setMode('image')
    }
  },
)

const currentFrame = computed(() => viewerStore.currentFrame)
const scale = computed(() => viewer.scale.value)
const isFullscreen = computed(() => viewer.isFullscreen.value)

// 手机浏览器"请求桌面版"检测：触屏 primary pointer 且未走移动树（与 BrowserPlatformPage 分流条件一致）
const touchPointer = useMediaQuery('(pointer: coarse)')
const isMobileViewport = useIsMobile()

// 当前帧宽高比（w/h；双页取宽度累加 / 最大高、忽略页间距的近似）
const frameRatio = computed(() => {
  const sized = (currentFrame.value?.images ?? []).filter(
    (i) => i.width !== null && i.height !== null && i.height > 0,
  )
  if (sized.length === 0) return null
  const width = sized.reduce((sum, i) => sum + (i.width ?? 0), 0)
  const height = Math.max(...sized.map((i) => i.height ?? 0))
  return width / height
})

// 手机"请求桌面版"时 layout viewport 竖长（如 980×2100），画布区被 flex-1 拉满 100vh 虚高，
// 竖帧 Fit 后大面积留白。收缩包裹层让高度贴合帧比例：高 = min(可用高, 宽 ÷ 帧比例)。
// 正常桌面横屏下"宽 ÷ 比例"恒大于可用高（max-height 恒命中），布局零变化；移动树不经此组件。
// 仅全图模式启用：图片模式的底部缩略图条挂在 BrowserPage 虚高底部，收缩画布无法上移它。
const canvasWrapStyle = computed(() => {
  if (viewerStore.mode !== 'full' || isMobileViewport.value || !touchPointer.value) return undefined
  return frameRatio.value ? { aspectRatio: String(frameRatio.value), maxHeight: '100%' } : undefined
})
// 收缩时高度交给 aspect-ratio 推导（height auto）；否则撑满外层（ViewerCanvas 根依赖父级定高）
const canvasWrapClass = computed(() => (canvasWrapStyle.value ? 'w-full' : 'h-full w-full'))

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
  toggleFullscreen: () => void handleToggleFullscreen(),
  fit: () => viewer.fit(),
  setScale100: () => viewer.setScale100(),
  rotate: () => viewer.rotate(),
  close: () => viewerStore.stepBack(),
  toggleFavorite: () => toggleFrameFavorite(0),
})

// immediate 覆盖持久化模式刷新首屏（isOpen 初始即 true 的场景）；关闭时取消在途预取
watch(
  () => viewerStore.isOpen,
  (open) => {
    if (open) {
      keyboard.bind()
    } else {
      keyboard.unbind()
      cancelAll()
    }
  },
  { immediate: true },
)
// 卸载时必须解绑：onKeyDown 是 useKeyboard 每次调用创建的闭包（引用不同），
// HMR 或 v-if/v-else 重挂后旧监听残留会导致一次 Esc 触发两次 stepBack（全图→文件跳级）
onUnmounted(() => {
  keyboard.unbind()
  cancelAll()
})

// 换帧（帧内图片集合变化）或切换预览/原图（variant 变化）时：Loading、重置旋转、同步预取；
// 布局尺寸由 SpreadFrame 全部 onload 后上报（onFrameLayout）统一 Fit
watch(
  () =>
    `${viewerStore.currentFrame?.images.map((i) => i.path).join('|') ?? ''}|${viewerStore.variant}`,
  (key) => {
    if (!key.split('|')[0]) return
    loading.value = true
    loadError.value = false
    viewer.resetRotation()
    syncViewerPreload(spreadStore.frames, viewerStore.currentFrameIndex)
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
        :mirrored="viewer.mirrored.value"
        @fit="viewer.fit()"
        @scale100="viewer.setScale100()"
        @zoom-in="zoomAtCenter(1.2)"
        @zoom-out="zoomAtCenter(0.8)"
        @rotate="viewer.rotate()"
        @toggle-mirror="viewer.toggleMirror()"
        @toggle-fullscreen="handleToggleFullscreen()"
        @toggle-info="showInfo = !showInfo"
        @toggle-variant="viewerStore.setVariant(viewerStore.variant === 'preview' ? 'original' : 'preview')"
        @toggle-filmstrip="viewerStore.toggleFilmstrip()"
        @go-first="viewerStore.select(0)"
      />

      <!-- 画布区：占满剩余高度（缩略图条在下方占据自然高度）；触屏桌面树按帧比例收缩 -->
      <div class="relative min-h-0 flex-1">
        <div class="w-full" :class="canvasWrapClass" :style="canvasWrapStyle">
          <ViewerCanvas :viewer="viewer">
            <SpreadFrame
              v-if="currentFrame"
              :frame="currentFrame"
              :variant="viewerStore.variant"
              @layout="onFrameLayout"
              @image-error="onFrameImageError"
            />
          </ViewerCanvas>
        </div>

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

      <!-- 右上角控件行：镜像激活标志（最左）→ 最大化/还原 → 关闭 X；镜像标志仅激活时出现，flex 自动避让不重叠。
           top 取 17px：左侧控件条 pill 高 42px（p-1+边框+h-8），本组裸按钮高 32px，
           同顶时右侧中心偏高 5px；17px 使两侧按钮中心同在 y≈33 的水平线上 -->
      <div class="absolute right-3 top-[17px] z-10 flex items-center gap-2">
        <!-- 镜像激活悬浮标志：点击退出镜像 -->
        <AppButton
          v-if="viewer.mirrored.value"
          variant="icon"
          title="退出左右镜像"
          class="opacity-80"
          @click="viewer.toggleMirror()"
        >
          <span class="flex text-accent-text">
            <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="m18 7 4 4-4 4" />
              <path d="m6 7-4 4 4 4" />
              <path d="M12 3v18" />
            </svg>
          </span>
        </AppButton>

        <!-- 最大化/还原：图片模式 → 全图覆盖；全图模式 → 回到嵌入图片（与工具栏 API 全屏相互独立） -->
        <AppButton
          v-if="viewerStore.mode !== 'full'"
          variant="icon"
          title="最大化"
          class="opacity-80"
          @click="viewerStore.setMode('full')"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <rect x="4" y="4" width="16" height="16" rx="1.5" />
          </svg>
        </AppButton>
        <AppButton
          v-else
          variant="icon"
          title="还原"
          class="opacity-80"
          @click="viewerStore.setMode('image')"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 3H5a2 2 0 0 0-2 2v10" />
            <rect x="9" y="9" width="12" height="12" rx="1.5" />
          </svg>
        </AppButton>

        <AppButton
          variant="icon"
          title="关闭 (Esc)"
          class="opacity-80"
          @click="viewerStore.stepBack()"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
            <path d="M6 6l12 12M18 6L6 18" />
          </svg>
        </AppButton>
      </div>

      <!-- 收藏浮动按钮组：✕ 正下方竖排、同垂直中线（每格 h-8 w-8 与 ✕ 同宽）；
           L/R 徽标绝对定位在心形左侧，不挤偏按钮；
           top 61px 跟随右上角行（top 17 + 高 32 + 12px 间距）；
           低调（半透明）不影响浏览；已收藏柔和红 -->
      <div
        v-if="favorites.available && currentFrame"
        class="absolute right-3 top-[61px] z-10 flex flex-col items-center gap-1"
      >
        <button
          v-for="(image, i) in currentFrame.images"
          :key="image.path"
          type="button"
          :title="favorites.has(image.path) ? '取消收藏 (S)' : '收藏 (S)'"
          class="relative flex h-8 w-8 items-center justify-center opacity-50 transition-opacity hover:opacity-90"
          :class="favorites.has(image.path) ? 'text-red-400' : 'text-ink'"
          @click="toggleFrameFavorite(i)"
        >
          <span
            v-if="currentFrame.images.length > 1"
            class="absolute right-full mr-0.5 text-[10px] leading-none"
            :class="favorites.has(image.path) ? 'text-red-400' : 'text-ink'"
          >{{ i === 0 ? 'L' : 'R' }}</span>
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
  </Teleport>
</template>
