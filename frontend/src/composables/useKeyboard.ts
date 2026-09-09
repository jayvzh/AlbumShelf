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

export function useKeyboard(handlers: KeyboardHandlers) {
  function onKeyDown(e: KeyboardEvent) {
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
