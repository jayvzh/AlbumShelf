<script setup lang="ts">
// 根目录空状态品牌插画（桌面/移动两树共用）：favicon 同款"相框 + 山形 + 搁板"构图放大重绘，
// 三张相框错落呼应"图集馆"。配色沿用 favicon 约定——zinc 深底 / amber 强调固定不随主题，
// 后景相框描边随主题（text-line-strong）。platform 仅区分提示文案（移动端无键盘快捷键）与插画尺寸。
import { computed } from 'vue'
import { APP_NAME } from '../../constants/app'

const props = defineProps<{ platform: 'desktop' | 'mobile' }>()

// 上手提示：移动端不含键盘快捷键
const tips = computed(() => {
  const base = ['把图片或漫画目录放入根目录即可自动扫描', '点击封面进入阅读，支持双指缩放']
  return props.platform === 'desktop'
    ? [...base, '点击心形收藏喜欢的图片（快捷键 S）']
    : [...base, '点击心形收藏喜欢的图片']
})
</script>

<template>
  <!-- 垂直不居中：顶部/底部弹性 spacer 按 1:2 比例，内容重心落在容器约 1/3 高度处（视觉更舒适） -->
  <div class="flex h-full flex-col items-center p-8 text-center select-none">
    <div class="shrink-0 grow-[1]" />
    <div class="flex flex-col items-center gap-5">
      <!-- 插画：三张相框错落摆放在搁板上 -->
    <svg
      class="w-auto"
      :class="platform === 'mobile' ? 'h-20' : 'h-28'"
      viewBox="0 0 128 96"
      fill="none"
      aria-hidden="true"
    >
      <!-- 左后相框（随主题淡描边） -->
      <g class="text-line-strong" transform="rotate(-6 26 44)" stroke="currentColor" stroke-width="3">
        <rect x="12" y="34" width="28" height="20" rx="3" />
      </g>
      <!-- 右后相框 -->
      <g class="text-line-strong" transform="rotate(6 102 44)" stroke="currentColor" stroke-width="3">
        <rect x="88" y="34" width="28" height="20" rx="3" />
      </g>
      <!-- 主相框（深底卡片 + 白描边 + 琥珀太阳/山形，favicon 同款） -->
      <g>
        <rect x="34" y="12" width="60" height="46" rx="6" fill="#18181b" />
        <rect x="40" y="18" width="48" height="32" rx="3" class="text-zinc-100" stroke="currentColor" stroke-width="3" />
        <circle cx="52" cy="27" r="4" fill="#fcd34d" />
        <path
          d="M44 46l8-9 6 7 4.5-4.5L74 48"
          stroke="#fcd34d"
          stroke-width="3.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </g>
      <!-- 搁板 -->
      <path d="M18 78h92" stroke="#fcd34d" stroke-width="5" stroke-linecap="round" />
    </svg>

    <!-- 品牌与定位 -->
    <div class="space-y-1">
      <p class="text-lg font-semibold text-ink">{{ APP_NAME }}</p>
      <p class="text-sm text-muted">NAS 私有图集与漫画库</p>
    </div>

    <!-- 上手提示 -->
    <ul class="space-y-1.5">
      <li v-for="tip in tips" :key="tip" class="text-xs text-faint">{{ tip }}</li>
    </ul>
    </div>
    <div class="shrink-0 grow-[2]" />
  </div>
</template>
