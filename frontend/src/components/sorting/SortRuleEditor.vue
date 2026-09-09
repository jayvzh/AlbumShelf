<script setup lang="ts">
// 单条正则排序规则行编辑组件（SPRINT5_TASK §5.2）：defineModel 受控，不含请求逻辑
import type { SortRule } from '../../types/sort'

const rule = defineModel<SortRule>({ required: true })

const emit = defineEmits<{ remove: [] }>()
</script>

<template>
  <div class="flex items-center gap-2 text-sm">
    <label class="flex items-center gap-1 text-muted">
      Group
      <input
        v-model.number="rule.group"
        type="number"
        min="1"
        class="w-16 rounded border border-line-strong bg-elevated px-2 py-1 text-body focus:border-amber-400 focus:outline-none"
      />
    </label>
    <select
      v-model="rule.type"
      class="rounded border border-line-strong bg-elevated px-2 py-1 text-body focus:border-amber-400 focus:outline-none"
    >
      <option value="number">数值</option>
      <option value="string">文本</option>
    </select>
    <select
      v-model="rule.direction"
      class="rounded border border-line-strong bg-elevated px-2 py-1 text-body focus:border-amber-400 focus:outline-none"
    >
      <option value="asc">升序</option>
      <option value="desc">降序</option>
    </select>
    <button
      type="button"
      title="删除规则"
      class="ml-auto rounded px-2 py-1 text-faint transition-colors hover:bg-elevated hover:text-red-500"
      @click="emit('remove')"
    >
      ✕
    </button>
  </div>
</template>
