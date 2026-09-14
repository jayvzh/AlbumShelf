<script setup lang="ts">
// 左上角浮动工具栏：页码与缩放率（常显）+ 可展开控件区（信息 / 原图 / 缩放 / Fit / 100% / 旋转 / 全屏 / 缩略图条显隐）
// 默认收起，展开与否持久化到 localStorage；展开时控件排在状态区右侧；模式切换在顶部 AppHeader（全图模式 Esc 退出）
import { ref } from 'vue'
import AppButton from '../common/AppButton.vue'
import type { ViewMode } from '../../stores/viewer'

withDefaults(
  defineProps<{
    scale: number
    isFullscreen?: boolean
    showInfo?: boolean
    // 页码标签：单图帧 "N / total"，双图帧 "N–M / total"（按帧翻页）
    pageLabel: string
    // 当前加载变体：preview 预览图 / original 原图
    variant: 'preview' | 'original'
    // 当前视图模式（仅用于全图模式下的缩略图条显隐按钮）
    mode: ViewMode
    // 全图模式底部缩略图条是否可见（仅影响显隐按钮提示文案）
    filmstripVisible?: boolean
    // 左右镜像是否开启（运行态）
    mirrored?: boolean
  }>(),
  { isFullscreen: false, showInfo: false, filmstripVisible: false, mirrored: false },
)

const emit = defineEmits<{
  fit: []
  scale100: []
  'zoom-in': []
  'zoom-out': []
  // 顺时针旋转 90°（useViewer 运行态）
  rotate: []
  // 切换左右镜像
  'toggle-mirror': []
  'toggle-fullscreen': []
  'toggle-info': []
  // 切换预览图 / 原图
  'toggle-variant': []
  // 下载当前帧（按当前变体：预览图 / 原图）
  download: []
  // 全图模式：显示/隐藏底部缩略图条
  'toggle-filmstrip': []
  // 点击页码：回到第一张图
  'go-first': []
}>()

const EXPANDED_KEY = 'albumshelf:toolbar-expanded'

function storedExpanded(): boolean {
  try {
    return localStorage.getItem(EXPANDED_KEY) === 'true'
  } catch {
    return false
  }
}

// 展开状态：默认收起，切换时持久化（存储不可用时仅本次会话生效）
const expanded = ref(storedExpanded())

function toggleExpanded() {
  expanded.value = !expanded.value
  try {
    localStorage.setItem(EXPANDED_KEY, String(expanded.value))
  } catch {
    /* 存储不可用时仅本次会话生效 */
  }
}
</script>

