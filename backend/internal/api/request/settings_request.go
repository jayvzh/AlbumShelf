package request

// FolderSettingsRequest 文件夹设置请求体（API.md §3.6）。
// RegexPattern/RegexConfig 仅 sort_mode=regex 时生效。
// spread 字段为指针表达可选语义：null/缺省=未提供，由 service 层规范化为默认值。
type FolderSettingsRequest struct {
	Path             string   `json:"path"`
	SortMode         string   `json:"sort_mode"`
	SortDirection    string   `json:"sort_direction"`
	RegexPattern     string   `json:"regex_pattern"`
	RegexConfig      string   `json:"regex_config"`
	PageMode         *string  `json:"page_mode"`
	ReadOrder        *string  `json:"read_order"`
	WideRatio        *float64 `json:"wide_ratio"`
	SingleFirstPage  *bool    `json:"single_first_page"`
	SingleLastPage   *bool    `json:"single_last_page"`
}
