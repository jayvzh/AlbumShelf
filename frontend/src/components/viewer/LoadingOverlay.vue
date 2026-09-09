<script setup lang="ts">
// 加载遮罩：切换图片时避免白屏闪烁（docs/UI_DESIGN.md §6）
defineProps<{ visible: boolean }>()
</script>

<template>
  <Transition
    enter-active-class="transition-opacity duration-150"
    enter-from-class="opacity-0"
    leave-active-class="transition-opacity duration-150"
    leave-to-class="opacity-0"
  >
    <div
      v-if="visible"
      class="absolute inset-0 z-20 flex items-center justify-center bg-base/60"
    >
      <div class="flex items-center gap-3">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-muted border-t-ink" />
        <span class="text-sm text-muted">加载中…</span>
      </div>
    </div>
  </Transition>
</template>
