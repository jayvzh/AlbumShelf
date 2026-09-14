<script setup lang="ts">
// 移动端完整阅读器：全屏画布 + tap 点区翻页（RTL 感知）+ 双指缩放 + 进度滑条 + 控件自动隐藏。
// 帧计算绕开 spreadStore：强制 single 模式自算（帧索引 == 图片索引），
// 翻页直改 viewerStore.currentFrameIndex，桌面双页设置零污染（同 MobileBrowserPage.openViewer）。
// 换帧加载 / 预载 / 键盘绑定模式照搬桌面 ImageViewer；tap 分区由 usePan 透传（仅 touch/pen）。
import { computed, onUnmounted, ref, watch } from 'vue'
import { useViewerStore } from '../../stores/viewer'
import { useSettingsStore } from '../../stores/settings'
import { useFavoritesStore } from '../../stores/favorites'
import { useViewer } from '../../composables/useViewer'
import { useKeyboard } from '../../composables/useKeyboard'
import type { TapDetail } from '../../composables/usePan'
import { buildSpreadFrames, DEFAULT_SPREAD_OPTIONS } from '../../utils/spread'
import { cancelAll, syncViewerPreload } from '../../utils/imagePreloader'
import { formatFileSize } from '../../utils/format'
import ViewerCanvas from '../viewer/ViewerCanvas.vue'
import SpreadFrame from '../viewer/SpreadFrame.vue'
import LoadingOverlay from '../viewer/LoadingOverlay.vue'
import ImageInfo from '../viewer/ImageInfo.vue'

const viewerStore = useViewerStore()
const settingsStore = useSettingsStore()
const favorites = useFavoritesStore()

// 强制单页构帧（single 恒等映射），不触碰 spreadStore
const frames = computed(
  () =>
    buildSpreadFrames(viewerStore.images, { ...DEFAULT_SPREAD_OPTIONS, pageMode: 'single' })
      .frames,
)
const frameCount = computed(() => frames.value.length)
const currentFrame = computed(() => frames.value[viewerStore.currentFrameIndex])
const currentImage = computed(() => currentFrame.value?.images[0])
const isFavorited = computed(() =>
  currentImage.value ? favorites.has(currentImage.value.path) : false,
)
const rtl = computed(() => settingsStore.readOrder === 'right_to_left')

// 微信式原图提示：未看过原图时按钮带体积（如"查看原图 2M"），看过一次后仅显示"查看原图"
const originalViewed = ref(new Set<string>())
const originalSizeHint = computed(() => {
  const image = currentImage.value
  if (!image || originalViewed.value.has(image.path)) return ''
  return formatFileSize(image.size, true)
})

// 缩放 / 平移 / 旋转运行态（tap 回调在下方定义，函数声明存在提升）
const viewer = useViewer({ onTap })

// 控件可见性：3s 无操作自动隐藏
const uiVisible = ref(true)
let uiTimer: ReturnType<typeof setTimeout> | undefined

function scheduleUiHide() {
  if (uiTimer) clearTimeout(uiTimer)
  uiTimer = setTimeout(() => (uiVisible.value = false), 3000)
}
function showControls() {
  uiVisible.value = true
  scheduleUiHide()
}
function toggleControls() {
  uiVisible.value = !uiVisible.value
  if (uiVisible.value) scheduleUiHide()
  else if (uiTimer) clearTimeout(uiTimer)
}
onUnmounted(() => {
  if (uiTimer) clearTimeout(uiTimer)
  cancelAll()
})

// 翻页：落位帧索引 + 回预览档 + 唤起控件
function goToFrame(index: number) {
  viewerStore.currentFrameIndex = Math.min(Math.max(index, 0), frameCount.value - 1)
  viewerStore.variant = 'preview'
}
function nextFrame() {
  if (viewerStore.currentFrameIndex < frameCount.value - 1) {
    goToFrame(viewerStore.currentFrameIndex + 1)
  }
  if (uiVisible.value) scheduleUiHide()
}
function previousFrame() {
  if (viewerStore.currentFrameIndex > 0) {
    goToFrame(viewerStore.currentFrameIndex - 1)
  }
  if (uiVisible.value) scheduleUiHide()
}

