import { request } from './api'
import type { QuickAccessListResponse, QuickAccessToggleResponse } from '../types/quickAccess'

// 获取快捷访问列表（固定时间倒序；失效条目已由后端惰性清理）
export function getQuickAccess(): Promise<QuickAccessListResponse> {
  return request<QuickAccessListResponse>('/quick-access')
}

// 切换目录固定状态，返回切换后的状态
export function toggleQuickAccess(path: string): Promise<QuickAccessToggleResponse> {
  return request<QuickAccessToggleResponse>('/quick-access/toggle', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path }),
  })
}
