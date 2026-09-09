<script setup lang="ts">
// 设置页（SPRINT7_TASK.md §5.11）：纵向五区块——项目信息 / 私有目录 / 缓存管理 / 配置导入导出 / 账户与系统信息
// auth enabled 且未登录时由路由守卫拦截（→ /login）；disabled 时直接开放
import { useRouter } from 'vue-router'
import ProjectInfoSection from '../components/settings/ProjectInfoSection.vue'
import ProtectedFoldersSection from '../components/settings/ProtectedFoldersSection.vue'
import CacheSection from '../components/settings/CacheSection.vue'
import ConfigSection from '../components/settings/ConfigSection.vue'
import AccountSection from '../components/settings/AccountSection.vue'

const router = useRouter()
</script>

<template>
  <div class="flex h-screen flex-col bg-base">
    <header class="flex h-14 shrink-0 items-center gap-3 border-b border-line px-4">
      <button
        type="button"
        title="返回浏览"
        class="flex h-8 w-8 shrink-0 items-center justify-center rounded border border-line-strong bg-panel text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
        @click="router.push('/')"
      >
        ←
      </button>
      <h1 class="text-lg font-semibold text-ink">设置</h1>
    </header>

    <main class="mx-auto w-full max-w-2xl flex-1 space-y-4 overflow-y-auto p-4">
      <ProjectInfoSection />
      <ProtectedFoldersSection />
      <CacheSection />
      <ConfigSection />
      <AccountSection />
    </main>
  </div>
</template>
