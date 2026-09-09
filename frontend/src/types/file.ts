// 图片文件元信息（对应 docs/API.md §3.2 实际 JSON，字段为 snake_case）
export interface ImageFile {
  name: string
  path: string
  extension: string
  size: number
  width: number | null // 读图片文件头获得，解析失败为 null
  height: number | null
  modified_at: string // ISO 8601
}
