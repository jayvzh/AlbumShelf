<script setup lang="ts">
// 通用确认弹窗（SPRINT7_TASK.md §5.11）：缓存清理 / 配置导入等破坏性操作的二次确认。
// 与 HelpModal 一致：仅点击遮罩/按钮关闭，不监听 Esc（避免与大图浏览 Esc 冲突）
withDefaults(
  defineProps<{
    open: boolean
    title: string
    message: string
    confirmText?: string
    // 危险操作确认按钮显示强调色
    danger?: boolean
  }>(),
  { confirmText: '确认', danger: false },
)
const emit = defineEmits<{ confirm: []; close: [] }>()
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    @click.self="emit('close')"
  >
    <div class="w-80 max-w-[92vw] rounded-lg border border-line-strong bg-panel shadow-xl">
      <header class="border-b border-line px-4 py-3">
        <h2 class="text-sm font-medium text-ink">{{ title }}</h2>
      </header>
      <div class="px-4 py-4">
        <p class="text-sm leading-relaxed text-body">{{ message }}</p>
      </div>
      <footer class="flex justify-end gap-2 border-t border-line px-4 py-3">
        <button
          type="button"
          class="rounded border border-line-strong bg-panel px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
          @click="emit('close')"
        >
          取消
        </button>
        <button
          type="button"
          :class="
            danger
              ? 'rounded border border-accent-text bg-elevated px-3 py-1.5 text-sm text-accent-text transition-colors hover:bg-elevated-hover'
              : 'rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-ink transition-colors hover:border-line-hover'
          "
          @click="emit('confirm')"
        >
          {{ confirmText }}
        </button>
      </footer>
    </div>
  </div>
</template>
