import { defineStore } from 'pinia'
import { ApiError } from '../services/api'
import { getFolder, type FolderQueryOptions } from '../services/folder.service'
import { useFavoritesStore } from './favorites'
import type { FolderItem, FolderResponse } from '../types/folder'
import type { ImageFile } from '../types/file'

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
      this.error = null
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