// tap 点区：左 1/3 上一帧 / 右 1/3 下一帧（RTL 互换）/ 中 1/3 切控件；
// 放大态（scale > 1.1）tap 意图是平移回看，仅切控件防误触翻页
function onTap(detail: TapDetail) {
  if (viewer.scale.value > 1.1) {
    toggleControls()
    return
  }
  const rel = detail.x / detail.width
  if (rel < 1 / 3) (rtl.value ? nextFrame : previousFrame)()
  else if (rel > 2 / 3) (rtl.value ? previousFrame : nextFrame)()
  else toggleControls()
}

// 收藏当前帧图片（角标回滚由 store 负责，失败静默）
async function toggleFavoriteCurrent() {
  const image = currentImage.value
  if (!image) return
  try {
    await favorites.toggle(image.path, image)
  } catch {
    // 静默
  }
}

// 键盘：随阅读器开关绑定/解绑
const keyboard = useKeyboard({
  previous: previousFrame,
  next: nextFrame,
  toggleFullscreen: () => viewer.toggleFullscreen(),
  fit: () => viewer.fit(),
  setScale100: () => viewer.setScale100(),
  rotate: () => viewer.rotate(),
  close: () => viewerStore.setMode('file'),
  toggleFavorite: toggleFavoriteCurrent,
})

watch(
  () => viewerStore.isOpen,
  (open) => {
    if (open) keyboard.bind()
    else {
      keyboard.unbind()
      cancelAll()
    }
  },
  { immediate: true },
)

// 换帧 / 换档：标记加载中、重置旋转、同步预取（相邻帧抢占置顶 + 其余帧空闲预热）
const loading = ref(true)
const loadError = ref(false)

watch(
  () => `${viewerStore.currentFrameIndex}|${viewerStore.variant}`,
  () => {
    loading.value = true
    loadError.value = false
    viewer.resetRotation()
    syncViewerPreload(frames.value, viewerStore.currentFrameIndex)
  },
  { immediate: true },
)

// SpreadFrame 完成等高布局 → 上报内容尺寸 + Fit + 结束加载（同桌面）
function onFrameLayout(size: { width: number; height: number }) {
  viewer.setContentSize(size.width, size.height)
  viewer.fit()
  loading.value = false
  // 原图档加载成功 → 记为"已看过"，后续按钮不再提示体积
  if (viewerStore.variant === 'original' && !loadError.value && currentImage.value) {
    originalViewed.value.add(currentImage.value.path)
  }
}
function onImageError() {
  loadError.value = true
  loading.value = false
}

function onSliderInput(e: Event) {
  goToFrame(Number((e.target as HTMLInputElement).value))
  showControls()
}

function toggleVariant() {
  viewerStore.setVariant(viewerStore.variant === 'preview' ? 'original' : 'preview')
  showControls()
}

// 左上角面包菜单：信息 / 镜像 / 旋转 / 收起（垂直透明条，随控件一起自动隐藏）
const menuOpen = ref(false)
const showInfo = ref(false)
function toggleMenu() {
  menuOpen.value = !menuOpen.value
  showControls()
}

function closeReader() {
  viewerStore.setMode('file')
}

// 全屏切换：仅浏览器 API 全屏（无模式联动），点击后重置 3s 自动隐藏计时
function onToggleFullscreen() {
  viewer.toggleFullscreen()
  showControls()
}

// 关闭阅读器时复位菜单与信息面板（组件常驻，状态跨开关保留）
watch(
  () => viewerStore.isOpen,
  (open) => {
    if (!open) {
      menuOpen.value = false
      showInfo.value = false
    }
  },
)
</script>

