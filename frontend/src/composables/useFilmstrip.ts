import { computed, ref, watchEffect, type Ref } from 'vue'
import type { ImageFile } from '../types/file'
import {
  centerScrollLeft,
  computeItemWidth,
  computeOffsets,
  computeTotalWidth,
  findVisibleRange,
} from '../utils/filmstrip'

// Filmstrip 响应式接线：布局数学全部在 utils/filmstrip.ts（纯函数），本文件只做状态粘合
export function useFilmstrip(images: Ref<ImageFile[]>) {
  // 滚动容器元素（模板绑定 ref）
  const containerRef = ref<HTMLElement | null>(null)
  // 容器实时宽度（ResizeObserver 测量，窗口缩放/侧栏折叠自动重算可见数量）
  const viewportWidth = ref(0)
  // 当前横向滚动位置
  const scrollLeft = ref(0)

  // 等高不等宽布局：宽度按宽高比缩放 + clamp，偏移用前缀和数组
  const layout = computed(() => {
    const widths = images.value.map((img) => computeItemWidth(img.width, img.height))
    return { widths, offsets: computeOffsets(widths), totalWidth: computeTotalWidth(widths) }
  })

  // 可见区间：前缀和偏移 + 二分查找（± overscan），供虚拟渲染 v-for 切片
  const visibleRange = computed(() =>
    findVisibleRange(
      layout.value.offsets,
      layout.value.widths,
      scrollLeft.value,
      viewportWidth.value,
    ),
  )

  // 横向滚动处理器（模板绑定 @scroll）
  function onScroll(e: Event) {
    scrollLeft.value = (e.currentTarget as HTMLElement).scrollLeft
  }

  // 自动居中滚动到第 index 项；容器未挂载（或索引越界）时安全跳过
  function scrollToCenter(index: number) {
    const el = containerRef.value
    if (!el || index < 0 || index >= images.value.length) return
    const left = centerScrollLeft(
      layout.value.offsets[index],
      layout.value.widths[index],
      viewportWidth.value,
      layout.value.totalWidth,
    )
    el.scrollTo({ left, behavior: 'smooth' })
  }

  // ResizeObserver 跟踪容器宽度：containerRef 挂载/卸载时自动重建观察并测量初始宽度，
  // 断开由 onCleanup 处理（组件卸载时随作用域停止，等价 onScopeDispose/onUnmounted 清理）
  watchEffect((onCleanup) => {
    const el = containerRef.value
    if (!el) return
    viewportWidth.value = el.clientWidth
    const observer = new ResizeObserver((entries) => {
      viewportWidth.value = entries[0]?.contentRect.width ?? el.clientWidth
    })
    observer.observe(el)
    onCleanup(() => observer.disconnect())
  })

  return { containerRef, viewportWidth, scrollLeft, layout, visibleRange, onScroll, scrollToCenter }
}

export type UseFilmstripReturn = ReturnType<typeof useFilmstrip>
