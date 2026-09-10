import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useFavoritesStore } from '../stores/favorites'
import { fetchSetupStatus } from '../services/setup.service'
import BrowserPlatformPage from '../pages/BrowserPlatformPage.vue'
import LoginPage from '../pages/LoginPage.vue'
import SetupPage from '../pages/SetupPage.vue'

// 三条路由：/ 浏览（平台分流：桌面 BrowserPage / 移动 MobileBrowserPage）、
// /login 登录（SPRINT7_TASK.md §5.9）、
// /setup 初始化引导（SPRINT8_TASK.md §5.4）；
// 设置为账户菜单内的弹窗（SettingsModal），不再占用路由
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'browser', component: BrowserPlatformPage },
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

  // —— favorites 门控联动（PRD F010）——
  // 可用（auth 关闭的游客模式 或 已登录）时拉取一次收藏；不可用（退出登录后）清空本地状态。
  // 必须在下方「游客模式提前返回」之前执行：游客模式 auth.enabled=false 同样需要加载收藏。
  // 不 await，避免阻塞导航；收藏夹视图打开时以 loaded/loading 防重复请求。
  const favorites = useFavoritesStore()
  if (favorites.available) {
    if (!favorites.loaded && !favorites.loading) favorites.fetchAll()
  } else if (favorites.loaded) {
    favorites.clear()
  }

  // 登录系统关闭：全部放行（单用户场景管理端点开放），仅 /login 回浏览页
  if (!auth.enabled) {
    return to.path === '/login' ? { path: '/' } : true
  }

  return true
})

export default router
