<script setup lang="ts">
// 目录树：递归渲染，节点点击 emit select 由本组件统一调 store action
import { computed, ref } from 'vue'
import { useFolderStore } from '../../stores/folder'
import { useAuthStore } from '../../stores/auth'
import { useProtectedStore } from '../../stores/protected'
import type { FolderItem as FolderItemType } from '../../types/folder'

const props = withDefaults(
  defineProps<{
    folder: FolderItemType
    defaultExpanded?: boolean
  }>(),
  { defaultExpanded: false },
)
const emit = defineEmits<{ select: [path: string] }>()

const store = useFolderStore()
const auth = useAuthStore()
const protectedStore = useProtectedStore()

// 展开状态为组件局部 UI 状态
const expanded = ref(props.defaultExpanded)

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

async function toggleLock() {
  if (!canLock.value || toggling.value) return
  const ok = await protectedStore.toggle(props.folder.path)
  if (!ok) {
    flashError.value = true
    setTimeout(() => (flashError.value = false), 2000)
  }
}

async function toggle() {
  expanded.value = !expanded.value
  if (expanded.value && !store.children[props.folder.path]) {
    await store.loadChildren(props.folder.path)
  }
}
</script>

<template>
  <div>
    <div
      class="group flex items-center gap-1 rounded px-1 py-1"
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
      <span v-else class="flex h-4 w-4 shrink-0 items-center justify-center text-faint">
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
      <button class="min-w-0 flex-1 truncate text-left text-sm" @click="emit('select', folder.path)">
        {{ folder.name }}
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
