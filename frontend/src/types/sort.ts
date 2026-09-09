// 排序类型（对齐 docs/API.md §3.6/§3.7/§4 契约）
export type SortMode =
  | 'filename'
  | 'natural'
  | 'modified_time'
  | 'created_time'
  | 'file_size'
  | 'regex'

export type SortDirection = 'asc' | 'desc'

// 正则排序单条规则（API.md §4；type: number=数值比较，string=大小写不敏感比较）
export interface SortRule {
  group: number
  type: 'number' | 'string'
  direction: SortDirection
}

// POST /api/v1/sort/preview 请求体（API.md §3.7）
export interface SortPreviewRequest {
  files: string[]
  mode: 'regex'
  regex: string
  rules?: SortRule[]
}

// POST /api/v1/sort/preview 响应体：matches/unmatched 保持输入原序，sorted 为完整最终顺序
export interface SortPreviewMatch {
  filename: string
  groups: string[]
}

export interface SortPreviewResponse {
  sorted: string[]
  matches: SortPreviewMatch[]
  unmatched: string[]
}
