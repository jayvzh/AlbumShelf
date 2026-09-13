<script setup lang="ts">
// 移动端顶栏（min-h-12 + safe-area）：汉堡（目录抽屉）+ 品牌（三级收纳）+ 弱化目录名 + 排序 + 网格/阅读切换 + 账户头像 + 设置入口
// 注意必须用 min-h-12 而非 h-12：小程序 WebView 沉浸式下 safe-area-inset-top 可达 47~59px（计入 border-box），
// 固定 48px 会导致 44px 按钮溢出 header 下边界；min-h 允许顶栏随安全区撑高
// 移动端舍弃：主题切换 / 只看收藏 / 双页阅读设置 / 帮助 / 全路径文本（退出登录走账户头像菜单）
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useFolderStore } from '../../stores/folder'
import { useViewerStore } from '../../stores/viewer'
import { useAuthStore } from '../../stores/auth'
import { APP_BASE_NAME, APP_ICON, APP_NAME } from '../../constants/app'
import { useMediaQuery } from '../../composables/useMediaQuery'
import MobileSortMenu from './MobileSortMenu.vue'
import SettingsModal from '../common/SettingsModal.vue'

const emit = defineEmits<{ menu: [] }>()

const router = useRouter()
const folderStore = useFolderStore()
const viewerStore = useViewerStore()
const auth = useAuthStore()

// 账户头像下拉菜单开关
const accountOpen = ref(false)
// 设置弹窗开关
const settingsOpen = ref(false)

// 头像首字母：用户名首字符（空用户名兜底 ?）
const initial = computed(() => (auth.username ?? '').trim().charAt(0).toUpperCase() || '?')

// 退出登录：服务端删除会话后回浏览页（回到游客态，同桌面 AppHeader）
async function logout() {
  try {
    await auth.logout()
  } finally {
    router.push('/')
  }
}

// 标题区三级收纳（与桌面 AppHeader 的"后缀 → 主名 → 藏品牌"顺序一致）：
//   1. 后缀（・NAS图集馆）仅在根目录且视口宽裕（≥420px）时随全名显示
//   2. 子目录/收藏夹：品牌主名 AlbumShelf 常驻 + 弱化样式目录名并存
//   3. 目录名超长：隐藏品牌文字（Logo 保留），目录名优先完整显示
// 当前目录名（末段）；根目录为空字符串
const dirName = computed(() => {
  if (folderStore.favoritesView) return '收藏夹'
  return folderStore.currentPath.split('/').filter(Boolean).pop() ?? ''
})

// 目录名显示宽度估算（CJK ≈14px、其余 ≈8px）：超出品牌右侧可用区（约 130px）即视为超长
const dirNameWidth = computed(() =>
  [...dirName.value].reduce((w, c) => w + (c.charCodeAt(0) > 255 ? 14 : 8), 0),
)
const dirNameTooLong = computed(() => dirNameWidth.value > 130)

// 根目录全名是否放得下：主流手机 390pt 只够主名，Pro Max / 平板（≥420px）才显后缀
const wideEnough = useMediaQuery('(min-width: 420px)')

// 品牌文字：空 = 只留 Logo
const brandText = computed(() => {
  if (dirNameTooLong.value) return ''
  if (!dirName.value) return wideEnough.value ? APP_NAME : APP_BASE_NAME
  return APP_BASE_NAME
})

// 阅读态 = 查看器打开（image/full 均由 MobileReader 呈现）
const reading = computed(() => viewerStore.isOpen)

// 网格/阅读两态切换（移动端舍弃桌面三段式中的嵌入 image/full 区分）
function toggleReading() {
  viewerStore.setMode(reading.value ? 'file' : 'image')
}
</script>

