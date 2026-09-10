<script setup lang="ts">
// 项目信息区块（设置弹窗顶部）：名称 / 作者 / 版本 / 仓库网址（无图标，名称弱化为普通标题样式）
// 品牌字段统一取自 constants/app.ts；卡片样式与其他设置区块一致（rounded-lg border bg-panel）
import { APP_AUTHOR, APP_NAME, APP_REPO_URL, APP_VERSION } from '../../constants/app'
</script>

<template>
  <section class="rounded-lg border border-line bg-panel p-4">
    <!-- 名称行：普通标题样式，与其他区块标题同级观感 -->
    <h2 class="text-sm font-medium text-body">{{ APP_NAME }}</h2>

    <!-- 元信息：细分隔线与品牌行区分；dt 列宽 w-24 与 AccountSection 的 dl 对齐 -->
    <dl class="mt-4 space-y-1.5 border-t border-line pt-3">
      <div class="flex gap-2">
        <dt class="w-24 shrink-0 text-xs text-muted">作者</dt>
        <dd class="text-xs text-body">{{ APP_AUTHOR }}</dd>
      </div>
      <div class="flex gap-2">
        <dt class="w-24 shrink-0 text-xs text-muted">版本</dt>
        <dd class="text-xs text-body">{{ APP_VERSION }}</dd>
      </div>
      <div class="flex gap-2">
        <dt class="w-24 shrink-0 text-xs text-muted">网址</dt>
        <dd class="min-w-0 truncate text-xs">
          <a
            :href="APP_REPO_URL"
            target="_blank"
            rel="noopener noreferrer"
            class="font-mono text-accent-text hover:underline"
            >github.com/jayvzh/AlbumShelf</a
          >
        </dd>
      </div>
    </dl>
  </section>
</template>
