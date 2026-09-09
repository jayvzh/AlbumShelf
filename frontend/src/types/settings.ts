import type { SortDirection, SortMode } from './sort'
import type { PageMode, ReadOrder } from './spread'

// 文件夹设置（API.md §3.6/§4；regex_pattern/regex_config 仅 sort_mode=regex 时生效；
// view_mode 为 Phase 2 保留字段，本次不接入前端）
export interface FolderSettings {
  path: string
  sort_mode: SortMode | null
  sort_direction: SortDirection | null
  regex_pattern: string | null
  // regex_config 为 SortRule[] 的 JSON 信封字符串：{"rules":[...]}
  regex_config: string | null
  // 双页阅读选项（后端未保存过时为 null，前端按默认值处理）
  page_mode: PageMode | null
  read_order: ReadOrder | null
  wide_ratio: number | null
  single_first_page: boolean | null
  single_last_page: boolean | null
}
