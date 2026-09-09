// 路径提示：悬停任意带 data-path 标记的元素（目录树行 / 文件模式卡片）时，
// 在视口左下角胶囊条显示完整路径；document 级事件委托，元素只需标记 data-path
import { onMounted, onUnmounted, ref } from 'vue'

const hoveredPath = ref<string | null>(null)

function onOver(e: MouseEvent) {
  hoveredPath.value = (e.target as Element).closest('[data-path]')?.getAttribute('data-path') ?? null
}

// 指针离开窗口（relatedTarget 为 null）时清除，避免胶囊条残留
function onOut(e: MouseEvent) {
  if (!e.relatedTarget) hoveredPath.value = null
}

export function usePathHint() {
  onMounted(() => {
    document.addEventListener('mouseover', onOver)
    document.addEventListener('mouseout', onOut)
  })
  onUnmounted(() => {
    document.removeEventListener('mouseover', onOver)
    document.removeEventListener('mouseout', onOut)
    hoveredPath.value = null
  })
  return { hoveredPath }
}
