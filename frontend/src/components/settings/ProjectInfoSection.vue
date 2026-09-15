<script setup lang="ts">
// 项目信息页脚（设置弹窗底部）：单行精简呈现 名称 / 版本 / 作者 / 仓库链接。
// 品牌字段统一取自 constants/app.ts；卡片样式与其他设置区块一致（rounded-lg border bg-panel）
import { APP_AUTHOR, APP_NAME, APP_REPO_URL, APP_VERSION } from '../../constants/app'
</script>

<template>
  <section class="rounded-lg border border-line bg-panel px-4 py-3">
    <p class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-faint">
      <span class="text-body">{{ APP_NAME }}</span>
      <span>v{{ APP_VERSION }}</span>
      <span>by {{ APP_AUTHOR }}</span>
      <a
        :href="APP_REPO_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="font-mono text-accent-text hover:underline"
        >GitHub</a
      >
    </p>
  </section>
</template>
