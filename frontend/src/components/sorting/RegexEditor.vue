<script setup lang="ts">
// 正则排序编辑器（SPRINT5_TASK §5.2 / UI_DESIGN §6）：
// 输入即校验（JS 即时红框，后端 INVALID_REGEX 权威）+ 300ms 防抖实时预览三列展示。
// 数据链路：previewSort → settings/folder store，组件内不发原始 fetch。
import { computed, onUnmounted, ref, watch } from 'vue'
import { ApiError } from '../../services/api'
import { previewSort } from '../../services/folder.service'
import { useFolderStore } from '../../stores/folder'
import { useSettingsStore } from '../../stores/settings'
import type { SortPreviewResponse, SortRule } from '../../types/sort'
import SortRuleEditor from './SortRuleEditor.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const folderStore = useFolderStore()
const settingsStore = useSettingsStore()

const pattern = ref('')
const rules = ref<SortRule[]>([])
const serverError = ref('') // 后端 INVALID_REGEX 或其他请求错误
const preview = ref<SortPreviewResponse | null>(null)
const previewing = ref(false)

// 每次打开从 settings store 预填当前配置
watch(
  () => props.open,
  (open) => {
    if (!open) return
    pattern.value = settingsStore.regexPattern
    rules.value = settingsStore.regexRules.map((r) => ({ ...r }))
    serverError.value = ''
    preview.value = null
  },
  { immediate: true },
)

// JS 即时语法校验；Go RE2 与 JS 语法存在差异（反向引用/lookahead 等），
// 此处仅做即时提示，合法性以后端 INVALID_REGEX 为权威。
const jsError = computed(() => {
  if (!pattern.value) return ''
  try {
    // eslint-disable-next-line no-new
    new RegExp(pattern.value)
    return ''
  } catch (e) {
    return e instanceof Error ? e.message : String(e)
  }
})

const inputError = computed(() => jsError.value || serverError.value)

// regex 为空或有未通过的校验时禁用应用
const canApply = computed(() => pattern.value.trim() !== '' && !inputError.value)

// 预览请求的规则：无规则且整体降序时合成单规则，与 GET /folders 的简写行为保持一致
// （POST /sort/preview 请求体无顶层 direction 字段）
function effectiveRules(): SortRule[] | undefined {
  if (rules.value.length === 0 && settingsStore.sortDirection === 'desc') {
    return [{ group: 1, type: 'number', direction: 'desc' }]
  }
  return rules.value.length ? rules.value : undefined
}

async function runPreview() {
  serverError.value = ''
  if (!pattern.value || jsError.value) {
    preview.value = null
    return
  }
  previewing.value = true
  try {
    preview.value = await previewSort({
      files: folderStore.images.map((img) => img.name),
      mode: 'regex',
      regex: pattern.value,
      rules: effectiveRules(),
    })
  } catch (e) {
    preview.value = null
    serverError.value =
      e instanceof ApiError ? (e.code === 'INVALID_REGEX' ? `正则无效：${e.message}` : `${e.code}: ${e.message}`) : String(e)
  } finally {
    previewing.value = false
  }
}

let debounceTimer: ReturnType<typeof setTimeout> | undefined
watch([pattern, rules], () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(runPreview, 300)
}, { deep: true })

function addRule() {
  rules.value.push({ group: rules.value.length + 1, type: 'number', direction: 'asc' })
}

function removeRule(index: number) {
  rules.value.splice(index, 1)
}

// 应用：写 settings store 本地状态 → 带 regex 参数重新请求当前目录 → 关闭
function apply() {
  if (!canApply.value) return
  settingsStore.setRegex(pattern.value, rules.value)
  settingsStore.setSort('regex', settingsStore.sortDirection)
  folderStore.openFolder(folderStore.currentPath, {
    sort: 'regex',
    direction: settingsStore.sortDirection,
    regex: pattern.value,
    regexRules: rules.value,
  })
  emit('close')
}

// 预览 Groups 列展示：filename → "10, 2"
const groupText = computed(() => {
  const map = new Map(preview.value?.matches.map((m) => [m.filename, m.groups.join(', ')]) ?? [])
  return (name: string) => map.get(name) ?? '—'
})

onUnmounted(() => clearTimeout(debounceTimer))
</script>

