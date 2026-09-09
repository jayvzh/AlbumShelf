<script setup lang="ts">
// 双页阅读控制面板：布局 / 阅读方向 / WideRatio 阈值 / 封面封底单页开关（docs/UI_DESIGN.md §3）
// 全部变更经由 spreadStore.updateOptions 统一编排（重建帧 → 锚点重同步 → 持久化）
import { useSpreadStore } from '../../stores/spread'
import type { PageMode, ReadOrder } from '../../types/spread'

const spreadStore = useSpreadStore()

const PAGE_MODES: { value: PageMode; label: string }[] = [
  { value: 'single', label: '单页' },
  { value: 'spread', label: '双页拼页' },
]

const READ_ORDERS: { value: ReadOrder; label: string }[] = [
  { value: 'left_to_right', label: '从左到右' },
  { value: 'right_to_left', label: '从右到左' },
]

// 两态分段按钮样式（选中态 amber 高亮，与 ViewerToolbar 原图按钮一致）
function segClass(active: boolean): string {
  return active
    ? 'flex-1 rounded border border-amber-400/70 bg-amber-400/15 px-2 py-1 text-accent-text transition-colors'
    : 'flex-1 rounded border border-transparent bg-elevated px-2 py-1 text-body transition-colors hover:bg-elevated-hover'
}

// WideRatio：横图判定阈值（宽 > 高 × 阈值 视为横图），非法输入回落默认 1.0
function onWideRatioChange(e: Event) {
  const raw = Number((e.target as HTMLInputElement).value)
  const ratio = Number.isFinite(raw) && raw > 0 ? raw : 1.0
  spreadStore.updateOptions({ wideRatio: ratio })
}
</script>

<template>
  <div class="w-64 space-y-4 rounded-lg border border-line bg-panel/95 p-4 text-sm shadow-xl">
    <section>
      <div class="mb-1.5 text-xs text-faint">布局</div>
      <div class="flex gap-1">
        <button
          v-for="mode in PAGE_MODES"
          :key="mode.value"
          type="button"
          :class="segClass(spreadStore.pageMode === mode.value)"
          @click="spreadStore.updateOptions({ pageMode: mode.value })"
        >
          {{ mode.label }}
        </button>
      </div>
    </section>

    <section>
      <div class="mb-1.5 text-xs text-faint">阅读方向</div>
      <div class="flex gap-1">
        <button
          v-for="order in READ_ORDERS"
          :key="order.value"
          type="button"
          :class="segClass(spreadStore.readOrder === order.value)"
          @click="spreadStore.updateOptions({ readOrder: order.value })"
        >
          {{ order.label }}
        </button>
      </div>
    </section>

    <section>
      <div class="mb-1.5 text-xs text-faint">横图判定阈值（宽 &gt; 高 × 阈值）</div>
      <input
        type="number"
        class="w-full rounded border border-line-strong bg-elevated px-2 py-1 text-body focus:border-amber-400/70 focus:outline-none"
        :value="spreadStore.wideRatio"
        min="0.1"
        step="0.1"
        @change="onWideRatioChange"
      />
    </section>

    <section class="space-y-2">
      <label class="flex cursor-pointer items-center gap-2 text-body">
        <input
          type="checkbox"
          class="h-4 w-4 accent-amber-400"
          :checked="spreadStore.singleFirstPage"
          @change="spreadStore.updateOptions({ singleFirstPage: ($event.target as HTMLInputElement).checked })"
        />
        封面单独成页
      </label>
      <label class="flex cursor-pointer items-center gap-2 text-body">
        <input
          type="checkbox"
          class="h-4 w-4 accent-amber-400"
          :checked="spreadStore.singleLastPage"
          @change="spreadStore.updateOptions({ singleLastPage: ($event.target as HTMLInputElement).checked })"
        />
        封底单独成页
      </label>
    </section>
  </div>
</template>
