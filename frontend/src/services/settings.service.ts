import { request } from './api'
import type { FolderSettings } from '../types/settings'

// 读取文件夹设置（未保存过的目录字段为 null）
export function getFolderSettings(path: string): Promise<FolderSettings> {
  // 根目录（'/' 或空）直接传 path=/，其余路径做 URL 编码
  const query = !path || path === '/' ? '/' : encodeURIComponent(path)
  return request<FolderSettings>(`/folder/settings?path=${query}`)
}

// 保存文件夹设置
export function saveFolderSettings(settings: FolderSettings): Promise<FolderSettings> {
  return request<FolderSettings>('/folder/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  })
}
