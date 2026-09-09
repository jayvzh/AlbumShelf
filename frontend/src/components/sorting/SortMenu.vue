<script setup lang="ts">
// 顶栏排序菜单：模式/方向即时生效（重新请求目录，不落库），"保存到当前目录"落库（docs/UI_DESIGN.md §1）
// regex 模式经 RegexEditor 配置后生效（SPRINT5_TASK §5.2）
import { computed, onUnmounted, ref } from 'vue'
import { useFolderStore } from '../../stores/folder'
import { useSettingsStore } from '../../stores/settings'
import type { SortDirection, SortMode } from '../../types/sort'
import RegexEditor from './RegexEditor.vue'

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

// 顶栏当前模式显示：regex 为已生效状态，不带菜单项的"打开编辑器"省略号
const currentModeLabel = computed(() => {
  if (settingsStore.sortMode === 'regex') return '正则'
  return modeOptions.find((o) => o.value === settingsStore.sortMode)?.label ?? settingsStore.sortMode
})
const directionArrow = computed(() => (settingsStore.sortDirection === 'asc' ? '↑' : '↓'))

// 保存反馈：成功短暂显示"已保存"，失败行内红字
const saved = ref(false)
const saveError = ref('')
let savedTimer: ReturnType<typeof setTimeout> | undefined

function toggle() {
  open.value = !open.value
  saveError.value = ''
}

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

// 保存当前排序到 settingsStore.path（即当前目录）；失败向上抛由行内红字提示
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
  <div class="relative">
    <button
      type="button"
      title="排序方式"
      class="flex h-8 shrink-0 items-center gap-1 rounded border border-line-strong bg-panel px-3 text-sm text-body transition-colors hover:border-line-hover"
      @click="toggle"
    >
      <span>排序：{{ currentModeLabel }} {{ directionArrow }}</span>
    </button>

    <!-- 透明全屏遮罩：点击任意处关闭 -->
    <div v-if="open" class="fixed inset-0 z-30" @click="close" />

    <div
      v-if="open"
      class="absolute right-0 top-full z-40 mt-1 w-44 rounded border border-line-strong bg-panel py-1 shadow-lg"
    >
      <button
        v-for="opt in modeOptions"
        :key="opt.value"
        type="button"
        class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors hover:bg-elevated"
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
        class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors hover:bg-elevated"
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
        class="w-full px-3 py-1.5 text-left text-sm transition-colors hover:bg-elevated disabled:opacity-50"
        :class="saved ? 'text-emerald-500' : 'text-accent-text'"
        @click="onSave"
      >
        {{ saved ? '已保存' : settingsStore.saving ? '保存中…' : '保存到当前目录' }}
      </button>
      <p v-if="saveError" class="px-3 py-1 text-xs text-red-500">保存失败：{{ saveError }}</p>
    </div>

    <RegexEditor :open="showRegexEditor" @close="showRegexEditor = false" />
  </div>
</template>
