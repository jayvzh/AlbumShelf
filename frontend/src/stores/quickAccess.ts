import { defineStore } from 'pinia'
import { ApiError } from '../services/api'
import { getQuickAccess, toggleQuickAccess } from '../services/quickAccess.service'
import { useAuthStore } from './auth'

// 展开状态持久化键（localStorage，约定同 folder.ts）：刷新后恢复快捷访问节点展开状态
const EXPANDED_KEY = 'albumshelf:quick-access-expanded'

function storedExpanded(): boolean {
  try {
    return localStorage.getItem(EXPANDED_KEY) === '1'
  } catch {
    return false
  }
}

function persistExpanded(expanded: boolean) {
  try {
    localStorage.setItem(EXPANDED_KEY, expanded ? '1' : '0')
  } catch {
    /* 存储不可用时仅本次会话生效 */
  }
}

// 快捷访问（目录钉住）全局状态：
// list 供快捷访问节点列表渲染（固定时间倒序）；paths 供目录树图钉按钮 O(1) 判定。
// 门控：auth.enabled 且未登录时不发起请求、不渲染入口（available getter）。
export const useQuickAccessStore = defineStore('quickAccess', {
  state: () => ({
    list: [] as string[],
    paths: new Set<string>(),
    expanded: storedExpanded(),
    loaded: false,
    loading: false,
    error: null as string | null,
  }),
  getters: {
    count: (state) => state.paths.size,
    has: (state) => (path: string) => state.paths.has(path),
    // 快捷访问是否可用：auth 关闭（无密码模式）或已登录
    available: () => {
      const auth = useAuthStore()
      return !auth.enabled || auth.authenticated
    },
  },
  actions: {
    // 拉取固定目录列表，填充 list 与 paths
    async fetchAll() {
      this.loading = true
      try {
        const res = await getQuickAccess()
        this.list = res.paths
        this.paths = new Set(res.paths)
        this.loaded = true
        this.error = null
      } catch (e) {
        this.error = e instanceof ApiError ? `${e.code}: ${e.message}` : String(e)
      } finally {
        this.loading = false
      }
    },
    // 切换固定：乐观更新，失败回滚。返回切换后的状态（以后端响应为准）。
    async toggle(path: string): Promise<boolean> {
      const wasPinned = this.paths.has(path)
      this.apply(path, !wasPinned)
      try {
        const res = await toggleQuickAccess(path)
        this.apply(path, res.pinned)
        return res.pinned
      } catch (e) {
        this.apply(path, wasPinned) // 回滚
        throw e
      }
    },
    // 应用固定状态到 paths 与 list
    apply(path: string, pinned: boolean) {
      if (pinned) {
        this.paths.add(path)
        if (!this.list.includes(path)) {
          this.list.unshift(path) // 列表为固定时间倒序，新固定在前
        }
      } else {
        this.paths.delete(path)
        this.list = this.list.filter((p) => p !== path)
      }
    },
    // 展开/收起快捷访问列表并持久化（浏览器本地偏好）
    toggleExpanded() {
      this.expanded = !this.expanded
      persistExpanded(this.expanded)
    },
    // 退出登录 / 门控切换时清空数据（expanded 属本地偏好，不清）
    clear() {
      this.list = []
      this.paths = new Set()
      this.loaded = false
      this.error = null
    },
  },
})
