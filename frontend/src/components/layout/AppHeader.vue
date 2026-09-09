<script setup lang="ts">
// 顶栏：Logo + 当前文件夹路径 + 主题切换 + 视图模式切换 + SortMenu + PageMode 入口（docs/UI_DESIGN.md §1/§3）
// Sprint 7 新增：设置页齿轮入口 + 登录态区域（auth enabled 时显示，见 SPRINT7_TASK.md §5.10）
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useFolderStore } from '../../stores/folder'
import { useFavoritesStore } from '../../stores/favorites'
import { useSettingsStore } from '../../stores/settings'
import { useSpreadStore } from '../../stores/spread'
import { useAuthStore } from '../../stores/auth'
import SortMenu from '../sorting/SortMenu.vue'
import ViewModeSwitch from '../common/ViewModeSwitch.vue'
import SpreadControls from '../viewer/SpreadControls.vue'
import HelpModal from '../common/HelpModal.vue'
import { useTheme } from '../../composables/useTheme'
import { APP_ICON, APP_NAME } from '../../constants/app'

const router = useRouter()
const store = useFolderStore()
const favorites = useFavoritesStore()
const settingsStore = useSettingsStore()
const spreadStore = useSpreadStore()
const auth = useAuthStore()
const { theme, toggleTheme } = useTheme()

// PageMode 下拉面板开关
const spreadOpen = ref(false)
// 帮助弹窗开关
const helpOpen = ref(false)

// 打开/切换目录时加载该目录已保存的排序设置（immediate 覆盖首屏根目录）；
// 收藏夹虚拟视图（currentPath 为显示用文本）无目录设置，跳过加载
watch(
  () => store.currentPath,
  (path) => {
    if (!store.favoritesView) settingsStore.load(path)
  },
  { immediate: true },
)

// 退出登录：服务端删除会话后回浏览页（回到游客态）
async function logout() {
  try {
    await auth.logout()
  } finally {
    router.push('/')
  }
}
</script>

<template>
  <header class="flex h-14 shrink-0 items-center gap-3 border-b border-line px-4">
    <!-- 品牌 Logo：直接引用 favicon.svg（深底 + 白色相框 + 琥珀色山形/搁板），与浏览器标签页图标完全一致 -->
    <span class="flex shrink-0 items-center gap-2">
      <img :src="APP_ICON" :alt="APP_NAME" class="h-7 w-7 shrink-0" />
      <span class="text-lg font-semibold text-ink">{{ APP_NAME }}</span>
    </span>
    <span class="min-w-0 flex-1 truncate font-mono text-sm text-muted">{{ store.currentPath }}</span>
    <!-- 主题切换：浅色/深色，localStorage 持久化（useTheme） -->
    <button
      type="button"
      :title="theme === 'dark' ? '切换浅色主题' : '切换深色主题'"
      class="flex h-8 shrink-0 items-center rounded border border-line-strong bg-panel px-2 text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
      @click="toggleTheme"
    >
      <svg v-if="theme === 'dark'" class="h-[18px] w-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
      </svg>
      <svg v-else class="h-[18px] w-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79Z" />
      </svg>
    </button>
    <!-- 视图模式切换：文件 / 图片 / 全图（localStorage 持久化；全图模式被覆盖层遮挡，工具栏内有同款控件） -->
    <ViewModeSwitch />
    <!-- 只看收藏：目录内筛选已收藏图片（文件网格与图片/全图模式均适用；收藏夹虚拟视图内不显示） -->
    <button
      v-if="favorites.available && !store.favoritesView"
      type="button"
      title="只看收藏"
      class="flex h-8 shrink-0 items-center gap-1 rounded border px-3 text-sm transition-colors"
      :class="store.favoritesOnly ? 'border-red-400/60 text-ink' : 'border-line-strong bg-panel text-body hover:border-line-hover hover:text-ink'"
      @click="store.favoritesOnly = !store.favoritesOnly"
    >
      <svg
        class="h-3.5 w-3.5"
        :class="store.favoritesOnly ? 'text-red-400' : ''"
        viewBox="0 0 24 24"
        :fill="store.favoritesOnly ? 'currentColor' : 'none'"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z" />
      </svg>
      只看收藏
    </button>
    <!-- 排序：收藏夹按收藏时间倒序展示，不适用排序 -->
    <SortMenu v-if="!store.favoritesView" class="shrink-0" />
    <!-- PageMode 入口：显示当前布局，展开 SpreadControls（布局/阅读方向/阈值/封面封底，docs/UI_DESIGN.md §3）；
         收藏夹强制单页帧化，隐藏入口避免向虚拟路径落库 -->
    <div v-if="!store.favoritesView" class="relative shrink-0">
      <button
        type="button"
        title="双页阅读设置"
        class="flex h-8 items-center rounded border border-line-strong bg-panel px-3 text-sm text-body transition-colors hover:border-line-hover"
        @click="spreadOpen = !spreadOpen"
      >
        阅读：{{ spreadStore.pageMode === 'spread' ? '双页' : '单页' }}
      </button>
      <!-- 透明全屏遮罩：点击任意处关闭（与 SortMenu 一致） -->
      <div v-if="spreadOpen" class="fixed inset-0 z-30" @click="spreadOpen = false" />
      <div v-if="spreadOpen" class="absolute right-0 top-full z-40 mt-1">
        <SpreadControls />
      </div>
    </div>
    <!-- 登录态区域：仅 auth enabled 时显示（SPRINT7_TASK.md §5.10；disabled 时无任何登录态元素） -->
    <template v-if="auth.enabled">
      <template v-if="auth.authenticated">
        <span class="mx-1 max-w-32 shrink-0 truncate text-sm text-muted" :title="auth.username ?? ''">
          {{ auth.username }}
        </span>
        <button
          type="button"
          class="h-8 shrink-0 rounded border border-line-strong bg-panel px-3 text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
          @click="logout"
        >
          退出
        </button>
      </template>
      <button
        v-else
        type="button"
        class="h-8 shrink-0 rounded border border-line-strong bg-panel px-3 text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
        @click="router.push('/login')"
      >
        登录
      </button>
    </template>
    <!-- 设置页入口：齿轮按钮（PROJECT_STRUCTURE.md §2 预留） -->
    <button
      type="button"
      title="设置"
      class="flex h-8 w-8 shrink-0 items-center justify-center rounded border border-line-strong bg-panel text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
      @click="router.push('/settings')"
    >
      <svg class="h-[18px] w-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.08a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h.08a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.08a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
      </svg>
    </button>
    <!-- 用法说明：视图模式 / 快捷键 / 排序速查；裸问号描边图标（不带圈），比字体字形更清晰 -->
    <button
      type="button"
      title="使用说明"
      class="flex h-8 w-8 shrink-0 items-center justify-center rounded border border-line-strong bg-panel text-body transition-colors hover:border-line-hover hover:text-ink"
      @click="helpOpen = true"
    >
      <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"></circle><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path><path d="M12 17h.01"></path>
      </svg>
    </button>
  </header>
  <HelpModal :open="helpOpen" @close="helpOpen = false" />
</template>
