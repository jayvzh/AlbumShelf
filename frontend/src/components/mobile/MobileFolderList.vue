<script setup lang="ts">
// 移动端目录列表行：递归渲染（同桌面 FolderItem 结构，触控优化：行高 44px、箭头热区 44px）
// 复用 folderStore（children 缓存 / expandedPaths 持久化 / toggleExpanded 按需加载）
// 钉住/私有操作收进长按菜单（与桌面共用 FolderContextMenu，500ms 长按触发）；
// 行上静态标记同桌面：钉住 = 行尾实心图钉（随字色），私有 = 名称前置小锁 + 黄色名
import { computed, onUnmounted, ref } from 'vue'
import { useFolderStore } from '../../stores/folder'
import { useAuthStore } from '../../stores/auth'
import { useProtectedStore } from '../../stores/protected'
import { useQuickAccessStore } from '../../stores/quickAccess'
import type { FolderItem as FolderItemType } from '../../types/folder'
import FolderContextMenu from '../folder/FolderContextMenu.vue'

const props = defineProps<{
  folder: FolderItemType
}>()
const emit = defineEmits<{ select: [path: string] }>()

const store = useFolderStore()
const auth = useAuthStore()
const protectedStore = useProtectedStore()
const quickAccessStore = useQuickAccessStore()

// 展开状态收敛到 folder store（与桌面共享持久化）；根节点恒展开
const expanded = computed(() => props.folder.path === '/' || !!store.expandedPaths[props.folder.path])
const children = computed(() => store.children[props.folder.path])
// 未加载（undefined）时先显示箭头；已加载且无子目录则隐藏
const expandable = computed(() => children.value === undefined || children.value.length > 0)
const active = computed(() => store.currentPath === props.folder.path)
// 根节点不可折叠，以文件夹图标占位
const isRoot = computed(() => props.folder.path === '/')

// 图钉 / 私有锁门控同桌面：仅登录态可操作，根目录不支持（后端拒绝）
const canPin = computed(() => auth.enabled && auth.authenticated && !isRoot.value)
const isPinned = computed(() => quickAccessStore.has(props.folder.path))
const canLock = computed(() => auth.enabled && auth.authenticated && !isRoot.value)
const isProtected = computed(() => protectedStore.isProtected(props.folder.path))
const toggling = computed(() => protectedStore.toggling.has(props.folder.path))
// 切换失败时短暂红字提示
const flashError = ref(false)

// 长按菜单：与桌面右键菜单共用 FolderContextMenu；500ms 长按触发，移动/抬指取消
const menu = ref<{ x: number; y: number } | null>(null)
const menuItems = computed(() => {
  const items: { key: string; label: string }[] = []
  if (canPin.value) items.push({ key: 'pin', label: isPinned.value ? '取消固定' : '固定到快捷访问' })
  if (canLock.value) items.push({ key: 'lock', label: isProtected.value ? '设为公开' : '设为私有' })
  return items
})

function openMenuAt(x: number, y: number) {
  if (menuItems.value.length === 0 || menu.value) return
  menu.value = { x, y }
}

let pressTimer: ReturnType<typeof setTimeout> | undefined

function onTouchStart(e: TouchEvent) {
  const t = e.touches[0]
  if (!t) return
  const x = t.clientX
  const y = t.clientY
  pressTimer = setTimeout(() => openMenuAt(x, y), 500)
}
// 滚动/抬指/打断均取消长按
function cancelPress() {
  clearTimeout(pressTimer)
}

// Android 长按会触发原生 contextmenu：阻止默认，未弹出时按触点直接打开（已弹出则幂等）
function onContextMenu(e: MouseEvent) {
  e.preventDefault()
  openMenuAt(e.clientX, e.clientY)
}

function onMenuSelect(key: string) {
  if (key === 'pin') togglePin()
  else if (key === 'lock') toggleLock()
}
onUnmounted(() => clearTimeout(pressTimer))

function flashFailure() {
  flashError.value = true
  setTimeout(() => (flashError.value = false), 2000)
}

async function togglePin() {
  if (!canPin.value) return
  try {
    await quickAccessStore.toggle(props.folder.path)
  } catch {
    flashFailure()
  }
}

async function toggleLock() {
  if (!canLock.value || toggling.value) return
  const ok = await protectedStore.toggle(props.folder.path)
  if (!ok) flashFailure()
}

// 点击目录：打开目录并向上转发（页壳据此收起抽屉）
function onSelect() {
  store.openFolder(props.folder.path)
  emit('select', props.folder.path)
}
</script>

<template>
  <div>
    <div
      class="relative flex select-none items-center [-webkit-touch-callout:none]"
      :class="active ? 'bg-elevated text-ink' : 'text-body'"
      @touchstart="onTouchStart"
      @touchmove="cancelPress"
      @touchend="cancelPress"
      @touchcancel="cancelPress"
      @contextmenu="onContextMenu"
    >
      <!-- 前置图标列统一 w-11（44px）：箭头 / 叶子占位 / 根文件夹图标同列居中，
           与抽屉收藏夹、快捷访问行图标列对齐；叶子也占位避免加载后文字左跳 -->
      <!-- 展开箭头：44px 独立热区 -->
      <button
        v-if="expandable && !isRoot"
        type="button"
        class="flex h-11 w-11 shrink-0 items-center justify-center text-faint"
        @click.stop="store.toggleExpanded(folder.path)"
      >
        <svg
          class="h-4 w-4 transition-transform"
          :class="expanded ? 'rotate-90' : ''"
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
      <span v-else-if="!isRoot" class="h-11 w-11 shrink-0" />
      <span v-else class="flex h-11 w-11 shrink-0 items-center justify-center">
        <svg
          class="h-4 w-4"
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
        type="button"
        class="flex h-11 min-w-0 flex-1 items-center gap-1.5 text-left text-sm"
        :class="isProtected ? 'text-protected' : ''"
        @click="onSelect"
      >
        <svg
          v-if="isProtected"
          class="h-4 w-4 shrink-0"
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
      <span v-if="isPinned" class="flex h-11 w-9 shrink-0 items-center justify-center" title="已固定到快捷访问">
        <svg
          class="h-4 w-4"
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
    <!-- 长按菜单：钉住/私有切换，关闭后清空坐标 -->
    <FolderContextMenu
      v-if="menu"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menu = null"
    />
    <!-- 切换失败短暂红字提示（不展开为弹层，避免抽屉跳动过强） -->
    <p v-if="flashError" class="pl-3 text-xs leading-5 text-accent-text">操作失败，请重试</p>
    <!-- 递归渲染子节点：缩进 + 引导线，select 逐层向上转发 -->
    <div v-show="expanded" class="ml-3 border-l border-line">
      <MobileFolderList
        v-for="child in children ?? []"
        :key="child.path"
        :folder="child"
        @select="emit('select', $event)"
      />
    </div>
  </div>
</template>
