<script setup lang="ts">
// 目录树：递归渲染，节点点击 emit select 由本组件统一调 store action
import { computed, ref } from 'vue'
import { useFolderStore } from '../../stores/folder'
import { useAuthStore } from '../../stores/auth'
import { useProtectedStore } from '../../stores/protected'
import { useQuickAccessStore } from '../../stores/quickAccess'
import type { FolderItem as FolderItemType } from '../../types/folder'

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
// 叠加按钮占用宽度（w-4 × 2 + 按钮间距 gap-1），名称据此做右缘渐隐
const overlayWidth = computed(
  () => (canPin.value ? 16 : 0) + (canLock.value ? 16 : 0) + (canPin.value && canLock.value ? 4 : 0),
)
// 长目录名右缘按叠加跨度渐入透明（含省略号随尾部淡出），避免与叠加图标打架；
// mask 作用于文字自身 alpha，选中高亮行无需切换底色；短名不受影响
const nameMask = computed(() =>
  overlayWidth.value
    ? {
        WebkitMaskImage: `linear-gradient(to right, #000 calc(100% - ${overlayWidth.value + 8}px), transparent)`,
        maskImage: `linear-gradient(to right, #000 calc(100% - ${overlayWidth.value + 8}px), transparent)`,
      }
    : undefined,
)

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
      class="group relative flex items-center gap-1 rounded px-1 py-1"
      :class="active ? 'bg-elevated text-ink' : 'text-body hover:bg-panel'"
      :data-path="folder.path"
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
      <!-- 名称占满剩余宽度（按钮叠加不占文档流）；登录态下右缘渐隐防与图标打架 -->
      <button class="min-w-0 flex-1 truncate text-left text-sm" :style="nameMask" @click="emit('select', folder.path)">
        {{ folder.name }}
      </button>
      <!-- 钉/锁叠加层：绝对定位不占文档流（right-1 对齐行内边距），所有目录名称区等宽；
           已激活常驻，未激活仅 hover 虚化出现 -->
      <div class="absolute inset-y-0 right-1 flex items-center gap-1">
        <!-- 快捷访问钉住按钮：已钉住实心图钉 + 主文字色常驻；未钉住仅 hover 虚化出现 -->
        <button
          v-if="canPin"
          type="button"
          class="flex h-4 w-4 shrink-0 items-center justify-center transition-opacity hover:text-ink"
          :class="isPinned ? 'text-ink opacity-90' : 'text-faint opacity-0 group-hover:opacity-60'"
          :title="isPinned ? '取消固定' : '固定到快捷访问'"
          @click.stop="togglePin"
        >
          <svg
            class="h-3.5 w-3.5"
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
        </button>
        <!-- 私有锁按钮：已加锁空心锁 + 主文字色常驻；未加锁仅 hover 虚化出现；切换中禁用 -->
        <button
          v-if="canLock"
          type="button"
          class="flex h-4 w-4 shrink-0 items-center justify-center transition-opacity hover:text-ink"
          :class="[
            isProtected ? 'text-ink opacity-90' : 'text-faint opacity-0 group-hover:opacity-60',
            toggling ? 'cursor-not-allowed opacity-50' : '',
          ]"
          :title="isProtected ? '设为公开' : '设为私有'"
          :disabled="toggling"
          @click.stop="toggleLock"
        >
          <svg
            class="h-3.5 w-3.5"
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
        </button>
      </div>
    </div>
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
