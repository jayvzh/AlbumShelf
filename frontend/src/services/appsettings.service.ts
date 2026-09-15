import { request } from './api'
import type { CacheStats, CleanupResult, ConfigImportResult, WarmLevel, WarmStatus } from '../types/appsettings'

// 私有目录列表（GET /protected-folders）
export function getProtectedFolders(): Promise<string[]> {
  return request<{ paths: string[] }>('/protected-folders').then((r) => r.paths)
}

// 全量替换私有目录（PUT /protected-folders；返回规范化后的列表）
export function replaceProtectedFolders(paths: string[]): Promise<string[]> {
  return request<{ paths: string[] }>('/protected-folders', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ paths }),
  }).then((r) => r.paths)
}

// 缓存统计（GET /cache/stats）
export function getCacheStats(): Promise<CacheStats> {
  return request<CacheStats>('/cache/stats')
}

// 清理孤儿缓存（POST /cache/cleanup/orphan）
export function cleanupOrphanCaches(): Promise<CleanupResult> {
  return request<CleanupResult>('/cache/cleanup/orphan', { method: 'POST' })
}

// 清理全部缓存（POST /cache/cleanup/all）
export function cleanupAllCaches(): Promise<CleanupResult> {
  return request<CleanupResult>('/cache/cleanup/all', { method: 'POST' })
}

// 清空单变体缓存（POST /cache/cleanup/variant；thumb 或 preview 全部清空）
export function cleanupVariantCaches(variant: 'thumb' | 'preview'): Promise<CleanupResult> {
  return request<CleanupResult>('/cache/cleanup/variant', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ variant }),
  })
}

// 预热器状态快照（GET /cache/warm/status）
export function getWarmStatus(): Promise<WarmStatus> {
  return request<WarmStatus>('/cache/warm/status')
}

// 更新预热级别（PUT /cache/warm/settings；持久化并即时生效，后端按新级别重新扫描）
export function setWarmLevel(level: WarmLevel): Promise<WarmStatus> {
  return request<WarmStatus>('/cache/warm/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ level }),
  })
}

// 配置导出下载地址（响应带 attachment 头，浏览器直接下载；不走 fetch 以保留下载行为）
export function exportConfigUrl(): string {
  return '/api/v1/config/export'
}

// 配置导入（POST /config/import；body 为导出文件原文）
export function importConfig(json: string): Promise<ConfigImportResult> {
  return request<ConfigImportResult>('/config/import', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: json,
  })
}
