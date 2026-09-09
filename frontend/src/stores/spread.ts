import { defineStore } from 'pinia'
import type { ImageFile } from '../types/file'
import type { PageMode, ReadOrder, SpreadFrame, SpreadOptions } from '../types/spread'
import { buildSpreadFrames } from '../utils/spread'
import { useFolderStore } from './folder'
import { useSettingsStore } from './settings'
import { useViewerStore } from './viewer'

// 双页阅读状态：SpreadOptions + 拼帧结果（docs/PROJECT_STRUCTURE.md §13）
// 与 viewer store 相互引用：Pinia 支持 store 间循环依赖，跨 store 调用均发生在 action/getter 内
export const useSpreadStore = defineStore('spread', {
  state: () => ({
    pageMode: 'single' as PageMode,
    wideRatio: 1.0 as number,
    readOrder: 'left_to_right' as ReadOrder,
    singleFirstPage: true as boolean,
    singleLastPage: false as boolean,
    frames: [] as SpreadFrame[],
    imageToFrame: [] as number[],
  }),
  getters: {
    // 聚合为引擎入参（utils/spread.buildSpreadFrames）
    spreadOptions(state): SpreadOptions {
      return {
        pageMode: state.pageMode,
        wideRatio: state.wideRatio,
        readOrder: state.readOrder,
        singleFirstPage: state.singleFirstPage,
        singleLastPage: state.singleLastPage,
      }
    },
  },
  actions: {
    // 以当前 options 对 images 重建帧（纯计算，不触发翻页/持久化）
    rebuild(images: ImageFile[]) {
      const layout = buildSpreadFrames(images, this.spreadOptions)
      this.frames = layout.frames
      this.imageToFrame = layout.imageToFrame
    },
    // 用当前目录图片重建帧，并保持锚点图片不跳变
    rebuildForCurrentFolder(anchorImageIndex?: number) {
      const folder = useFolderStore()
      const viewer = useViewerStore()
      this.rebuild(folder.images)
      if (anchorImageIndex != null && viewer.isOpen) {
        viewer.resyncFrame(anchorImageIndex)
      }
    },
    // 更新选项（SpreadControls / AppHeader 调用）：重建帧 → 重同步查看器 → 持久化
    updateOptions(patch: Partial<SpreadOptions>) {
      const viewer = useViewerStore()
      const anchor = viewer.currentIndex // 当前帧首图，作为不跳变锚点
      Object.assign(this, patch)
      this.rebuildForCurrentFolder(anchor)
      void this.persist()
    },
    // 单页/双页切换入口
    togglePageMode() {
      this.updateOptions({ pageMode: this.pageMode === 'single' ? 'spread' : 'single' })
    },
    // 镜像进 settingsStore 并落库；失败静默（阅读不因保存失败中断）
    async persist() {
      const settings = useSettingsStore()
      settings.applySpread(this.spreadOptions)
      try {
        await settings.save()
      } catch {
        // 静默：保存失败不影响阅读
      }
    },
  },
})
