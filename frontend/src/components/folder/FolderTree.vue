<script setup lang="ts">
// 目录树入口：顶部虚拟「收藏夹」节点 + 根节点 "Images"（恒展开）；启动时恢复上次会话（当前目录与展开状态）
import { onMounted, watch } from 'vue'
import { useFolderStore } from '../../stores/folder'
import { useFavoritesStore } from '../../stores/favorites'
import { useAuthStore } from '../../stores/auth'
import { useProtectedStore } from '../../stores/protected'
import { usePathHint } from '../../composables/usePathHint'
import FolderItem from './FolderItem.vue'
import type { FolderItem as FolderItemType } from '../../types/folder'

const store = useFolderStore()
const favorites = useFavoritesStore()
const auth = useAuthStore()
const protectedStore = useProtectedStore()

const root: FolderItemType = { name: 'Images', path: '/' }

// 悬停目录树行 / 文件卡片（带 data-path）时，左下角胶囊条显示完整路径（委托与状态在 usePathHint）
const { hoveredPath } = usePathHint()

// 登录后加载私有目录列表（供目录树锁按钮与设置页共用）；退出登录时清空
watch(
  () => auth.authenticated,
  (authed) => {
    if (authed) protectedStore.load()
    else protectedStore.paths = new Set()
  },
  { immediate: true },
)

// 初始加载统一由此处负责（BrowserPage 不再重复触发，避免双重请求）；
// restore 恢复上次会话的当前目录与展开状态，无记录/失效时回退根目录
onMounted(() => {
  store.restore()
})
</script>

<template>
  <nav class="text-sm">
    <!-- 收藏夹虚拟节点：favorites 功能不可用（auth 开启且未登录）时整体隐藏 -->
    <button
      v-if="favorites.available"
      type="button"
      class="flex w-full items-center gap-1 rounded px-1 py-1"
      :class="store.favoritesView ? 'bg-elevated text-ink' : 'text-body hover:bg-panel'"
      @click="store.openFavorites()"
    >
      <span class="flex h-4 w-4 shrink-0 items-center justify-center">
        <svg
          class="h-3.5 w-3.5"
          viewBox="0 0 24 24"
          :fill="store.favoritesView ? 'currentColor' : 'none'"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z" />
        </svg>
      </span>
      <span class="min-w-0 flex-1 truncate text-left text-sm">收藏夹</span>
      <span
        v-if="favorites.count > 0"
        class="shrink-0 rounded-full bg-panel px-1.5 text-xs tabular-nums text-muted"
      >{{ favorites.count }}</span>
    </button>
    <FolderItem
      :folder="root"
      @select="(path: string) => store.openFolder(path)"
    />
  </nav>
  <!-- 路径提示条：固定视口左下角，不受侧栏宽度限制；pointer-events-none 不挡交互 -->
  <Transition
    enter-active-class="transition-opacity duration-150"
    enter-from-class="opacity-0"
    leave-active-class="transition-opacity duration-150"
    leave-to-class="opacity-0"
  >
    <div
      v-if="hoveredPath"
      class="pointer-events-none fixed bottom-2 left-2 z-50 max-w-[60vw] truncate rounded border border-line bg-elevated/95 px-2 py-1 font-mono text-xs text-body shadow-lg"
    >
      {{ hoveredPath }}
    </div>
  </Transition>
</template>