<template>
  <Teleport to="body">
    <div v-if="viewerStore.isOpen && currentFrame" class="fixed inset-0 z-50 bg-black">
      <!-- 全屏画布：tap 点区 / 拖拽平移 / 双指缩放（手势绑定在画布上，控件栏在其外不受影响） -->
      <ViewerCanvas :viewer="viewer">
        <SpreadFrame
          :frame="currentFrame"
          :variant="viewerStore.variant"
          @layout="onFrameLayout"
          @image-error="onImageError"
        />
      </ViewerCanvas>

      <LoadingOverlay :visible="loading" />

      <!-- 加载失败提示（SpreadFrame 已落 400×600 兜底尺寸） -->
      <div
        v-if="loadError"
        class="pointer-events-none absolute inset-0 z-20 flex items-center justify-center text-sm text-white/70"
      >
        图片加载失败
      </div>

      <!-- 顶部控件：面包菜单 / 页码 / 关闭；面包点击向下展开垂直透明控件条 -->
      <Transition name="reader-fade">
        <div
          v-show="uiVisible"
          class="absolute inset-x-0 top-0 z-30 bg-gradient-to-b from-black/70 to-transparent px-2 pt-[max(env(safe-area-inset-top),0.5rem)] pb-2"
        >
          <div class="relative flex items-center gap-1">
            <button
              type="button"
              class="flex h-11 w-11 items-center justify-center text-white/90 active:bg-white/10"
              :aria-label="menuOpen ? '收起菜单' : '展开菜单'"
              @click="toggleMenu"
            >
              <svg
                class="h-5 w-5"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
              >
                <path d="M4 6h16" />
                <path d="M4 12h16" />
                <path d="M4 18h16" />
              </svg>
            </button>
            <div class="flex-1 text-center text-sm tabular-nums text-white/90">
              {{ viewerStore.currentFrameIndex + 1 }} / {{ frameCount }}
            </div>
            <!-- 镜像激活悬浮标志：全屏按钮左侧，点击退出镜像（-ml-2 与全屏/X 收紧为成组图标） -->
            <button
              v-if="viewer.mirrored.value"
              type="button"
              class="-ml-2 flex h-11 w-11 items-center justify-center text-amber-400 active:bg-white/10"
              aria-label="退出左右镜像"
              @click="viewer.toggleMirror()"
            >
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m18 7 4 4-4 4" />
                <path d="m6 7-4 4 4 4" />
                <path d="M12 3v18" />
              </svg>
            </button>
            <!-- 全屏 / 退出全屏（浏览器 API 全屏，与桌面工具栏同义；移动端无最大化概念）。
                 -ml-2 与镜像/X 收紧为成组图标（镜像隐藏时仅拉近与页码区间距，不影响布局） -->
            <button
              type="button"
              class="-ml-2 flex h-11 w-11 items-center justify-center text-white/90 active:bg-white/10"
              :aria-label="viewer.isFullscreen.value ? '退出全屏' : '全屏'"
              @click="onToggleFullscreen"
            >
              <svg v-if="!viewer.isFullscreen.value" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M8 3H5a2 2 0 0 0-2 2v3M16 3h3a2 2 0 0 1 2 2v3M8 21H5a2 2 0 0 1-2-2v-3M16 21h3a2 2 0 0 0 2-2v-3" />
              </svg>
              <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M8 3v3a2 2 0 0 1-2 2H3M16 3v3a2 2 0 0 0 2 2h3M8 21v-3a2 2 0 0 0-2-2H3M16 21v-3a2 2 0 0 1 2-2h3" />
              </svg>
            </button>
            <!-- 关闭 + 收纳入同一锚点容器：心形以 X 的 44px 盒为参照绝对定位，
                 无论前面按钮的负边距如何收紧，二者右缘/中心始终同垂直线 -->
            <div class="relative -ml-2">
              <button
                type="button"
                class="flex h-11 w-11 items-center justify-center text-white/90 active:bg-white/10"
                aria-label="关闭阅读器"
                @click="closeReader"
              >
                <svg
                  class="h-5 w-5"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M18 6 6 18" />
                  <path d="m6 6 12 12" />
                </svg>
              </button>
              <!-- 收藏按钮：✕ 正下方同垂直线（同宽 w-11），配色低调仿桌面（未收藏 white/50，已收藏柔和红） -->
              <button
                v-if="favorites.available"
                type="button"
                class="absolute right-0 top-full mt-1 flex h-11 w-11 items-center justify-center active:bg-white/10"
                :class="isFavorited ? 'text-red-400' : 'text-white/50'"
                :aria-label="isFavorited ? '取消收藏' : '收藏'"
                @click="toggleFavoriteCurrent"
              >
                <svg
                  class="h-5 w-5"
                  :fill="isFavorited ? 'currentColor' : 'none'"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  viewBox="0 0 24 24"
                >
                  <path
                    d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"
                  />
                </svg>
              </button>
            </div>
          </div>
          <!-- 垂直透明控件条：信息 / 镜像 / 旋转（-ml-1 使图标列与面包按钮对齐；再次点面包即收起） -->
          <div
            v-show="menuOpen"
            class="-ml-1 mt-1 flex w-fit flex-col gap-0.5 rounded-xl bg-black/50 p-1"
          >
            <button
              type="button"
              class="flex h-11 w-11 items-center justify-center active:bg-white/10"
              :class="showInfo ? 'text-amber-400' : 'text-white/90'"
              aria-label="图片信息"
              @click="showInfo = !showInfo"
            >
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <circle cx="12" cy="12" r="9" />
                <path d="M12 8h.01M12 11v5" />
              </svg>
            </button>
            <button
              type="button"
              class="flex h-11 w-11 items-center justify-center active:bg-white/10"
              :class="viewer.mirrored.value ? 'text-amber-400' : 'text-white/90'"
              :aria-label="viewer.mirrored.value ? '退出左右镜像' : '左右镜像'"
              @click="viewer.toggleMirror()"
            >
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m18 7 4 4-4 4" />
                <path d="m6 7-4 4 4 4" />
                <path d="M12 3v18" />
              </svg>
            </button>
            <button
              type="button"
              class="flex h-11 w-11 items-center justify-center text-white/90 active:bg-white/10"
              aria-label="顺时针旋转 90°"
              @click="viewer.rotate()"
            >
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 12a9 9 0 1 1-2.64-6.36" />
                <path d="M21 3v6h-6" />
              </svg>
            </button>
          </div>
        </div>
      </Transition>

      <!-- 底部控件：查看原图（进度条上方） / 进度滑条 / 收藏 -->
      <Transition name="reader-fade">
        <div
          v-show="uiVisible"
          class="absolute inset-x-0 bottom-0 z-30 flex flex-col gap-1 bg-gradient-to-t from-black/70 to-transparent px-3 pt-2 pb-[max(env(safe-area-inset-bottom),0.5rem)]"
        >
          <button
            type="button"
            class="ml-1 flex h-8 shrink-0 items-center justify-center self-start rounded-full border border-white/30 px-3 text-xs text-white/90 active:bg-white/10"
            @click="toggleVariant"
          >
            <template v-if="viewerStore.variant === 'original'">返回预览</template>
            <template v-else>
              查看原图<span v-if="originalSizeHint" class="ml-1 text-[10px] text-white/55">{{ originalSizeHint }}</span>
            </template>
          </button>
          <div class="flex items-center gap-2">
            <input
              type="range"
              class="h-11 min-w-0 flex-1 accent-white"
              :min="0"
              :max="Math.max(frameCount - 1, 0)"
              :value="viewerStore.currentFrameIndex"
              aria-label="阅读进度"
              @input="onSliderInput"
            />
          </div>
        </div>
      </Transition>

      <!-- 图片信息面板：抬离底部控件条，避免遮挡 -->
      <ImageInfo
        v-if="showInfo"
        class="bottom-28!"
        :images="currentFrame?.images ?? []"
      />
    </div>
  </Teleport>
</template>

<style scoped>
.reader-fade-enter-active,
.reader-fade-leave-active {
  transition: opacity 0.2s ease;
}
.reader-fade-enter-from,
.reader-fade-leave-to {
  opacity: 0;
}
</style>
