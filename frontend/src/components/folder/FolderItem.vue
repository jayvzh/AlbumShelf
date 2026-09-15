<script setup lang="ts">
// 目录树：递归渲染，节点点击 emit select 由本组件统一调 store action；
// 钉住/私有操作收进右键菜单（FolderContextMenu）；行上静态标记：钉住 = 行尾实心图钉（随字色），私有 = 名称前置小锁 + 黄色名
import { computed, ref } from 'vue'
import { useFolderStore } from '../../stores/folder'
import { useAuthStore } from '../../stores/auth'
import { useProtectedStore } from '../../stores/protected'
import { useQuickAccessStore } from '../../stores/quickAccess'
import type { FolderItem as FolderItemType } from '../../types/folder'
import FolderContextMenu from './FolderContextMenu.vue'

const props = defineProps<{
  folder: FolderItemType
}>()
const emit = defineEmits<{ select: [path: string] }>()

const store = useFolderStore()
const auth = useAuthStore()
const protectedStore = useProtectedStore()
const quickAccessStore = useQuickAccessStore()

// 展开状态收敛到 folder store（localStorage 持久化，刷新后恢复）；根节点恒展开
const expanded = computed(() => props.folder.path === '/' || !!store.expandedPaths[props.folder.path])

const children = computed(() => store.children[props.folder.path])
// 未加载（undefined）时先显示箭头；已加载且无子目录则隐藏
const expandable = computed(() => children.value === undefined || children.value.length > 0)
const active = computed(() => store.currentPath === props.folder.path)
// 媒体库根（path 固定 /）：库锚点无需折叠，以文件夹图标占位与收藏夹节点对齐
const isRoot = computed(() => props.folder.path === '/')

// 私有锁：仅登录态可操作，根目录不可设为私有（后端拒绝）
const canLock = computed(() => auth.enabled && auth.authenticated && !isRoot.value)
const isProtected = computed(() => protectedStore.isProtected(props.folder.path))
const toggling = computed(() => protectedStore.toggling.has(props.folder.path))
// 切换失败时短暂提示
const flashError = ref(false)

// 快捷访问钉住：与 canLock 同口径（登录态、非根目录）
const canPin = computed(() => auth.enabled && auth.authenticated && !isRoot.value)
const isPinned = computed(() => quickAccessStore.has(props.folder.path))

// 右键菜单：仅登录态且存在可操作项时打开
const menu = ref<{ x: number; y: number } | null>(null)
const menuItems = computed(() => {
  const items: { key: string; label: string }[] = []
  if (canPin.value) items.push({ key: 'pin', label: isPinned.value ? '取消固定' : '固定到快捷访问' })
  if (canLock.value) items.push({ key: 'lock', label: isProtected.value ? '设为公开' : '设为私有' })
  return items
})

function openMenu(e: MouseEvent) {
  if (menuItems.value.length === 0) return
  menu.value = { x: e.clientX, y: e.clientY }
}

function onMenuSelect(key: string) {
  if (key === 'pin') togglePin()
  else if (key === 'lock') toggleLock()
}

async function togglePin() {
  if (!canPin.value) return
  try {
    await quickAccessStore.toggle(props.folder.path)
  } catch {
    flashError.value = true
    setTimeout(() => (flashError.value = false), 2000)
  }
}

async function toggleLock() {
  if (!canLock.value || toggling.value) return
  const ok = await protectedStore.toggle(props.folder.path)
  if (!ok) {
    flashError.value = true
    setTimeout(() => (flashError.value = false), 2000)
  }
}

// 展开/折叠走 store action（含按需加载子目录与持久化）
function toggle() {
  return store.toggleExpanded(props.folder.path)
}
</script>

<template>
  <div>
    <div
      class="relative flex items-center gap-1 rounded px-1 py-1"
      :class="active ? 'bg-elevated text-ink' : 'text-body hover:bg-panel'"
      :data-path="folder.path"
      @contextmenu.prevent="openMenu"
    >
      <button
        v-if="expandable && !isRoot"
        class="flex h-4 w-4 shrink-0 items-center justify-center text-faint transition-transform"
        :class="expanded ? 'rotate-90' : ''"
        @click.stop="toggle"
      >
        <svg
          class="h-3 w-3"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m9 18 6-6-6-6" />
        </svg>
      </button>
      <span v-else-if="!isRoot" class="h-4 w-4 shrink-0" />
      <span v-else class="flex h-4 w-4 shrink-0 items-center justify-center">
        <svg
          class="h-3.5 w-3.5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z" />
        </svg>
      </span>
      <!-- 名称占满剩余宽度；私有目录名用黄色并前置小锁标识 -->
      <button
        class="flex min-w-0 flex-1 items-center gap-1 text-left text-sm"
        :class="isProtected ? 'text-protected' : ''"
        @click="emit('select', folder.path)"
      >
        <svg
          v-if="isProtected"
          class="h-3 w-3 shrink-0"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <rect x="3" y="11" width="18" height="11" rx="2" />
          <path d="M7 11V7a5 5 0 0 1 10 0v4" />
        </svg>
        <span class="min-w-0 truncate">{{ folder.name }}</span>
      </button>
      <!-- 静态状态标记：已钉住行尾实心图钉（随字色），不参与交互 -->
      <span v-if="isPinned" class="shrink-0" title="已固定到快捷访问">
        <svg
          class="h-3 w-3"
          viewBox="0 0 24 24"
          fill="currentColor"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M12 17v5" />
          <path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a2 2 0 0 0 0-4H8a2 2 0 0 0 0 4h1z" />
        </svg>
      </span>
    </div>
    <!-- 右键菜单：钉住/私有切换，关闭后清空坐标 -->
    <FolderContextMenu
      v-if="menu"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menu = null"
    />
    <!-- 切换失败短暂提示（行内红字） -->
    <p v-if="flashError" class="px-2 text-xs text-accent-text">操作失败，请重试</p>
    <!-- 递归渲染子节点：每层缩进 6px，左缘引导线填充缩进空白，select 事件逐层向上转发 -->
    <div v-show="expanded" class="ml-1.5 border-l border-line">
      <FolderItem
        v-for="child in children ?? []"
        :key="child.path"
        :folder="child"
        @select="emit('select', $event)"
      />
    </div>
  </div>
</template>
