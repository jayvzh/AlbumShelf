// 应用级设置相关类型（私有目录 / 缓存管理 / 配置导入导出，SPRINT7_TASK.md §5.5-§5.7）

// 单变体缓存统计（以磁盘为准）
export interface VariantCacheStats {
  count: number
  bytes: number
}

// 缓存统计（GET /cache/stats）
export interface CacheStats {
  thumb: VariantCacheStats
  preview: VariantCacheStats
}

// 清理结果（POST /cache/cleanup/orphan | /cache/cleanup/all）
export interface CleanupResult {
  removed_files: number
  removed_bytes: number
}

// 配置导出文件中的单目录设置条目（字段与后端 wire format 同形，null = 未保存）
export interface ConfigFolderEntry {
  path: string
  sort_mode: string | null
  sort_direction: string | null
  regex_pattern: string | null
  regex_config: string | null
  page_mode: string | null
  read_order: string | null
  wide_ratio: number | null
  single_first_page: boolean | null
  single_last_page: boolean | null
}

// 配置导入导出 payload（GET /config/export 下载文件与 POST /config/import 请求体同形）
export interface ConfigPayload {
  version: number
  exported_at: string
  folder_settings: ConfigFolderEntry[]
  protected_folders: string[]
}

// 配置导入结果（POST /config/import 响应：各类导入条数）
export interface ConfigImportResult {
  folder_settings: number
  protected_folders: number
}
