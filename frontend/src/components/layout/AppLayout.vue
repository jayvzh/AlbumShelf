<script setup lang="ts">
// 三栏布局：顶部 Header + 左侧 Sidebar + 右侧 main（slot 传入）
import AppHeader from './AppHeader.vue'
import AppSidebar from './AppSidebar.vue'
</script>

<template>
  <div class="flex h-screen flex-col bg-base">
    <AppHeader />
    <div class="flex min-h-0 flex-1">
      <AppSidebar />
      <main class="min-w-0 flex-1 overflow-auto bg-base">
        <slot />
      </main>
    </div>
  </div>
</template>
