import { request } from './api'
import type { FolderResponse } from '../types/folder'
import type {
  SortDirection,
  SortMode,
  SortPreviewRequest,
  SortPreviewResponse,
  SortRule,
} from '../types/sort'

// GET /folders 查询参数（API.md §3.2；regex 模式附带 regex 与 regex_rules）
export interface FolderQueryOptions {
  sort?: SortMode
  direction?: SortDirection
  regex?: string
  // 多规则配置，请求时序列化为 {"rules":[...]} 信封放进 query.regex_rules
  regexRules?: SortRule[]
}

// 列出目录内容（子目录 + 图片）；提供的参数才拼进 query（API.md §3.2）
export function getFolder(path: string, opts?: FolderQueryOptions): Promise<FolderResponse> {
  // 根目录（'/' 或空）直接传 path=/，其余路径做 URL 编码
  const query = !path || path === '/' ? '/' : encodeURIComponent(path)
  let url = `/folders?path=${query}`
  if (opts?.sort) url += `&sort=${encodeURIComponent(opts.sort)}`
  if (opts?.direction) url += `&direction=${encodeURIComponent(opts.direction)}`
  if (opts?.regex) url += `&regex=${encodeURIComponent(opts.regex)}`
  if (opts?.regexRules?.length) {
    url += `&regex_rules=${encodeURIComponent(JSON.stringify({ rules: opts.regexRules }))}`
  }
  return request<FolderResponse>(url)
}

// 正则排序预览（API.md §3.7）：非法正则返回 ApiError(code=INVALID_REGEX)
export function previewSort(req: SortPreviewRequest): Promise<SortPreviewResponse> {
  return request<SortPreviewResponse>('/sort/preview', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}
