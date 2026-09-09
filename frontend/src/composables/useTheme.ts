// 主题切换：localStorage 持久化（键 imageshelf:theme，index.html 内联脚本按同一约定防首屏闪烁）
import { ref } from 'vue'

type Theme = 'light' | 'dark'

const STORAGE_KEY = 'imageshelf:theme'

// 初始值与 index.html 内联脚本保持一致：存过 light 即浅色，否则深色
const theme = ref<Theme>(
  (() => {
    try {
      return localStorage.getItem(STORAGE_KEY) === 'light' ? 'light' : 'dark'
    } catch {
      return 'dark'
    }
  })(),
)

function apply(t: Theme) {
  document.documentElement.classList.toggle('dark', t === 'dark')
}

apply(theme.value)

export function useTheme() {
  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
    try {
      localStorage.setItem(STORAGE_KEY, theme.value)
    } catch {
      /* 存储不可用时仅本次会话生效 */
    }
    apply(theme.value)
  }
  return { theme, toggleTheme }
}
