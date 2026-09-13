<script setup lang="ts">
// 帮助弹窗：项目信息（简略） / 视图模式 / 快捷键 / 排序用法速查（AppHeader 问号按钮入口）
// 仅点击遮罩/✕ 关闭，不监听 Esc：避免与大图浏览的 Esc 逐级返回（useKeyboard）冲突
import { APP_ICON, APP_NAME, APP_REPO_URL, APP_VERSION } from '../../constants/app'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

// 快捷键表（与 composables/useKeyboard.ts 键位一致）
const shortcuts: { keys: string[]; desc: string }[] = [
  { keys: ['←', '→'], desc: '上一帧 / 下一帧（空格同下一帧）' },
  { keys: ['F'], desc: '进入 / 退出浏览器全屏（图片模式下自动切换全图，退出后回到图片模式）' },
  { keys: ['0'], desc: '适应窗口大小（Fit）' },
  { keys: ['1'], desc: '按原始尺寸显示（100%）' },
  { keys: ['R'], desc: '顺时针旋转 90°' },
  { keys: ['S'], desc: '收藏 / 取消收藏当前图片（双页时收藏左图）' },
  { keys: ['Esc'], desc: '逐级返回：全图 → 图片 → 文件' },
]

// 键位小方块统一样式（Tailwind 主题工具类）
const kbdClass =
  'inline-flex min-w-6 items-center justify-center rounded border border-line-strong bg-elevated px-1.5 py-0.5 font-mono text-xs text-ink'
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
      <!-- 关闭按钮：绝对定位置于弹窗右上角（悬浮于品牌带之上，不随内容滚动） -->
      <button
        type="button"
        aria-label="关闭"
        class="absolute right-3 top-3 z-10 rounded px-2 py-1 text-faint transition-colors hover:bg-elevated hover:text-body"
        @click="emit('close')"
      >
        ✕
      </button>

      <!-- 项目信息（简略）：弹窗最顶部居中品牌带——图标 / 名称 / 版本 / 仓库链接 -->
      <div class="flex flex-col items-center gap-1.5 px-4 pt-5 pb-4">
        <img :src="APP_ICON" :alt="APP_NAME" class="h-10 w-10 shrink-0" />
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium text-ink">{{ APP_NAME }}</span>
          <span class="text-xs text-muted">{{ APP_VERSION }}</span>
        </div>
        <a
          :href="APP_REPO_URL"
          target="_blank"
          rel="noopener noreferrer"
          class="text-xs text-accent-text hover:underline"
          >GitHub 仓库</a
        >
      </div>

      <div class="flex-1 space-y-5 overflow-y-auto px-4 py-4">
        <h2 class="border-b border-line pb-2 text-sm font-medium text-ink">使用说明</h2>
        <!-- 视图模式 -->
        <section>
          <h3 class="mb-1.5 text-xs font-medium text-muted">视图模式</h3>
          <ul class="list-disc space-y-1 pl-5 text-sm leading-relaxed text-body">
            <li><span class="text-ink">文件</span>：卡片网格，点击卡片或底部缩略图进入大图浏览</li>
            <li><span class="text-ink">图片</span>：原地嵌入大图，底部保留缩略图条</li>
            <li><span class="text-ink">全图</span>：全屏覆盖浏览，入口为图片模式右上角最大化按钮（进入后原位变为还原按钮）；底部缩略图条可在工具栏显隐</li>
            <li>
              图片 / 全图模式下按 <kbd :class="kbdClass">Esc</kbd> 逐级返回上一级；
              全屏时按 <kbd :class="kbdClass">Esc</kbd> 直接退出全屏
            </li>
            <li>
              <span class="text-ink">收藏</span>：查看器 ✕ 号下方心形按钮或按
              <kbd :class="kbdClass">S</kbd> 收藏当前图片；侧栏「收藏夹」汇总全部收藏，顶栏「只看收藏」筛选当前目录
            </li>
          </ul>
        </section>

        <!-- 快捷键（大图浏览时） -->
        <section>
          <h3 class="mb-1.5 text-xs font-medium text-muted">快捷键（大图浏览时）</h3>
          <div class="rounded border border-line">
            <div
              v-for="(s, i) in shortcuts"
              :key="i"
              class="flex items-center justify-between gap-4 px-3 py-1.5"
              :class="i % 2 === 0 ? 'bg-panel' : 'bg-elevated/40'"
            >
              <span class="flex shrink-0 gap-1">
                <kbd v-for="k in s.keys" :key="k" :class="kbdClass">{{ k }}</kbd>
              </span>
              <span class="text-right text-xs text-body">{{ s.desc }}</span>
            </div>
          </div>
          <p class="mt-1.5 text-xs text-faint">鼠标：滚轮缩放（以指针为锚点），按住拖拽平移画面</p>
        </section>

        <!-- 排序 -->
        <section>
          <h3 class="mb-1.5 text-xs font-medium text-muted">排序</h3>
          <ul class="list-disc space-y-1 pl-5 text-sm leading-relaxed text-body">
            <li>
              顶栏排序菜单选择模式与方向后即时生效；点「保存到当前目录」可记住该目录的排序偏好
            </li>
            <li>
              <span class="text-ink">自定义正则</span>：用正则捕获组提取文件名中的编号（如
              <code class="rounded bg-elevated px-1 font-mono text-xs">chapter(\d+)_page(\d+)</code>
              ），可配置多条规则按顺序逐条比较，并实时预览排序结果
            </li>
          </ul>
        </section>
      </div>
    </div>
  </div>
</template>
