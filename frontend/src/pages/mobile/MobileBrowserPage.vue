<script setup lang="ts">
// 移动端浏览页：顶栏（汉堡/标题/网格阅读切换/设置）+ 目录抽屉 + 封面网格
// 复刻桌面两个副作用（桌面由 FolderTree / AppHeader 承担，移动组件树独立挂载）：
//   1. FolderTree：onMounted restore（恢复上次会话目录与展开状态）+ auth→protected/quickAccess 联动
//   2. AppHeader：watch currentPath → settingsStore.load（加载该目录排序/阅读设置）
// 打开阅读器绕开 viewerStore.open（其内部 spread.rebuild 会以桌面持久化 pageMode 构帧），
// 直接写 images + currentFrameIndex（MobileReader 强制单页构帧：帧索引 == 图片索引），
// spreadStore 完全不被触碰，桌面双页设置不受污染
import { computed, onMounted, ref, watch } from 'vue'
import { useFolderStore } from '../../stores/folder'
import { useFavoritesStore } from '../../stores/favorites'
import { useViewerStore } from '../../stores/viewer'
import { useAuthStore } from '../../stores/auth'
import { useProtectedStore } from '../../stores/protected'
import { useQuickAccessStore } from '../../stores/quickAccess'
import { useSettingsStore } from '../../stores/settings'
import MobileHeader from '../../components/mobile/MobileHeader.vue'
import MobileFolderDrawer from '../../components/mobile/MobileFolderDrawer.vue'
import MobileImageGrid from '../../components/mobile/MobileImageGrid.vue'
import MobileReader from '../../components/mobile/MobileReader.vue'
import BrandEmptyState from '../../components/common/BrandEmptyState.vue'
import EmptyState from '../../components/common/EmptyState.vue'
import { DIR_UNLOCK_THRESHOLD, markDirWarmUnlocked, syncDirWarm } from '../../utils/imagePreloader'
import type { ImageFile } from '../../types/file'

const folderStore = useFolderStore()
const favorites = useFavoritesStore()
const viewerStore = useViewerStore()
const auth = useAuthStore()
const protectedStore = useProtectedStore()
const quickAccess = useQuickAccessStore()
const settingsStore = useSettingsStore()

// 目录抽屉开关
const drawerOpen = ref(false)

// 内容区数据源（语义同桌面 BrowserPage）：收藏夹虚拟视图 → favorites 实时列表；
// 「只看收藏」筛选当前目录；否则当前目录完整列表
const displayImages = computed(() => {
  if (folderStore.favoritesView) return favorites.images
  if (folderStore.favoritesOnly) return folderStore.images.filter((image) => favorites.has(image.path))
  return folderStore.images
})
const displayLoading = computed(() => (folderStore.favoritesView ? favorites.loading : folderStore.loading))
const displayError = computed(() => (folderStore.favoritesView ? favorites.error : folderStore.error))

// 打开阅读器：不调用 viewerStore.open（见头部说明），直接落位单页帧索引
function openViewer(images: ImageFile[], index: number) {
  if (images.length === 0) return
  viewerStore.images = images
  viewerStore.currentFrameIndex = Math.min(Math.max(index, 0), images.length - 1)
  viewerStore.variant = 'preview'
  if (!viewerStore.isOpen) viewerStore.setMode('image')
}

// 登录后加载私有目录与快捷访问列表（同 FolderTree）；退出登录时清空
watch(
  () => auth.authenticated,
  (authed) => {
    if (authed) {
      protectedStore.load()
      quickAccess.fetchAll()
    } else {
      protectedStore.paths = new Set()
      quickAccess.clear()
    }
  },
  { immediate: true },
)

// 打开/切换目录时加载该目录已保存的设置（immediate 覆盖首屏；收藏夹虚拟视图跳过，同 AppHeader）
watch(
  () => folderStore.currentPath,
  (path) => {
    if (!folderStore.favoritesView) settingsStore.load(path)
  },
  { immediate: true },
)

// 目录列表变化时若已在阅读模式（刷新首屏恢复、阅读中切换目录）：以新列表第 1 张继续
watch(
  () => folderStore.images,
  (images) => {
    if (!folderStore.favoritesView && images.length > 0 && viewerStore.mode !== 'file') {
      openViewer(displayImages.value, 0)
    }
  },
)

// 进入收藏夹虚拟视图时若已在阅读模式：以收藏列表第 1 张继续
watch(
  () => folderStore.favoritesView,
  (favoritesView) => {
    if (favoritesView && favorites.images.length > 0 && viewerStore.mode !== 'file') {
      openViewer(favorites.images, 0)
    }
  },
)

// 经 Header 切换按钮从网格进入阅读模式：尚无内容时自动打开数据源第 1 张
watch(
  () => viewerStore.mode,
  (mode) => {
    if (mode !== 'file' && viewerStore.images.length === 0 && displayImages.value.length > 0) {
      openViewer(displayImages.value, 0)
    }
  },
)

// 目录空闲预热（DIRWARM）：同桌面 BrowserPage——目录列表/收藏夹视图/阅读器开关
// 变化时重建最低档（切目录与关阅读器都会 cancelAll 清空队列）；收藏夹传空数组仅清档
watch(
  () => [folderStore.images, folderStore.favoritesView, viewerStore.isOpen],
  () => syncDirWarm(folderStore.favoritesView ? [] : folderStore.images, folderStore.currentPath),
  { immediate: true },
)

// 深翻解锁：移动端强制单页（帧索引 == 图片索引），查看第 101 张及以后即解锁该目录
watch(
  () => viewerStore.currentFrameIndex,
  (index) => {
    if (folderStore.favoritesView || index < DIR_UNLOCK_THRESHOLD) return
    markDirWarmUnlocked(folderStore.currentPath)
    syncDirWarm(folderStore.images, folderStore.currentPath)
  },
)

// 初始加载统一由此处负责：恢复上次会话（同 FolderTree）
onMounted(() => {
  folderStore.restore()
})

// 抽屉内选中目录/收藏夹 → 收起抽屉
function onDrawerSelect() {
  drawerOpen.value = false
}
</script>

<template>
  <div class="flex h-dvh flex-col bg-base text-body">
    <MobileHeader @menu="drawerOpen = true" />

    <!-- 内容区：封面网格（阅读器为 MobileReader 的 fixed 覆盖层，网格滚动位置天然保持） -->
    <main class="flex-1 overflow-y-auto overscroll-contain">
      <div v-if="displayLoading" class="flex items-center justify-center p-8 text-sm text-faint">
        加载中…
      </div>
      <div v-else-if="displayError" class="flex items-center justify-center p-8 text-sm text-red-400">
        {{ displayError }}
      </div>
      <div
        v-else-if="displayImages.length === 0 && !folderStore.favoritesView && folderStore.currentPath === '/'"
        class="h-full"
      >
        <!-- 根目录无图片：品牌化空状态（与桌面共用组件，移动端缩小插画/去键盘提示） -->
        <BrandEmptyState platform="mobile" />
      </div>
      <div v-else-if="displayImages.length === 0" class="flex h-full items-center justify-center">
        <EmptyState :message="folderStore.favoritesView ? '还没有收藏图片' : '此目录没有图片'" />
      </div>
      <MobileImageGrid v-else :images="displayImages" @open="openViewer(displayImages, $event)" />
    </main>

    <MobileFolderDrawer
      :open="drawerOpen"
      @close="drawerOpen = false"
      @select="onDrawerSelect"
    />

    <!-- 全屏阅读器（fixed 覆盖层，由 viewerStore.isOpen 控制显隐） -->
    <MobileReader />
  </div>
</template>
