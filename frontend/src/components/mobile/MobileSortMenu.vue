<script setup lang="ts">
// 移动端排序菜单：顶栏排序图标按钮 → 底部 action sheet 弹出
// 功能对齐桌面 SortMenu：模式/方向即时生效（重新请求目录，不落库），"保存到当前目录"落库
// regex 模式经共用的 RegexEditor 配置后生效（其弹窗 max-w-[92vw]，窄屏可用）；收藏夹视图不挂载本组件
import { onUnmounted, ref } from 'vue'
import { useFolderStore } from '../../stores/folder'
import { useSettingsStore } from '../../stores/settings'
import type { SortDirection, SortMode } from '../../types/sort'
import RegexEditor from '../sorting/RegexEditor.vue'

const folderStore = useFolderStore()
const settingsStore = useSettingsStore()

const open = ref(false)
const showRegexEditor = ref(false)

// 菜单暴露 6 种模式；regex 项打开 RegexEditor 而非直接生效
const modeOptions: { value: SortMode; label: string }[] = [
  { value: 'filename', label: '文件名' },
  { value: 'natural', label: '自然数字' },
  { value: 'modified_time', label: '修改时间' },
  { value: 'created_time', label: '创建时间' },
  { value: 'file_size', label: '文件大小' },
  { value: 'regex', label: '自定义正则…' },
]
const directionOptions: { value: SortDirection; label: string }[] = [
  { value: 'asc', label: '升序' },
  { value: 'desc', label: '降序' },
]

// 保存反馈：成功短暂显示"已保存"，失败行内红字
const saved = ref(false)
const saveError = ref('')
let savedTimer: ReturnType<typeof setTimeout> | undefined

function close() {
  open.value = false
  saveError.value = ''
}

// 排序变更：本地即时生效并按新排序重新请求当前目录内容，不落库
function selectMode(mode: SortMode) {
  if (mode === 'regex') {
    // regex 经编辑器配置，应用时才生效
    close()
    showRegexEditor.value = true
    return
  }
  applySort(mode, settingsStore.sortDirection)
  close()
}

// 方向切换不关菜单，便于连续切换
function selectDirection(direction: SortDirection) {
  applySort(settingsStore.sortMode, direction)
}

function applySort(mode: SortMode, direction: SortDirection) {
  settingsStore.setSort(mode, direction)
  folderStore.openFolder(folderStore.currentPath, { sort: mode, direction })
}

// 保存当前排序到 settingsStore.path（即当前目录）；失败行内红字提示
async function onSave() {
  saveError.value = ''
  try {
    await settingsStore.save()
    saved.value = true
    clearTimeout(savedTimer)
    savedTimer = setTimeout(() => {
      saved.value = false
    }, 1500)
  } catch (e) {
    saveError.value = e instanceof Error ? e.message : String(e)
  }
}

onUnmounted(() => clearTimeout(savedTimer))
</script>

<template>
  <!-- 顶栏排序按钮：上下箭头图标 + 当前方向 -->
  <button
    type="button"
    title="排序方式"
    aria-label="排序方式"
    class="flex h-9 w-9 shrink-0 items-center justify-center rounded text-body"
    @click="open = true"
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
      <path d="m21 16-4 4-4-4" />
      <path d="M17 20V4" />
      <path d="m3 8 4-4 4 4" />
      <path d="M7 4v16" />
    </svg>
  </button>

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
        @click="close"
      />
    </Transition>
    <!-- 底部 action sheet：上滑/下滑 -->
    <Transition
      enter-active-class="transition-transform duration-200 ease-out"
      enter-from-class="translate-y-full"
      leave-active-class="transition-transform duration-200 ease-in"
      leave-to-class="translate-y-full"
    >
      <div
        v-if="open"
        class="fixed inset-x-0 bottom-0 z-50 rounded-t-xl border-t border-line-strong bg-panel pb-[max(env(safe-area-inset-bottom),0.5rem)] shadow-xl"
      >
        <div class="flex h-12 shrink-0 items-center justify-between border-b border-line px-2">
          <span class="px-2 text-sm font-medium text-ink">排序方式</span>
          <button
            type="button"
            title="关闭"
            aria-label="关闭排序菜单"
            class="flex h-11 w-11 items-center justify-center text-faint"
            @click="close"
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
              <path d="M18 6 6 18" />
              <path d="m6 6 12 12" />
            </svg>
          </button>
        </div>

        <div class="py-1 text-sm">
          <button
            v-for="opt in modeOptions"
            :key="opt.value"
            type="button"
            class="flex h-11 w-full items-center gap-3 px-4 text-left transition-colors active:bg-elevated"
            :class="opt.value === settingsStore.sortMode ? 'text-ink' : 'text-body'"
            @click="selectMode(opt.value)"
          >
            <span class="w-4 shrink-0 text-accent-text">{{ opt.value === settingsStore.sortMode ? '✓' : '' }}</span>
            {{ opt.label }}
          </button>

          <div class="my-1 border-t border-line" />

          <button
            v-for="opt in directionOptions"
            :key="opt.value"
            type="button"
            class="flex h-11 w-full items-center gap-3 px-4 text-left transition-colors active:bg-elevated"
            :class="opt.value === settingsStore.sortDirection ? 'text-ink' : 'text-body'"
            @click="selectDirection(opt.value)"
          >
            <span class="w-4 shrink-0 text-accent-text">{{ opt.value === settingsStore.sortDirection ? '✓' : '' }}</span>
            {{ opt.label }}
          </button>

          <div class="my-1 border-t border-line" />

          <button
            type="button"
            :disabled="settingsStore.saving"
            class="flex h-11 w-full items-center px-4 text-left transition-colors active:bg-elevated disabled:opacity-50"
            :class="saved ? 'text-emerald-500' : 'text-accent-text'"
            @click="onSave"
          >
            {{ saved ? '已保存' : settingsStore.saving ? '保存中…' : '保存到当前目录' }}
          </button>
          <p v-if="saveError" class="px-4 py-1 text-xs text-red-500">保存失败：{{ saveError }}</p>
        </div>
      </div>
    </Transition>
  </Teleport>

  <RegexEditor :open="showRegexEditor" @close="showRegexEditor = false" />
</template>
