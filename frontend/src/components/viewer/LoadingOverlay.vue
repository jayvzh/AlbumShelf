<script setup lang="ts">
// 加载指示器：透明背景不遮挡画布——渐进式 JPEG 预览图传输中边收边显（模糊→清晰），
// 整面遮罩会挡住这一过程；仅悬浮小型半透明 pill 提示加载中（图片流式渲染在四周可见）
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
      class="pointer-events-none absolute inset-0 z-20 flex items-center justify-center"
    >
      <div class="flex items-center gap-3 rounded-full bg-base/60 px-4 py-2">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-muted border-t-ink" />
        <span class="text-sm text-muted">加载中…</span>
      </div>
    </div>
  </Transition>
</template>
