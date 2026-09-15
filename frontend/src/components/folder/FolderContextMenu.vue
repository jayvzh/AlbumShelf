<script setup lang="ts">
// 目录行右键/长按菜单：Teleport 到 body，按触发点坐标定位，挂载后按菜单尺寸做视口边缘收敛；
// 点击遮罩（左键/右键）或 Esc 关闭。纯展示组件，菜单项由调用方传入并处理。
import { onMounted, onUnmounted, ref } from 'vue'

export interface FolderContextMenuItem {
  key: string
  label: string
}

const props = defineProps<{
  x: number
  y: number
  items: FolderContextMenuItem[]
}>()
const emit = defineEmits<{
  select: [key: string]
  close: []
}>()

const el = ref<HTMLElement | null>(null)
// 边缘收敛后的实际坐标（初始取鼠标位置，挂载后按菜单实际尺寸修正）
const pos = ref({ x: props.x, y: props.y })

function clamp() {
  if (!el.value) return
  const rect = el.value.getBoundingClientRect()
  pos.value = {
    x: Math.min(props.x, window.innerWidth - rect.width - 8),
    y: Math.min(props.y, window.innerHeight - rect.height - 8),
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

// 移动端长按：菜单弹出的瞬间，同一次触摸的 click / contextmenu 会落在遮罩上；
// 开挡窗口内忽略关闭，防菜单闪现即逝（桌面右键与后续点击间隔远大于窗口，不受影响）
const openedAt = Date.now()

function requestClose() {
  if (Date.now() - openedAt < 400) return
  emit('close')
}

function choose(key: string) {
  emit('select', key)
  emit('close')
}

onMounted(() => {
  clamp()
  window.addEventListener('keydown', onKeydown)
})
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <!-- 透明遮罩：左键/右键点击任意处关闭，右键不弹浏览器原生菜单 -->
    <div
      class="fixed inset-0 z-40"
      @click="requestClose"
      @contextmenu.prevent="requestClose"
    />
    <div
      ref="el"
      class="fixed z-50 w-40 rounded border border-line-strong bg-panel py-1 shadow-lg"
      :style="{ left: `${pos.x}px`, top: `${pos.y}px` }"
    >
      <button
        v-for="item in items"
        :key="item.key"
        type="button"
        class="w-full px-3 py-1.5 text-left text-sm text-body transition-colors hover:bg-elevated hover:text-ink"
        @click="choose(item.key)"
      >
        {{ item.label }}
      </button>
    </div>
  </Teleport>
</template>
