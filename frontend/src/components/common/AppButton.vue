<script setup lang="ts">
// 通用按钮：default 常规按钮 / icon 方形图标按钮，深色 zinc 风格
withDefaults(
  defineProps<{
    variant?: 'default' | 'icon'
    title?: string
    disabled?: boolean
  }>(),
  { variant: 'default', title: undefined, disabled: false },
)
</script>

<template>
  <button
    type="button"
    :title="title"
    :disabled="disabled"
    :class="[
      variant === 'icon'
        ? 'flex h-8 w-8 items-center justify-center rounded text-body'
        : 'flex h-8 items-center rounded px-3 text-sm text-body',
      'bg-elevated transition-colors hover:bg-elevated-hover disabled:cursor-not-allowed disabled:bg-elevated disabled:text-faint',
    ]"
  >
    <slot />
  </button>
</template>
