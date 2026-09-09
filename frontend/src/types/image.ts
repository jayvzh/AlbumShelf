// 图片元信息（对应 docs/API.md §3.4，resolution 读文件头失败时为 null）
export interface ImageInfo {
  name: string
  path: string
  resolution: { width: number | null; height: number | null }
  size: number
  modified_at: string // ISO 8601
}