<template>
  <header
    class="flex min-h-12 shrink-0 items-center gap-1 border-b border-line bg-panel pl-1 pr-2 pt-[env(safe-area-inset-top)]"
  >
    <!-- 汉堡：打开目录抽屉 -->
    <button
      type="button"
      title="目录"
      class="flex h-11 w-11 shrink-0 items-center justify-center text-body"
      @click="emit('menu')"
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
        <path d="M4 6h16M4 12h16M4 18h16" />
      </svg>
    </button>
    <!-- 品牌 + 当前目录标题：品牌常驻（按宽度收纳后缀/整字），目录名弱化样式并存 -->
    <div class="flex h-11 min-w-0 flex-1 items-center gap-2">
      <img :src="APP_ICON" :alt="APP_NAME" class="h-6 w-6 shrink-0" />
      <span v-if="brandText" class="shrink-0 text-base font-semibold text-ink">{{ brandText }}</span>
      <span
        v-if="dirName"
        class="min-w-0 truncate text-sm font-normal text-muted"
        :title="folderStore.favoritesView ? '收藏夹' : folderStore.currentPath"
      >{{ dirName }}</span>
    </div>
    <!-- 排序：收藏夹按收藏时间倒序展示，不适用排序（同桌面 AppHeader） -->
    <MobileSortMenu v-if="!folderStore.favoritesView" />
    <!-- 网格/阅读切换 -->
    <button
      type="button"
      :title="reading ? '回到网格' : '开始阅读'"
      class="flex h-9 w-9 shrink-0 items-center justify-center rounded text-body"
      @click="toggleReading"
    >
      <svg
        v-if="reading"
        class="h-5 w-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <rect x="3" y="3" width="7" height="7" rx="1" />
        <rect x="14" y="3" width="7" height="7" rx="1" />
        <rect x="3" y="14" width="7" height="7" rx="1" />
        <rect x="14" y="14" width="7" height="7" rx="1" />
      </svg>
      <svg
        v-else
        class="h-5 w-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M2 4h6a4 4 0 0 1 4 4v12a3 3 0 0 0-3-3H2z" />
        <path d="M22 4h-6a4 4 0 0 0-4 4v12a3 3 0 0 1 3-3h7z" />
      </svg>
    </button>
    <!-- 账户头像：登录模式开启时显示（点击展开小菜单，主流形态）；
         已登录 = 用户名首字母圆头像 + 退出登录；未登录 = 人形轮廓 + 登录入口；
         登录系统关闭时隐藏（无登录态概念） -->
    <div v-if="auth.enabled" class="relative shrink-0">
      <button
        type="button"
        title="账户"
        class="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-full border text-base transition-colors"
        :class="auth.authenticated
          ? 'border-line-strong bg-panel font-semibold'
          : 'border-line-strong bg-panel text-body'"
        :style="auth.authenticated ? { color: 'var(--app-accent-text)' } : undefined"
        @click="accountOpen = !accountOpen"
      >
        <span v-if="auth.authenticated">{{ initial }}</span>
        <svg
          v-else
          class="h-[19px] w-[19px]"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
          <circle cx="12" cy="7" r="4" />
        </svg>
      </button>
      <!-- 透明全屏遮罩：点击任意处关闭 -->
      <div v-if="accountOpen" class="fixed inset-0 z-30" @click="accountOpen = false" />
      <!-- 账户下拉菜单（header 下方右对齐小卡片，面板/菜单项样式与 SortMenu 一致） -->
      <div v-if="accountOpen" class="absolute right-0 top-full z-40 mt-1 w-44 overflow-hidden rounded border border-line-strong bg-panel py-1 shadow-lg">
        <template v-if="auth.authenticated">
          <div class="border-b border-line px-3 py-2">
            <div class="truncate text-sm font-medium text-ink" :title="auth.username ?? ''">{{ auth.username }}</div>
            <div class="text-xs text-muted">已登录</div>
          </div>
          <button
            type="button"
            class="flex h-11 w-full items-center gap-2 px-3 text-sm text-body active:bg-elevated"
            @click="accountOpen = false; folderStore.openFavorites()"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z" />
            </svg>
            收藏夹
          </button>
          <button
            type="button"
            class="flex h-11 w-full items-center gap-2 px-3 text-sm text-body active:bg-elevated"
            @click="accountOpen = false; settingsOpen = true"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.08a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h.08a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.08a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            设置
          </button>
          <button
            type="button"
            class="flex h-11 w-full items-center gap-2 px-3 text-sm text-body active:bg-elevated"
            @click="accountOpen = false; logout()"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <path d="m16 17 5-5-5-5" />
              <path d="M21 12H9" />
            </svg>
            退出登录
          </button>
        </template>
        <template v-else>
          <div class="border-b border-line px-3 py-2 text-sm text-muted">未登录</div>
          <button
            type="button"
            class="flex h-11 w-full items-center gap-2 px-3 text-sm text-body active:bg-elevated"
            @click="accountOpen = false; router.push('/login')"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
              <path d="m10 17 5-5-5-5" />
              <path d="M3 12h12" />
            </svg>
            登录
          </button>
        </template>
      </div>
    </div>
    <!-- 设置页入口：仅登录系统关闭时独立显示；登录模式下收入账户头像菜单（需登录后可见，同桌面 AppHeader） -->
    <button
      v-if="!auth.enabled"
      type="button"
      title="设置"
      class="flex h-9 w-9 shrink-0 items-center justify-center rounded text-body"
      @click="settingsOpen = true"
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
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.08a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h.08a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.08a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
      </svg>
    </button>
  </header>
  <SettingsModal :open="settingsOpen" @close="settingsOpen = false" />
</template>