<template>
  <div class="absolute left-3 top-3 z-10 flex items-center gap-1 rounded-lg border border-line bg-panel/90 p-1 opacity-80">
    <!-- 状态区：常显（收起时与展开开关一起构成整条工具栏）；页码可点击回到第一张图 -->
    <button
      type="button"
      title="回到第一张"
      class="cursor-pointer px-2 text-xs text-muted transition-colors hover:text-ink"
      @click="emit('go-first')"
    >
      {{ pageLabel }}
    </button>
    <span class="px-2 text-xs text-muted">{{ Math.round(scale * 100) }}%</span>
    <!-- 展开控件区：顺序 显示信息 → 加载原图 → 下载 → 其他（缩放 / Fit / 100% / 旋转 / 全屏 / 缩略图条） -->
    <template v-if="expanded">
      <AppButton variant="icon" :title="showInfo ? '隐藏信息' : '显示信息'" @click="emit('toggle-info')">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8h.01M12 11v5" />
        </svg>
      </AppButton>
      <!-- 变体切换：preview 为"加载原图"常规按钮；original 为"原图"常亮高亮（amber accent），再点切回预览图 -->
      <button
        type="button"
        :title="variant === 'preview' ? '加载原图' : '当前为原图，点击切回预览图'"
        :class="
          variant === 'preview'
            ? 'flex h-8 items-center rounded bg-elevated px-3 text-sm text-body transition-colors hover:bg-elevated-hover'
            : 'flex h-8 items-center rounded border border-amber-400/70 bg-amber-400/15 px-3 text-sm font-medium text-accent-text transition-colors hover:bg-amber-400/25'
        "
        @click="emit('toggle-variant')"
      >
        {{ variant === 'preview' ? '加载原图' : '原图' }}
      </button>
      <!-- 下载：紧跟原图按钮，按当前变体下载（预览图 / 原图，优先命中浏览器缓存）；仿微信下载图标 -->
      <AppButton variant="icon" title="下载" @click="emit('download')">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <path d="m7 10 5 5 5-5" />
          <path d="M12 15V3" />
        </svg>
      </AppButton>
      <AppButton variant="icon" title="缩小" @click="emit('zoom-out')">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
          <path d="M5 12h14" />
        </svg>
      </AppButton>
      <AppButton variant="icon" title="放大" @click="emit('zoom-in')">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
          <path d="M12 5v14M5 12h14" />
        </svg>
      </AppButton>
      <AppButton title="适应窗口 (0)" @click="emit('fit')">Fit</AppButton>
      <AppButton title="原始尺寸 (1)" @click="emit('scale100')">1:1</AppButton>
      <!-- 镜像：开启时图标琥珀高亮（着色放插槽内，规避 AppButton 内建 text-body 的覆盖顺序问题） -->
      <AppButton
        variant="icon"
        :title="mirrored ? '退出左右镜像' : '左右镜像'"
        @click="emit('toggle-mirror')"
      >
        <span class="flex" :class="mirrored ? 'text-accent-text' : ''">
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="m18 7 4 4-4 4" />
            <path d="m6 7-4 4 4 4" />
            <path d="M12 3v18" />
          </svg>
        </span>
      </AppButton>
      <AppButton variant="icon" title="顺时针旋转 90° (R)" @click="emit('rotate')">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 12a9 9 0 1 1-2.64-6.36" />
          <path d="M21 3v6h-6" />
        </svg>
      </AppButton>
      <AppButton
        variant="icon"
        :title="isFullscreen ? '退出全屏 (F)' : '全屏 (F)'"
        @click="emit('toggle-fullscreen')"
      >
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path v-if="!isFullscreen" d="M8 3H5a2 2 0 0 0-2 2v3M16 3h3a2 2 0 0 1 2 2v3M8 21H5a2 2 0 0 1-2-2v-3M16 21h3a2 2 0 0 0 2-2v-3" />
          <path v-else d="M8 3v3a2 2 0 0 1-2 2H3M16 3v3a2 2 0 0 0 2 2h3M8 21v-3a2 2 0 0 0-2-2H3M16 21v-3a2 2 0 0 1 2-2h3" />
        </svg>
      </AppButton>
      <!-- 缩略图条显隐：仅全图模式提供（图片模式底部缩略图条常驻） -->
      <AppButton
        v-if="mode === 'full'"
        variant="icon"
        :title="filmstripVisible ? '隐藏缩略图条' : '显示缩略图条'"
        @click="emit('toggle-filmstrip')"
      >
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="4" width="18" height="16" rx="2" />
          <path d="M3 15h18" />
        </svg>
      </AppButton>
    </template>
    <!-- 展开/收起开关：固定最右端，窄条样式（w-6 + h-4 图标），收起时仅占状态区右侧一小段 -->
    <button
      type="button"
      :title="expanded ? '收起工具栏' : '展开工具栏'"
      class="flex h-8 w-6 shrink-0 items-center justify-center rounded text-body transition-colors hover:bg-elevated hover:text-ink"
      @click="toggleExpanded"
    >
      <svg v-if="expanded" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M15 6l-6 6 6 6" />
      </svg>
      <svg v-else class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M9 6l6 6-6 6" />
      </svg>
    </button>
  </div>
</template>
