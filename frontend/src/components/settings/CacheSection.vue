<script setup lang="ts">
// 缓存管理区块（SPRINT7_TASK.md §5.11）：磁盘统计 + 后台缓存构建（预热级别/进度）+
// 孤儿/全部/按变体清理（均 AppModal 二次确认）。清理期间浏览会触发缓存重建（后端不加锁，自用场景）
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  cleanupAllCaches,
  cleanupOrphanCaches,
  cleanupVariantCaches,
  getCacheStats,
  getWarmStatus,
  setWarmLevel,
} from '../../services/appsettings.service'
import type { CacheStats, CleanupResult, WarmLevel, WarmStatus } from '../../types/appsettings'
import { formatFileSize } from '../../utils/format'
import AppModal from '../common/AppModal.vue'

const stats = ref<CacheStats | null>(null)
const loading = ref(false)
const cleaning = ref(false)
const notice = ref('')

// 二次确认弹窗目标：orphan / all / thumb / preview（后两者 = 按变体清空），null = 关闭
const confirmTarget = ref<'orphan' | 'all' | 'thumb' | 'preview' | null>(null)

// ---- 后台缓存构建（预热器） ----
const warm = ref<WarmStatus | null>(null)
const warmError = ref(false)
const settingLevel = ref(false)
let warmTimer: ReturnType<typeof setInterval> | null = null

// 级别选项（描述与后端 model/warm.go 上限表同步）
const WARM_LEVELS: Array<{ value: WarmLevel; label: string; desc: string }> = [
  { value: 'off', label: '关闭', desc: '不后台构建' },
  { value: 'minimal', label: '预热', desc: '每目录前 20 缩略图 + 前 5 预览' },
  { value: 'level1', label: '一级', desc: '每目录前 100 缩略图 + 前 50 预览' },
  { value: 'level2', label: '二级', desc: '每目录前 300 缩略图 + 前 150 预览' },
  { value: 'full', label: '完整', desc: '全部缩略图 + 全部预览' },
]

const phaseText = computed(() => {
  switch (warm.value?.phase) {
    case 'warming':
      return '预热中'
    case 'paused':
      return '已暂停（前台浏览中）'
    default:
      return '空闲'
  }
})

// 目录进度百分比（目录数为分母；无目录或未开始时 0）
const warmPercent = computed(() => {
  const w = warm.value
  if (!w || w.dirs_total <= 0) return 0
  return Math.min(100, Math.round((w.dirs_done / w.dirs_total) * 100))
})

// 当前级别描述文案
const currentLevelDesc = computed(() => WARM_LEVELS.find((o) => o.value === warm.value?.level)?.desc ?? '')

async function loadWarm() {
  try {
    warm.value = await getWarmStatus()
    warmError.value = false
  } catch {
    // 拉取失败不隐藏整个区块（避免"消失"误解），显示提示并等下轮轮询重试
    warmError.value = true
  }
}

async function changeLevel(level: WarmLevel) {
  if (settingLevel.value || warm.value?.level === level) return
  settingLevel.value = true
  try {
    warm.value = await setWarmLevel(level)
  } catch (e) {
    notice.value = e instanceof Error ? `设置预热级别失败：${e.message}` : '设置预热级别失败'
  } finally {
    settingLevel.value = false
  }
}

onMounted(() => {
  load()
  loadWarm()
  // 5s 轮询预热进度（设置页打开期间）
  warmTimer = setInterval(loadWarm, 5000)
})

onUnmounted(() => {
  if (warmTimer) clearInterval(warmTimer)
})

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
    let result: CleanupResult
    if (target === 'orphan') result = await cleanupOrphanCaches()
    else if (target === 'all') result = await cleanupAllCaches()
    else result = await cleanupVariantCaches(target)
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

    <!-- 后台缓存构建（预热器）：置于磁盘统计之上，优先呈现 -->
    <div v-if="warm" class="mt-3 rounded-lg border border-line bg-elevated/20 p-3">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-medium text-ink">后台缓存构建</h3>
        <span
          class="rounded-full px-2 py-0.5 text-xs"
          :class="warm.phase === 'warming' ? 'bg-accent/15 text-accent' : warm.phase === 'paused' ? 'bg-elevated text-muted' : 'bg-elevated text-faint'"
        >{{ phaseText }}</span>
      </div>
      <p class="mt-1 text-xs text-muted">
        闲时自动按级别构建缓存，前台浏览时自动暂停。
        <template v-if="!warm.enabled">后台预热已被环境变量 THUMB_WARMER=0 关闭，此处仅可调整级别（重启生效）。</template>
      </p>

      <div class="mt-2 flex flex-wrap gap-1.5">
        <button
          v-for="opt in WARM_LEVELS"
          :key="opt.value"
          type="button"
          :disabled="settingLevel"
          :title="opt.desc"
          class="rounded border px-2.5 py-1 text-xs transition-colors disabled:cursor-not-allowed disabled:opacity-60"
          :class="warm.level === opt.value
            ? 'border-accent bg-accent/10 text-accent'
            : 'border-line-strong bg-elevated text-body hover:border-line-hover hover:text-ink'"
          @click="changeLevel(opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>
      <p class="mt-1.5 text-xs text-faint">{{ currentLevelDesc }}</p>

      <template v-if="warm.dirs_total > 0">
        <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-line">
          <div class="h-full rounded-full bg-accent transition-all" :style="{ width: `${warmPercent}%` }" />
        </div>
        <p class="mt-1.5 text-xs text-muted">
          目录 {{ warm.dirs_done }}/{{ warm.dirs_total }}（{{ warmPercent }}%）· 图片 {{ warm.images_done }}/{{ warm.images_planned }}
          <template v-if="warm.phase !== 'idle' && warm.current_dir">· {{ warm.current_dir }}</template>
        </p>
      </template>
    </div>
    <p v-else-if="warmError" class="mt-3 text-xs text-muted">后台缓存构建状态加载失败，将自动重试</p>

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

    <div class="mt-3 flex flex-wrap items-center gap-2">
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
        @click="confirmTarget = 'thumb'"
      >
        清空缩略图缓存
      </button>
      <button
        type="button"
        :disabled="cleaning"
        class="rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover hover:text-ink disabled:cursor-not-allowed disabled:opacity-60"
        @click="confirmTarget = 'preview'"
      >
        清空预览缓存
      </button>
      <button
        type="button"
        :disabled="cleaning"
        class="rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-accent-text transition-colors hover:border-line-hover disabled:cursor-not-allowed disabled:opacity-60"
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
      :open="confirmTarget === 'preview'"
      title="清空预览缓存"
      message="将删除全部预览图缓存（缩略图不受影响），查看大图时按新规格重新生成。是否继续？"
      confirm-text="清空"
      danger
      @confirm="doCleanup"
      @close="confirmTarget = null"
    />
    <AppModal
      :open="confirmTarget === 'thumb'"
      title="清空缩略图缓存"
      message="将删除全部缩略图缓存（预览图不受影响），浏览目录网格时重新生成。是否继续？"
      confirm-text="清空"
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
