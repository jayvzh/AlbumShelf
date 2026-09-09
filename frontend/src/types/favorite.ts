import type { ImageFile } from './file'

// GET /favorites 响应（images 与 GET /folders 的 images 同构，按收藏时间倒序）
export interface FavoriteListResponse {
  images: ImageFile[]
}

// POST /favorites/toggle 响应：切换后的收藏状态
export interface FavoriteToggleResponse {
  path: string
  favorited: boolean
}