<template>
  <div
    v-if="props.open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    @click.self="emit('close')"
  >
    <div
      class="flex max-h-[85vh] w-[680px] max-w-[92vw] flex-col rounded-lg border border-line-strong bg-panel shadow-xl"
    >
      <header class="flex items-center justify-between border-b border-line px-4 py-3">
        <h2 class="text-sm font-medium text-ink">正则排序</h2>
        <button
          type="button"
          class="rounded px-2 py-1 text-faint transition-colors hover:bg-elevated hover:text-body"
          @click="emit('close')"
        >
          ✕
        </button>
      </header>

      <div class="flex-1 space-y-4 overflow-y-auto px-4 py-3">
        <!-- 正则输入：即时校验 -->
        <div>
          <label class="mb-1 block text-xs text-muted">正则表达式（捕获组用于排序）</label>
          <input
            v-model="pattern"
            type="text"
            spellcheck="false"
            placeholder="例如：chapter(\d+)_page(\d+)"
            class="w-full rounded border bg-elevated px-3 py-2 font-mono text-sm text-body focus:outline-none"
            :class="inputError ? 'border-red-500 focus:border-red-500' : 'border-line-strong focus:border-amber-400'"
          />
          <p v-if="inputError" class="mt-1 text-xs text-red-500">{{ inputError }}</p>
        </div>

        <!-- 规则编辑：SortRuleEditor 行列表 -->
        <div>
          <div class="mb-1 flex items-center justify-between">
            <span class="text-xs text-muted">排序规则（按顺序逐条比较）</span>
            <button
              type="button"
              class="rounded border border-line-strong px-2 py-0.5 text-xs text-body transition-colors hover:border-line-hover"
              @click="addRule"
            >
              + 添加规则
            </button>
          </div>
          <div class="space-y-1">
            <SortRuleEditor
              v-for="(_, i) in rules"
              :key="i"
              v-model="rules[i]"
              @remove="removeRule(i)"
            />
            <p v-if="rules.length === 0" class="text-xs text-faint">
              未添加规则时按第 1 个捕获组数值{{ settingsStore.sortDirection === 'desc' ? '降序' : '升序' }}
            </p>
          </div>
        </div>

        <!-- 实时预览三列展示 -->
        <div>
          <div class="mb-1 flex items-center justify-between">
            <span class="text-xs text-muted">预览（基于当前目录 {{ folderStore.images.length }} 个文件）</span>
            <span v-if="previewing" class="text-xs text-faint">预览中…</span>
          </div>
          <div v-if="preview" class="max-h-64 overflow-y-auto rounded border border-line">
            <div class="sticky top-0 grid grid-cols-[2.5rem_1fr_1fr] gap-2 bg-elevated px-2 py-1 text-xs text-muted">
              <span>#</span>
              <span>排序后</span>
              <span>Groups</span>
            </div>
            <div
              v-for="(name, i) in preview.sorted"
              :key="name"
              class="grid grid-cols-[2.5rem_1fr_1fr] gap-2 px-2 py-1 text-xs text-body odd:bg-panel"
            >
              <span class="text-faint">{{ i + 1 }}</span>
              <span class="truncate font-mono">{{ name }}</span>
              <span class="truncate font-mono text-accent-text/80">{{ groupText(name) }}</span>
            </div>
            <div v-if="preview.unmatched.length" class="border-t border-line px-2 py-1">
              <p class="py-1 text-xs text-faint">未匹配（排最后，保持原序）</p>
              <p
                v-for="name in preview.unmatched"
                :key="name"
                class="truncate py-0.5 font-mono text-xs text-faint"
              >
                {{ name }}
              </p>
            </div>
          </div>
          <p v-else-if="!inputError && !previewing" class="text-xs text-faint">
            {{ folderStore.images.length ? '输入正则后自动预览' : '当前目录没有图片文件' }}
          </p>
        </div>
      </div>

      <footer class="flex justify-end gap-2 border-t border-line px-4 py-3">
        <button
          type="button"
          class="rounded border border-line-strong px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover"
          @click="emit('close')"
        >
          取消
        </button>
        <button
          type="button"
          :disabled="!canApply"
          class="rounded bg-amber-500 px-3 py-1.5 text-sm text-zinc-900 transition-colors hover:bg-amber-400 disabled:cursor-not-allowed disabled:opacity-50"
          @click="apply"
        >
          应用
        </button>
      </footer>
    </div>
  </div>
</template>
