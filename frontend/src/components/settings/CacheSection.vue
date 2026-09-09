<script setup lang="ts">
// 缓存管理区块（SPRINT7_TASK.md §5.11）：磁盘统计 + 孤儿清理 / 全部清理（均 AppModal 二次确认）。
// 清理期间浏览会触发缓存重建（后端不加锁，自用场景）
import { onMounted, ref } from 'vue'
import {
  cleanupAllCaches,
  cleanupOrphanCaches,
  getCacheStats,
} from '../../services/appsettings.service'
import type { CacheStats, CleanupResult } from '../../types/appsettings'
import { formatFileSize } from '../../utils/format'
import AppModal from '../common/AppModal.vue'

const stats = ref<CacheStats | null>(null)
const loading = ref(false)
const cleaning = ref(false)
const notice = ref('')

// 二次确认弹窗目标：orphan = 孤儿清理，all = 全部清理，null = 关闭
const confirmTarget = ref<'orphan' | 'all' | null>(null)

onMounted(load)

async function load() {
  loading.value = true
  try {
    stats.value = await getCacheStats()
  } catch (e) {
    notice.value = e instanceof Error ? e.message : '加载统计失败'
  } finally {
    loading.value = false
  }
}

async function doCleanup() {
  const target = confirmTarget.value
  if (!target || cleaning.value) return
  cleaning.value = true
  notice.value = ''
  try {
    const result: CleanupResult =
      target === 'orphan' ? await cleanupOrphanCaches() : await cleanupAllCaches()
    notice.value = `已清理 ${result.removed_files} 个文件（${formatFileSize(result.removed_bytes)}）`
    await load()
  } catch (e) {
    notice.value = e instanceof Error ? `清理失败：${e.message}` : '清理失败，请重试'
  } finally {
    cleaning.value = false
    confirmTarget.value = null
  }
}
</script>

<template>
  <section class="rounded-lg border border-line bg-panel p-4">
    <h2 class="text-sm font-medium text-ink">缓存管理</h2>
    <p class="mt-1 text-xs text-muted">缩略图与预览图缓存（磁盘统计）。清理期间浏览对应图片会触发重建。</p>

    <p v-if="loading" class="mt-3 text-sm text-faint">加载中…</p>

    <template v-else-if="stats">
      <div class="mt-3 grid grid-cols-2 gap-2">
        <div class="rounded border border-line bg-elevated/40 px-3 py-2">
          <p class="text-xs text-muted">缩略图（thumb）</p>
          <p class="mt-0.5 text-sm text-ink">{{ stats.thumb.count }} 个 · {{ formatFileSize(stats.thumb.bytes) }}</p>
        </div>
        <div class="rounded border border-line bg-elevated/40 px-3 py-2">
          <p class="text-xs text-muted">预览图（preview）</p>
          <p class="mt-0.5 text-sm text-ink">{{ stats.preview.count }} 个 · {{ formatFileSize(stats.preview.bytes) }}</p>
        </div>
      </div>
    </template>

    <div class="mt-3 flex items-center gap-3">
      <button
        type="button"
        :disabled="cleaning"
        class="rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover hover:text-ink disabled:cursor-not-allowed disabled:opacity-60"
        @click="confirmTarget = 'orphan'"
      >
        清理孤儿缓存
      </button>
      <button
        type="button"
        :disabled="cleaning"
        class="rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover hover:text-ink disabled:cursor-not-allowed disabled:opacity-60"
        @click="confirmTarget = 'all'"
      >
        清理全部缓存
      </button>
      <span v-if="notice" class="text-xs text-muted">{{ notice }}</span>
    </div>

    <AppModal
      :open="confirmTarget === 'orphan'"
      title="清理孤儿缓存"
      message="将删除源文件已不存在与索引外的遗留缓存文件，正常缓存不受影响。是否继续？"
      confirm-text="清理"
      danger
      @confirm="doCleanup"
      @close="confirmTarget = null"
    />
    <AppModal
      :open="confirmTarget === 'all'"
      title="清理全部缓存"
      message="将清空全部缩略图与预览图缓存，浏览图片时会重新生成。是否继续？"
      confirm-text="清理"
      danger
      @confirm="doCleanup"
      @close="confirmTarget = null"
    />
  </section>
</template>
