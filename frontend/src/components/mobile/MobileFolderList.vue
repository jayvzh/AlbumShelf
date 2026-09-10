<script setup lang="ts">
// 移动端目录列表行：递归渲染（同桌面 FolderItem 结构，触控优化：行高 44px、箭头热区 44px）
// 复用 folderStore（children 缓存 / expandedPaths 持久化 / toggleExpanded 按需加载）
// 行内操作（门控同桌面 FolderItem：登录态、非根目录）：图钉 = 固定到快捷访问；锁 = 设为私有
// 触控无 hover：未激活图标弱化常驻（opacity-40，按下加强），激活态主色常驻
// 钉/锁按钮绝对定位叠加行右侧（不占文档流，目录名全宽）；名称右缘按图标跨度渐隐（mask）防打架
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
// 叠加按钮占用宽度（w-7 × 按钮数），名称据此做右缘渐隐
const overlayWidth = computed(() => (canPin.value ? 28 : 0) + (canLock.value ? 28 : 0))
// 长目录名右缘按叠加跨度渐入透明（含省略号随尾部淡出），避免与叠加图标打架；
// mask 作用于文字自身 alpha，选中高亮行无需切换底色；短名不受影响
const nameMask = computed(() =>
  overlayWidth.value
    ? {
        WebkitMaskImage: `linear-gradient(to right, #000 calc(100% - ${overlayWidth.value + 12}px), transparent)`,
        maskImage: `linear-gradient(to right, #000 calc(100% - ${overlayWidth.value + 12}px), transparent)`,
      }
    : undefined,
)
// 切换失败时短暂红字提示
const flashError = ref(false)

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
      class="relative flex items-center"
      :class="active ? 'bg-elevated text-ink' : 'text-body'"
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
      <!-- 名称占满剩余宽度（按钮叠加不占文档流）；登录态下右缘渐隐防与图标打架 -->
      <button
        type="button"
        class="h-11 min-w-0 flex-1 truncate text-left text-sm"
        :style="nameMask"
        @click="onSelect"
      >
        {{ folder.name }}
      </button>
      <!-- 钉/锁叠加层：绝对定位不占文档流，所有目录名称区等宽；
           已激活主色常驻，未激活弱化（按下加强），保持触屏可发现性 -->
      <div class="absolute inset-y-0 right-0 flex">
        <!-- 图钉：固定到快捷访问。已固定主色常驻，未固定弱化（按下加强） -->
        <button
          v-if="canPin"
          type="button"
          class="flex h-11 w-7 shrink-0 items-center justify-center transition-opacity"
          :class="isPinned ? 'text-ink' : 'text-faint opacity-40 active:opacity-100'"
          :title="isPinned ? '取消固定' : '固定到快捷访问'"
          @click.stop="togglePin"
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
            <path d="M12 17v5" />
            <path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a2 2 0 0 0 0-4H8a2 2 0 0 0 0 4h1z" />
          </svg>
        </button>
        <!-- 私有锁：已加锁主色常驻，未加锁弱化（按下加强）；切换中禁用防竞态 -->
        <button
          v-if="canLock"
          type="button"
          class="flex h-11 w-7 shrink-0 items-center justify-center transition-opacity"
          :class="[
            isProtected ? 'text-ink' : 'text-faint opacity-40 active:opacity-100',
            toggling ? 'cursor-not-allowed opacity-50' : '',
          ]"
          :title="isProtected ? '设为公开' : '设为私有'"
          :disabled="toggling"
          @click.stop="toggleLock"
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
            <rect x="3" y="11" width="18" height="11" rx="2" />
            <path d="M7 11V7a5 5 0 0 1 10 0v4" />
          </svg>
        </button>
      </div>
    </div>
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
