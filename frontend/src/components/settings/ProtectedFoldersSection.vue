<script setup lang="ts">
// 私有目录区块（SPRINT7_TASK.md §5.11）：本地编辑列表（添加/删除）+「保存」全量替换。
// 数据来自 useProtectedStore，与目录树悬浮锁共用同一份状态，互相实时同步。
// 非法路径由后端 Resolve 校验，保存失败时按错误码提示（INVALID_PATH）
import { onMounted, ref, watch } from 'vue'
import { useProtectedStore } from '../../stores/protected'
import { ApiError } from '../../services/api'

const protectedStore = useProtectedStore()

const items = ref<string[]>([])
const newPath = ref('')
const loading = ref(false)
const saving = ref(false)
const notice = ref('') // 保存结果提示（成功/失败）
const loadError = ref('')

onMounted(load)

// 目录树切换私有状态时，同步刷新本地编辑列表
watch(
  () => protectedStore.pathList,
  (list) => {
    // 保存中不覆盖（避免用户正在编辑的内容被乐观更新回写打断）
    if (!saving.value) items.value = [...list]
  },
)

async function load() {
  loading.value = true
  loadError.value = ''
  notice.value = ''
  try {
    await protectedStore.load()
    items.value = [...protectedStore.pathList]
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function addPath() {
  const p = newPath.value.trim()
  if (!p) return
  if (items.value.includes(p)) {
    notice.value = `路径 ${p} 已在列表中`
    return
  }
  notice.value = ''
  items.value.push(p)
  newPath.value = ''
}

function removePath(p: string) {
  notice.value = ''
  items.value = items.value.filter((x) => x !== p)
}

async function save() {
  if (saving.value) return
  saving.value = true
  notice.value = ''
  try {
    await protectedStore.replace(items.value)
    items.value = [...protectedStore.pathList]
    notice.value = '保存成功'
  } catch (e) {
    if (e instanceof ApiError) {
      notice.value =
        e.code === 'INVALID_PATH' ? `路径无效或不存在：${e.message}` : `保存失败：${e.message}`
    } else {
      notice.value = '保存失败，请重试'
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="rounded-lg border border-line bg-panel p-4">
    <h2 class="text-sm font-medium text-ink">私有目录</h2>
    <p class="mt-1 text-xs text-muted">
      列表中的目录（含子级）对游客完全隐藏，登录后可见。也可在左侧目录树悬停目录，点击锁图标快速切换。路径为图片根目录相对路径（如 /Private）。
    </p>

    <p v-if="loading" class="mt-3 text-sm text-faint">加载中…</p>
    <p v-else-if="loadError" class="mt-3 text-sm text-accent-text">{{ loadError }}</p>

    <template v-else>
      <ul v-if="items.length" class="mt-3 space-y-1">
        <li
          v-for="p in items"
          :key="p"
          class="flex items-center justify-between rounded border border-line bg-elevated/40 px-3 py-1.5"
        >
          <span class="truncate font-mono text-sm text-body">{{ p }}</span>
          <button
            type="button"
            class="shrink-0 rounded px-2 py-0.5 text-xs text-muted transition-colors hover:bg-elevated hover:text-ink"
            title="移除"
            @click="removePath(p)"
          >
            删除
          </button>
        </li>
      </ul>
      <p v-else class="mt-3 text-sm text-faint">暂无私有目录</p>

      <div class="mt-3 flex gap-2">
        <input
          v-model="newPath"
          type="text"
          placeholder="/Private"
          class="min-w-0 flex-1 rounded border border-line-strong bg-elevated px-3 py-1.5 font-mono text-sm text-ink outline-none transition-colors focus:border-line-hover"
          @keyup.enter="addPath"
        />
        <button
          type="button"
          class="shrink-0 rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
          @click="addPath"
        >
          添加
        </button>
      </div>

      <div class="mt-3 flex items-center gap-3">
        <button
          type="button"
          :disabled="saving"
          class="rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-ink transition-colors hover:border-line-hover disabled:cursor-not-allowed disabled:opacity-60"
          @click="save"
        >
          {{ saving ? '保存中…' : '保存' }}
        </button>
        <span v-if="notice" class="text-xs text-muted">{{ notice }}</span>
      </div>
    </template>
  </section>
</template>
