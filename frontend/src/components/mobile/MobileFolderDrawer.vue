<script setup lang="ts">
// 移动端目录抽屉：fixed 左侧滑入 + 全屏遮罩，宽 min(78vw, 300px)
// 顶部虚拟节点：收藏夹 + 快捷访问（门控同桌面 FolderTree），下方递归目录列表
import { useFolderStore } from '../../stores/folder'
import { useFavoritesStore } from '../../stores/favorites'
import { useQuickAccessStore } from '../../stores/quickAccess'
import MobileFolderList from './MobileFolderList.vue'
import type { FolderItem as FolderItemType } from '../../types/folder'

defineProps<{
  open: boolean
}>()
const emit = defineEmits<{
  close: []
  select: [path: string]
}>()

const store = useFolderStore()
const favorites = useFavoritesStore()
const quickAccess = useQuickAccessStore()

const root: FolderItemType = { name: '图集目录', path: '/' }

// 快捷访问列表项显示末级目录名（同桌面 FolderTree）
function baseName(path: string): string {
  const parts = path.split('/').filter(Boolean)
  return parts.length > 0 ? parts[parts.length - 1] : '/'
}

// 点击快捷访问项：打开目录并向上转发（页壳据此收起抽屉）
function openQuick(path: string) {
  store.openFolder(path)
  emit('select', path)
}
</script>

<template>
  <Teleport to="body">
    <!-- 遮罩：点击关闭 -->
    <Transition
      enter-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      leave-active-class="transition-opacity duration-200"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="fixed inset-0 z-40 bg-black/50"
        @click="emit('close')"
      />
    </Transition>
    <!-- 抽屉面板：左侧滑入/滑出 -->
    <Transition
      enter-active-class="transition-transform duration-200 ease-out"
      enter-from-class="-translate-x-full"
      leave-active-class="transition-transform duration-200 ease-in"
      leave-to-class="-translate-x-full"
    >
      <aside
        v-if="open"
        class="fixed inset-y-0 left-0 z-50 flex w-[min(78vw,300px)] flex-col border-r border-line bg-panel pt-[env(safe-area-inset-top)]"
      >
        <div class="flex h-11 shrink-0 items-center px-4 text-xs font-medium tracking-wide text-faint">
          目录
        </div>
        <nav class="flex-1 overflow-y-auto pb-[env(safe-area-inset-bottom)] text-sm">
          <!-- 收藏夹虚拟节点：favorites 功能不可用（auth 开启且未登录）时隐藏。
               前置图标统一 w-11 图标列（与目录树箭头/根图标同列居中，文字起点对齐） -->
          <button
            v-if="favorites.available"
            type="button"
            class="flex h-11 w-full items-center"
            :class="store.favoritesView ? 'bg-elevated text-ink' : 'text-body'"
            @click="store.openFavorites(); emit('select', 'favorites')"
          >
            <span class="flex h-11 w-11 shrink-0 items-center justify-center">
              <svg
                class="h-4 w-4"
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
            <span class="min-w-0 flex-1 truncate pr-2 text-left text-sm">收藏夹</span>
            <span
              v-if="favorites.count > 0"
              class="mr-3 shrink-0 rounded-full bg-elevated px-2 py-0.5 text-xs tabular-nums text-muted"
            >{{ favorites.count }}</span>
          </button>
          <!-- 快捷访问虚拟节点：门控同桌面（auth 开启且未登录时隐藏）；展开状态与桌面共享持久化 -->
          <div v-if="quickAccess.available">
            <button
              type="button"
              class="flex h-11 w-full items-center text-body active:bg-elevated"
              @click="quickAccess.toggleExpanded()"
            >
              <span class="flex h-11 w-11 shrink-0 items-center justify-center">
                <svg
                  class="h-4 w-4"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M12 17v5" />
                  <path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a2 2 0 0 0 0-4H8a2 2 0 0 0 0 4h1z" />
                </svg>
              </span>
              <span class="min-w-0 flex-1 truncate pr-2 text-left text-sm">快捷访问</span>
              <span
                v-if="quickAccess.count > 0"
                class="mr-3 shrink-0 rounded-full bg-elevated px-2 py-0.5 text-xs tabular-nums text-muted"
              >{{ quickAccess.count }}</span>
            </button>
            <div v-show="quickAccess.expanded" class="ml-3 border-l border-line">
              <p v-if="quickAccess.list.length === 0" class="py-2 pl-11 pr-3 text-xs text-faint">暂无固定目录</p>
              <div
                v-for="path in quickAccess.list"
                :key="path"
                class="flex h-11 items-center"
                :class="store.currentPath === path ? 'bg-elevated text-ink' : 'text-body'"
              >
                <!-- w-11 占位：文字与同层目录子行（缩进 + 箭头列后）起点对齐 -->
                <span class="h-11 w-11 shrink-0" />
                <button
                  type="button"
                  class="h-11 min-w-0 flex-1 truncate pr-1 text-left text-sm"
                  @click="openQuick(path)"
                >
                  {{ baseName(path) }}
                </button>
                <!-- 取消固定：X 热区 36pt，乐观更新失败由 store 自动回滚 -->
                <button
                  type="button"
                  title="取消固定"
                  class="mr-2 flex h-11 w-9 shrink-0 items-center justify-center text-faint active:text-ink"
                  @click.stop="quickAccess.toggle(path)"
                >
                  <svg
                    class="h-4 w-4"
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
              </div>
            </div>
          </div>
          <MobileFolderList :folder="root" @select="emit('select', $event)" />
        </nav>
      </aside>
    </Transition>
  </Teleport>
</template>
