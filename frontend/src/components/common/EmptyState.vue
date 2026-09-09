<script setup lang="ts">
// 空状态 / 错误提示占位组件
defineProps<{ message: string }>()
</script>

<template>
  <div class="flex h-full flex-col items-center justify-center gap-3 p-8 text-center">
    <svg
      class="h-10 w-10 text-faint"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path
        d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z"
      />
    </svg>
    <p class="text-sm text-faint">{{ message }}</p>
  </div>
</template>
