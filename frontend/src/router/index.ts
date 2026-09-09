import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useFavoritesStore } from '../stores/favorites'
import { fetchSetupStatus } from '../services/setup.service'
import BrowserPage from '../pages/BrowserPage.vue'
import SettingsPage from '../pages/SettingsPage.vue'
import LoginPage from '../pages/LoginPage.vue'
import SetupPage from '../pages/SetupPage.vue'

// 四条路由：/ 浏览、/settings 设置、/login 登录（SPRINT7_TASK.md §5.9）、
// /setup 初始化引导（SPRINT8_TASK.md §5.4）
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'browser', component: BrowserPage },
    { path: '/settings', name: 'settings', component: SettingsPage },
    { path: '/login', name: 'login', component: LoginPage, meta: { public: true } },
    { path: '/setup', name: 'setup', component: SetupPage, meta: { public: true } },
    // 未匹配兜底回浏览页
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

// setup 检测结果（模块级 memo，进程内仅首次导航请求一次；null = 未检测）。
// 请求失败放行（视为就绪），避免后端临时不可达时全站锁死在引导页。
let setupInitialized: boolean | null = null

// SetupPage「重新检测」通过后更新 memo，使守卫放行进入应用
export function markSetupInitialized() {
  setupInitialized = true
}

// 全局守卫：setup 检测前置（先于 auth 检测，互不改写对方逻辑），未初始化强制引导页
router.beforeEach(async (to) => {
  if (setupInitialized === null) {
    try {
      setupInitialized = (await fetchSetupStatus()).initialized
    } catch {
      setupInitialized = true
    }
  }
  if (!setupInitialized && to.path !== '/setup') return { path: '/setup' }
  if (setupInitialized && to.path === '/setup') return { path: '/' }

  // —— Sprint 7 auth 守卫（原逻辑不变）——
  const auth = useAuthStore()
  if (!auth.loaded) await auth.fetchStatus()

  // 登录系统关闭：全部放行（单用户场景管理端点开放），仅 /login 回浏览页
  if (!auth.enabled) {
    return to.path === '/login' ? { path: '/' } : true
  }
  // 设置页需登录：未登录跳登录页并携带回跳地址
  if (to.path === '/settings' && !auth.authenticated) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  // —— favorites 门控联动（PRD F010）——
  // 可用（auth 关闭或已登录）时拉取一次收藏；不可用（退出登录后）清空本地状态。
  // 不 await，避免阻塞导航；收藏夹视图打开时以 loaded/loading 防重复请求。
  const favorites = useFavoritesStore()
  if (favorites.available) {
    if (!favorites.loaded && !favorites.loading) favorites.fetchAll()
  } else if (favorites.loaded) {
    favorites.clear()
  }
  return true
})

export default router
