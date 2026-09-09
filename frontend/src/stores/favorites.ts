import { defineStore } from 'pinia'
import { ApiError } from '../services/api'
import { getFavorites, toggleFavorite } from '../services/favorite.service'
import type { ImageFile } from '../types/file'
import { useAuthStore } from './auth'

// 收藏全局状态（第 6 个 Store，PRD F010 Phase 2）：
// paths 供卡片角标与「只看收藏」筛选用 O(1) 判定；images 供收藏夹视图渲染（收藏时间倒序）。
// 门控：auth.enabled 且未登录时不发起请求、不渲染入口（available getter）。
export const useFavoritesStore = defineStore('favorites', {
  state: () => ({
    images: [] as ImageFile[],
    paths: new Set<string>(),
    loaded: false,
    loading: false,
    error: null as string | null,
  }),
  getters: {
    count: (state) => state.paths.size,
    has: (state) => (path: string) => state.paths.has(path),
    // 收藏功能是否可用：auth 关闭（无密码模式）或已登录
    available: () => {
      const auth = useAuthStore()
      return !auth.enabled || auth.authenticated
    },
  },
  actions: {
    // 拉取收藏列表，填充 images 与 paths
    async fetchAll() {
      this.loading = true
      try {
        const res = await getFavorites()
        this.images = res.images
        this.paths = new Set(res.images.map((i) => i.path))
        this.loaded = true
        this.error = null
      } catch (e) {
        this.error = e instanceof ApiError ? `${e.code}: ${e.message}` : String(e)
      } finally {
        this.loading = false
      }
    },
    // 切换收藏：乐观更新，失败回滚；image 为新收藏时同步进收藏夹视图列表。
    // 返回切换后的状态（以后端响应为准），失败抛出由调用方提示。
    async toggle(path: string, image?: ImageFile): Promise<boolean> {
      const wasFavorited = this.paths.has(path)
      this.apply(path, !wasFavorited, image)
      try {
        const res = await toggleFavorite(path)
        this.apply(path, res.favorited, image)
        return res.favorited
      } catch (e) {
        this.apply(path, wasFavorited, undefined) // 回滚
        throw e
      }
    },
    // 应用收藏状态到 paths 与 images
    apply(path: string, favorited: boolean, image?: ImageFile) {
      if (favorited) {
        this.paths.add(path)
        if (image && !this.images.some((i) => i.path === path)) {
          this.images.unshift(image) // 列表为收藏时间倒序，新收藏在前
        }
      } else {
        this.paths.delete(path)
        this.images = this.images.filter((i) => i.path !== path)
      }
    },
    // 退出登录 / 门控切换时清空本地状态
    clear() {
      this.images = []
      this.paths = new Set()
      this.loaded = false
      this.error = null
    },
  },
})
