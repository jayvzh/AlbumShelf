<script setup lang="ts">
// 设置弹窗：与 HelpModal 同形态（点击遮罩/✕ 关闭，不监听 Esc：避免与大图浏览的 Esc 逐级返回冲突）；
// 纵向五区块复用设置 Section 组件（项目信息 / 私有目录 / 缓存管理 / 配置导入导出 / 账户与系统信息）
import ProjectInfoSection from '../settings/ProjectInfoSection.vue'
import ProtectedFoldersSection from '../settings/ProtectedFoldersSection.vue'
import CacheSection from '../settings/CacheSection.vue'
import ConfigSection from '../settings/ConfigSection.vue'
import AccountSection from '../settings/AccountSection.vue'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    @click.self="emit('close')"
  >
    <div
      class="relative flex max-h-[85vh] w-[640px] max-w-[92vw] flex-col rounded-lg border border-line-strong bg-panel shadow-xl"
    >
      <!-- 关闭按钮：绝对定位置于右上角（不随内容滚动） -->
      <button
        type="button"
        aria-label="关闭"
        class="absolute right-3 top-3 z-10 rounded px-2 py-1 text-faint transition-colors hover:bg-elevated hover:text-body"
        @click="emit('close')"
      >
        ✕
      </button>

      <header class="border-b border-line px-4 py-3">
        <h2 class="text-sm font-medium text-ink">设置</h2>
      </header>

      <div class="flex-1 space-y-4 overflow-y-auto px-4 pt-4 pb-10">
        <ProjectInfoSection />
        <ProtectedFoldersSection />
        <CacheSection />
        <ConfigSection />
        <AccountSection />
      </div>
    </div>
  </div>
</template>
