import { computed, onScopeDispose, ref, type Ref } from 'vue'

/**
 * 响应式媒体查询：返回随视口变化实时更新的布尔 Ref。
 * 无 window 环境（SSR/测试）安全，恒为 false。
 */
export function useMediaQuery(query: string): Ref<boolean> {
  const matches = ref(false)
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return matches
  }

  const mql = window.matchMedia(query)
  matches.value = mql.matches
  const onChange = (e: MediaQueryListEvent) => {
    matches.value = e.matches
  }
  mql.addEventListener('change', onChange)
  onScopeDispose(() => mql.removeEventListener('change', onChange))
  return matches
}

/** 移动端断点：与 Tailwind md 断点对齐（<768px 视为移动端） */
export const MOBILE_BREAKPOINT = '(max-width: 767px)'

/** 移动设备 UA 特征：iOS / Android / 各浏览器移动标识（宽度断点之外的兜底判断） */
const MOBILE_UA_RE = /Android|iPhone|iPad|iPod|Mobi/i

/** 是否移动端视口（<768px），路由分流与移动组件树共用 */
export function useIsMobile(): Ref<boolean> {
  const byWidth = useMediaQuery(MOBILE_BREAKPOINT)
  // UA 兜底：手机浏览器"请求电脑版"会把 layout viewport 拉到 980px+，仅靠宽度判断会误渲染桌面树，
  // 导致桌面布局被压缩/底栏推出视口；UA + 宽度 OR 关系，任一命中即视为移动端
  const byUA = ref(
    typeof navigator !== 'undefined' && MOBILE_UA_RE.test(navigator.userAgent),
  )
  return computed(() => byWidth.value || byUA.value)
}
