<script setup lang="ts">
// 浏览页：目录树（侧栏）+ 模式驱动内容区（文件模式卡片网格 / 图片·全图模式嵌入查看器）+ 底部 Filmstrip 缩略图条
// 初始根目录加载由 FolderTree onMounted 统一负责，本页不重复触发
import { computed, watch } from 'vue'
import { useFolderStore } from '../stores/folder'
import { useFavoritesStore } from '../stores/favorites'
import { useViewerStore } from '../stores/viewer'
import { useSpreadStore } from '../stores/spread'
import type { ImageFile } from '../types/file'
import AppLayout from '../components/layout/AppLayout.vue'
import EmptyState from '../components/common/EmptyState.vue'
import Filmstrip from '../components/filmstrip/Filmstrip.vue'
import ImageViewer from '../components/viewer/ImageViewer.vue'
import { formatFileSize } from '../utils/format'

const store = useFolderStore()
const favorites = useFavoritesStore()
const viewerStore = useViewerStore()
const spreadStore = useSpreadStore()

// 内容区数据源：收藏夹虚拟视图展示 favorites store 实时列表（收藏时间倒序，取消收藏即移除）；
// 目录内「只看收藏」筛选过滤当前目录；否则为当前目录完整列表
const displayImages = computed(() => {
  if (store.favoritesView) return favorites.images
  if (store.favoritesOnly) return store.images.filter((image) => favorites.has(image.path))
  return store.images
})
const displayLoading = computed(() => (store.favoritesView ? favorites.loading : store.loading))
const displayError = computed(() => (store.favoritesView ? favorites.error : store.error))

// 打开查看器：收藏夹为跨目录混合列表，强制单页帧化（一帧一图，不落库不写 folder_settings）
function openViewer(images: ImageFile[], index: number) {
  if (store.favoritesView) spreadStore.pageMode = 'single'
  viewerStore.open(images, index)
}

// 网格卡片心形角标：收藏/取消收藏；失败已回滚，静默
async function toggleGridFavorite(image: ImageFile) {
  try {
    await favorites.toggle(image.path, image)
  } catch {
    /* 忽略：favorites store 已回滚乐观状态 */
  }
}

// 目录列表变化时若已处于图片/全图模式（持久化模式刷新首屏、阅读中切换目录），
// 以筛选后列表（displayImages，「只看收藏」生效）第 1 帧继续当前模式
// （open 在已打开状态下不改变 mode）；收藏夹视图内取消收藏不触发本 watch
watch(
  () => store.images,
  (images) => {
    if (!store.favoritesView && images.length > 0 && viewerStore.mode !== 'file') {
      openViewer(displayImages.value, 0)
    }
  },
)

// 进入收藏夹虚拟视图时若已处于阅读模式，以收藏列表第 1 帧继续（强制单页）
watch(
  () => store.favoritesView,
  (favoritesView) => {
    if (favoritesView && favorites.images.length > 0 && viewerStore.mode !== 'file') {
      openViewer(favorites.images, 0)
    }
  },
)

// 图片/全图模式下切换「只看收藏」：以筛选后列表重建查看数据，尽量保持当前图（不在列表则回第 1 张）
watch(
  () => store.favoritesOnly,
  () => {
    if (store.favoritesView || viewerStore.mode === 'file') return
    const images = displayImages.value
    if (images.length === 0) return
    const current = viewerStore.currentImage
    const index = current ? images.findIndex((image) => image.path === current.path) : -1
    openViewer(images, index >= 0 ? index : 0)
  },
)

// 经 Header 切换按钮从文件模式进入图片/全图模式：查看器尚无内容时自动打开数据源第 1 帧
watch(
  () => viewerStore.mode,
  (mode) => {
    if (mode !== 'file' && viewerStore.images.length === 0 && displayImages.value.length > 0) {
      openViewer(displayImages.value, 0)
    }
  },
)
</script>

<template>
  <AppLayout>
    <!-- 纵向 flex：内容区独立滚动（仅文件模式需要），Filmstrip 固定底栏不参与页面滚动 -->
    <div class="flex h-full flex-col">
      <!-- 主内容区：文件模式为可滚动网格；图片/全图模式由查看器占满剩余高度 -->
      <div class="min-h-0 flex-1" :class="viewerStore.mode === 'file' ? 'overflow-y-auto' : ''">
        <template v-if="viewerStore.mode === 'file' || displayImages.length === 0">
          <div v-if="displayLoading" class="flex items-center justify-center p-8 text-sm text-faint">
            加载中…
          </div>

          <EmptyState v-else-if="displayError" :message="displayError" />

          <EmptyState
            v-else-if="displayImages.length === 0"
            :message="store.favoritesView ? '还没有收藏图片。打开图片后点击 ✕ 下方心形按钮或按 S 即可收藏。' : '此目录没有图片'"
          />

          <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-4 p-4">
            <div
              v-for="(image, index) in displayImages"
              :key="image.path"
              class="group relative cursor-pointer rounded-lg border border-line bg-panel p-3 transition-colors hover:border-line-hover hover:bg-elevated/60"
              :data-path="image.path"
              @click="openViewer(displayImages, index)"
            >
              <!-- 心形角标（右下角，避开长文件名）：已收藏常显柔和红；未收藏悬停卡片时半透明显现，点击切换（不触发打开） -->
              <button
                v-if="favorites.available"
                type="button"
                :title="favorites.has(image.path) ? '取消收藏' : '收藏'"
                class="absolute bottom-2 right-2 flex h-7 w-7 items-center justify-center rounded transition-opacity"
                :class="favorites.has(image.path)
                  ? 'text-red-400 opacity-90'
                  : 'text-ink opacity-0 group-hover:opacity-50 hover:!opacity-90'"
                @click.stop="toggleGridFavorite(image)"
              >
                <svg
                  class="h-4 w-4"
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
              <p class="truncate text-sm text-ink">{{ image.name }}</p>
              <p class="mt-1 text-xs text-muted">
                {{ image.width !== null && image.height !== null ? `${image.width}×${image.height}` : '—' }}
              </p>
              <p class="text-xs text-faint">{{ formatFileSize(image.size) }}</p>
            </div>
          </div>
        </template>

        <!-- 图片模式：原地嵌入查看器；全图模式经 Teleport 覆盖全屏（同一实例双形态） -->
        <ImageViewer v-else />
      </div>

      <!-- 底部缩略图条：文件/图片模式常驻（全图模式移入查看器内、可显隐）；打开时高亮当前帧，点击直达该帧 -->
      <Filmstrip
        v-if="viewerStore.mode !== 'full'"
        :images="displayImages"
        :active-indexes="viewerStore.isOpen ? viewerStore.activeImageIndexes : []"
        @select="openViewer(displayImages, $event)"
      />
    </div>
  </AppLayout>
</template>
