// 键盘快捷键：window keydown → Viewer Action（键盘不直接操作 DOM，docs/UI_DESIGN.md §5）
export interface KeyboardHandlers {
  previous(): void
  next(): void
  toggleFullscreen(): void
  fit(): void
  setScale100(): void
  rotate(): void
  close(): void
  toggleFavorite(): void
}

// e.code → handler 名
const KEY_ACTIONS: Record<string, keyof KeyboardHandlers> = {
  ArrowLeft: 'previous',
  ArrowRight: 'next',
  Space: 'next',
  KeyF: 'toggleFullscreen',
  Digit0: 'fit',
  Digit1: 'setScale100',
  KeyR: 'rotate',
  KeyS: 'toggleFavorite',
  Escape: 'close',
}

// 焦点在表单输入元素时不接管按键：登录框、设置弹窗、各处输入框内正常打字（含 1/F/R/S 等快捷键字符），
// 也兜底 HMR 残留监听导致的登录页按键失效
function isFormTarget(e: KeyboardEvent): boolean {
  const target = e.target as HTMLElement | null
  if (!target) return false
  return (
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.tagName === 'SELECT' ||
    target.isContentEditable
  )
}

export function useKeyboard(handlers: KeyboardHandlers) {
  function onKeyDown(e: KeyboardEvent) {
    if (isFormTarget(e)) return
    // 带功能键的组合键不接管：放行 Ctrl/Cmd/Alt+R/F/S/0/1 等浏览器快捷键，
    // 避免 Ctrl+Shift+R 误旋转、Ctrl+S 误收藏等（Ctrl+Shift+R 已被 ctrlKey 覆盖）
    if (e.ctrlKey || e.metaKey || e.altKey) return
    const action = KEY_ACTIONS[e.code]
    if (!action) return
    e.preventDefault()
    handlers[action]()
  }

  function bind() {
    window.addEventListener('keydown', onKeyDown)
  }

  function unbind() {
    window.removeEventListener('keydown', onKeyDown)
  }

  return { bind, unbind }
}
