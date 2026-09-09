import { defineStore } from 'pinia'
import { fetchAuthStatus, login as loginApi, logout as logoutApi } from '../services/auth.service'

// 认证全局状态（第 5 个 Store，SPRINT7_TASK.md §5.10）：
// enabled 为总开关（AUTH_PASSWORD 未设置即关闭），路由守卫与 UI 均以 enabled 为准；
// enabled=false 时 authenticated 恒保持 false
export const useAuthStore = defineStore('auth', {
  state: () => ({
    enabled: false,
    authenticated: false,
    username: null as string | null,
    loaded: false,
  }),
  actions: {
    // 探测登录状态；幂等（路由守卫首次导航前调用一次，后续按需刷新）
    async fetchStatus() {
      try {
        const status = await fetchAuthStatus()
        this.enabled = status.enabled
        this.authenticated = status.authenticated
        this.username = status.username ?? null
      } catch {
        // 探测失败按未登录处理，不阻断页面渲染（业务请求各自提示网络错误）
        this.enabled = false
        this.authenticated = false
        this.username = null
      } finally {
        this.loaded = true
      }
    },
    // 登录：成功后刷新状态（获取 username），失败向上抛出由调用方提示
    async login(username: string, password: string) {
      const result = await loginApi({ username, password })
      await this.fetchStatus()
      return result
    },
    // 退出：服务端删除会话 + 清 Cookie，随后刷新状态（回到游客态）
    async logout() {
      await logoutApi()
      await this.fetchStatus()
    },
  },
})
