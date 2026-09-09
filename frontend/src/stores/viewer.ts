import { defineStore } from 'pinia'
import type { ImageFile } from '../types/file'
import type { SpreadFrame } from '../types/spread'
import { useSpreadStore } from './spread'

// 视图模式：file 文件网格 / image 嵌入大图（底部保留 Filmstrip）/ full 全屏覆盖（Filmstrip 可显隐）
export type ViewMode = 'file' | 'image' | 'full'

const MODE_KEY = 'imageshelf:viewmode'
const FILMSTRIP_KEY = 'imageshelf:filmstrip'

function storedMode(): ViewMode {
  try {
    const raw = localStorage.getItem(MODE_KEY)
    return raw === 'image' || raw === 'full' ? raw : 'file'
  } catch {
    return 'file'
  }
}

function storedFilmstrip(): boolean {
  try {
    return localStorage.getItem(FILMSTRIP_KEY) !== 'false'
  } catch {
    return true
  }
}

function persistMode(mode: ViewMode) {
  try {
    localStorage.setItem(MODE_KEY, mode)
  } catch {
    /* 存储不可用时仅本次会话生效 */
  }
}

function persistFilmstrip(visible: boolean) {
  try {
    localStorage.setItem(FILMSTRIP_KEY, String(visible))
  } catch {
    /* 存储不可用时仅本次会话生效 */
  }
}

// 大图浏览状态：帧化翻页——一帧 = 1~2 张图（docs/PROJECT_STRUCTURE.md §13 / SPREAD_ENGINE.md §3.5）
// 帧列表在 spread store；本 store 只持翻页位置与视图模式。scale/pan/rotation/fullscreen 属 UI 运行态，由 useViewer 持有
export const useViewerStore = defineStore('viewer', {
  state: () => ({
    // 视图模式（localStorage 持久化，isOpen 为其派生）
    mode: storedMode(),
    // 全图模式底部缩略图条显隐（localStorage 持久化）
    filmstripVisible: storedFilmstrip(),
    images: [] as ImageFile[],
    // 翻页单位：帧索引（spread store frames 的下标）
    currentFrameIndex: 0,
    // 当前加载变体：每次换帧默认回落预览图（原图可能十几 MB，见 docs/UI_DESIGN.md §6）
    variant: 'preview' as 'preview' | 'original',
  }),
  getters: {
    // 查看器是否打开 = 非文件模式（单一事实源，消费方按 isOpen 判断无需感知模式）
    isOpen(): boolean {
      return this.mode !== 'file'
    },
    frameCount(): number {
      return useSpreadStore().frames.length
    },
    // 当前帧（含 1~2 张图）
    currentFrame(): SpreadFrame | null {
      return useSpreadStore().frames[this.currentFrameIndex] ?? null
    },
    // 当前帧首图（单图语义场景：标题/预载）
    currentImage(): ImageFile | null {
      return this.currentFrame?.images[0] ?? null
    },
    // 当前帧首图的图片索引（imageToFrame 反查，兼容旧消费方）
    currentIndex(): number {
      const first = useSpreadStore().imageToFrame.indexOf(this.currentFrameIndex)
      return first === -1 ? 0 : first
    },
    // 当前帧包含的所有图片索引（Filmstrip 帧级高亮）
    activeImageIndexes(): number[] {
      const spread = useSpreadStore()
      const indexes: number[] = []
      spread.imageToFrame.forEach((frameIndex, imageIndex) => {
        if (frameIndex === this.currentFrameIndex) indexes.push(imageIndex)
      })
      return indexes
    },
  },
  actions: {
    // 打开查看器：重建帧（保证帧与列表同步）并定位到图片所在帧，变体回落预览图；
    // 经卡片/缩略图点击（文件模式）进入时落地为图片模式；已处于图片/全图模式时仅重建并定位，不改变当前模式
    open(images: ImageFile[], index: number) {
      const spread = useSpreadStore()
      spread.rebuild(images)
      this.images = images
      this.currentFrameIndex = spread.imageToFrame[index] ?? 0
      this.variant = 'preview'
      if (!this.isOpen) this.setMode('image')
    },
    // 切换视图模式并持久化；由文件模式进入 image/full 时若未打开，由 BrowserPage watcher 负责 open
    setMode(mode: ViewMode) {
      this.mode = mode
      persistMode(mode)
    },
    // 逐级返回：全图 → 图片 → 文件（Esc / ✕）
    stepBack() {
      this.setMode(this.mode === 'full' ? 'image' : 'file')
    },
    // 全图模式底部缩略图条显隐并持久化
    toggleFilmstrip() {
      this.filmstripVisible = !this.filmstripVisible
      persistFilmstrip(this.filmstripVisible)
    },
    // 下一帧：到末尾即停，不循环；换帧时变体回落预览图
    nextFrame() {
      if (this.currentFrameIndex < this.frameCount - 1) {
        this.currentFrameIndex++
        this.variant = 'preview'
      }
    },
    // 上一帧：到开头即停
    previousFrame() {
      if (this.currentFrameIndex > 0) {
        this.currentFrameIndex--
        this.variant = 'preview'
      }
    },
    // 按键翻页语义入口（按帧推进）
    next() {
      this.nextFrame()
    },
    previous() {
      this.previousFrame()
    },
    // 跳转到指定图片所在的帧
    select(imageIndex: number) {
      const frameIndex = useSpreadStore().imageToFrame[imageIndex]
      if (frameIndex != null && frameIndex !== this.currentFrameIndex) {
        this.currentFrameIndex = frameIndex
        this.variant = 'preview'
      }
    },
    // 帧重建后按锚点图片重同步（模式切换/选项变更保持当前图片不跳变，SPREAD_ENGINE.md §3.5）
    resyncFrame(anchorImageIndex: number) {
      this.select(anchorImageIndex)
    },
    // 切换当前图加载变体（preview ↔ original）
    setVariant(v: 'preview' | 'original') {
      this.variant = v
    },
  },
})
