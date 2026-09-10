import { defineStore } from 'pinia'
import { ApiError } from '../services/api'
import { getFolder, type FolderQueryOptions } from '../services/folder.service'
import { useFavoritesStore } from './favorites'
import type { FolderItem, FolderResponse } from '../types/folder'
import type { ImageFile } from '../types/file'

// 会话持久化键（localStorage，约定同 viewer.ts / useTheme）：刷新后恢复当前目录与展开状态
const CURRENT_PATH_KEY = 'albumshelf:current-path'
const EXPANDED_KEY = 'albumshelf:expanded-folders'
// 收藏夹虚拟视图无真实路径，在 current-path 键中以哨兵值标记
const FAVORITES_SENTINEL = 'favorites'

function storedCurrentPath(): string {
  try {
    return localStorage.getItem(CURRENT_PATH_KEY) || '/'
  } catch {
    return '/'
  }
}

// 展开状态持久化为路径数组；根节点恒展开、不可折叠，不入存储
function storedExpanded(): Record<string, boolean> {
  try {
    const raw: unknown = JSON.parse(localStorage.getItem(EXPANDED_KEY) ?? 'null')
    const map: Record<string, boolean> = {}
    if (Array.isArray(raw)) {
      for (const p of raw) if (typeof p === 'string' && p && p !== '/') map[p] = true
    }
    return map
  } catch {
    return {}
  }
}

function persistCurrentPath(path: string) {
  try {
    localStorage.setItem(CURRENT_PATH_KEY, path)
  } catch {
    /* 存储不可用时仅本次会话生效 */
  }
}

function persistExpanded(expanded: Record<string, boolean>) {
  try {
    localStorage.setItem(EXPANDED_KEY, JSON.stringify(Object.keys(expanded)))
  } catch {
    /* 存储不可用时仅本次会话生效 */
  }
}

// 目录浏览状态：当前目录内容 + 目录树子节点缓存；favoritesView 为侧栏「收藏夹」虚拟视图标记
export const useFolderStore = defineStore('folder', {
  state: () => ({
    currentPath: '/' as string,
    favoritesView: false,
    // 目录内「只看收藏」筛选（不改变 store.images，由消费方过滤展示）
    favoritesOnly: false,
    folders: [] as FolderItem[],
    images: [] as ImageFile[],
    loading: false,
    error: null as string | null,
    // 目录树子节点缓存：path → 子目录列表（空数组 = 已加载且无子目录）
    children: {} as Record<string, FolderItem[]>,
    // 目录树展开状态：path → true（localStorage 持久化，刷新后恢复；根节点除外）
    expandedPaths: storedExpanded(),
  }),
  actions: {
    // 打开目录：更新当前目录内容并缓存其子目录；opts 携带排序参数时透传（API.md §3.2）；同时退出收藏夹虚拟视图
    async openFolder(path: string, opts?: FolderQueryOptions) {
      this.favoritesView = false
      this.loading = true
      try {
        const res: FolderResponse = await getFolder(path, opts)
        this.currentPath = res.path
        this.folders = res.folders
        this.images = res.images
        this.children[res.path] = res.folders
        persistCurrentPath(res.path)
        this.error = null
      } catch (e) {
        this.error = e instanceof ApiError ? `${e.code}: ${e.message}` : String(e)
      } finally {
        this.loading = false
      }
    },
    // 刷新当前目录
    refresh() {
      return this.openFolder(this.currentPath)
    },
    // 打开侧栏「收藏夹」虚拟视图：展示 favorites store 的列表（收藏时间倒序），不发起目录请求；
    // 实时增删由 BrowserPage 按 favoritesView 切换数据源保证
    openFavorites() {
      this.favoritesView = true
      this.currentPath = '收藏夹'
      this.folders = []
      this.images = useFavoritesStore().images
      persistCurrentPath(FAVORITES_SENTINEL)
      this.error = null
    },
    // 展开/折叠目录树节点并持久化；展开时按需加载子目录（根节点恒展开，忽略）
    async toggleExpanded(path: string) {
      if (path === '/') return
      if (this.expandedPaths[path]) {
        delete this.expandedPaths[path]
      } else {
        this.expandedPaths[path] = true
        if (!this.children[path]) await this.loadChildren(path)
      }
      persistExpanded(this.expandedPaths)
    },
    // 刷新后恢复上次会话（FolderTree onMounted 调用）：收藏夹哨兵 → 虚拟视图；
    // 普通路径 → 根节点与沿途祖先预载子目录并展开后打开；路径已失效（被删/无权限）时回退根目录
    async restore() {
      const saved = storedCurrentPath()
      if (saved === FAVORITES_SENTINEL) {
        // 根节点子目录是整棵目录树的基础，任何视图恢复前都需就位
        if (!this.children['/']) await this.loadChildren('/')
        this.openFavorites()
        return
      }
      if (saved && saved !== '/') {
        // 根节点与沿途祖先依次预载子目录（根节点恒展开、不入 expandedPaths），保证目录树逐层可见
        const segments = saved.split('/').filter(Boolean)
        const ancestors = ['/']
        let acc = ''
        for (const seg of segments.slice(0, -1)) {
          acc += `/${seg}`
          ancestors.push(acc)
        }
        for (const ancestor of ancestors) {
          if (ancestor !== '/') this.expandedPaths[ancestor] = true
          if (!this.children[ancestor]) await this.loadChildren(ancestor)
        }
        persistExpanded(this.expandedPaths)
        await this.openFolder(saved)
        if (!this.error) return
      }
      persistCurrentPath('/')
      await this.openFolder('/')
    },
    // 仅为目录树加载并缓存子目录，不影响当前目录内容
    async loadChildren(path: string) {
      if (this.children[path]) return
      try {
        const res = await getFolder(path)
        this.children[path] = res.folders
      } catch {
        // 加载失败缓存为空数组，避免箭头反复触发失败请求
        this.children[path] = []
      }
    },
  },
})
