import { defineStore } from 'pinia'
import { getProtectedFolders, replaceProtectedFolders } from '../services/appsettings.service'

// 私有目录全局状态：目录树悬浮锁与设置页共用同一份数据，
// 任意一端修改即时反映到另一端。代码标识符保留 protected（API 路径/表名不变）。
export const useProtectedStore = defineStore('protected', {
  state: () => ({
    paths: new Set<string>(),
    loaded: false,
    // 正在切换中的路径集合，用于禁用对应按钮避免竞态（全量替换 API 不支持单条操作）
    toggling: new Set<string>(),
  }),
  getters: {
    isProtected: (state) => (path: string) => state.paths.has(path),
    pathList: (state) => Array.from(state.paths),
  },
  actions: {
    // 加载私有目录列表（幂等；登录态变化或进入设置页时调用）
    async load() {
      try {
        const list = await getProtectedFolders()
        this.paths = new Set(list)
      } catch {
        // 加载失败保持空集，不阻断页面；设置页会自行重试
        this.paths = new Set()
      } finally {
        this.loaded = true
      }
    },
    // 切换单个目录的私有状态：乐观更新本地 Set，全量替换失败时回滚
    // 返回 true 表示成功，false 表示失败（调用方据此提示）
    async toggle(path: string): Promise<boolean> {
      if (path === '/' || this.toggling.has(path)) return false
      this.toggling.add(path)

      const wasProtected = this.paths.has(path)
      if (wasProtected) this.paths.delete(path)
      else this.paths.add(path)

      try {
        const list = await replaceProtectedFolders(Array.from(this.paths))
        this.paths = new Set(list)
        return true
      } catch {
        // 失败回滚到操作前状态
        if (wasProtected) this.paths.add(path)
        else this.paths.delete(path)
        return false
      } finally {
        this.toggling.delete(path)
      }
    },
    // 设置页保存：用新列表全量替换（供 ProtectedFoldersSection 使用）
    async replace(paths: string[]): Promise<void> {
      const list = await replaceProtectedFolders(paths)
      this.paths = new Set(list)
    },
  },
})
