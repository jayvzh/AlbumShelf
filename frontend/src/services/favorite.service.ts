import { request } from './api'
import type { FavoriteListResponse, FavoriteToggleResponse } from '../types/favorite'

// 获取收藏列表（收藏时间倒序；失效条目已由后端惰性清理）
export function getFavorites(): Promise<FavoriteListResponse> {
  return request<FavoriteListResponse>('/favorites')
}

// 切换收藏状态，返回切换后的状态
export function toggleFavorite(path: string): Promise<FavoriteToggleResponse> {
  return request<FavoriteToggleResponse>('/favorites/toggle', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path }),
  })
}
