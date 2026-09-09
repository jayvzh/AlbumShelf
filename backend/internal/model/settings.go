package model

// 阅读页模式常量（PRD F001；view_mode 列保留暂不接入）。
const (
	PageModeSingle = "single"
	PageModeSpread = "spread"
)

// 阅读方向常量（PRD F001）。
const (
	ReadOrderLeftToRight  = "left_to_right"
	ReadOrderRightToLeft  = "right_to_left"
)

// FolderSettings 每目录设置（DATA_MODEL §2.2）。
// Path/SortMode/SortDirection/RegexPattern/RegexConfig 空字符串表示未保存；
// spread 字段用指针表达 null（未保存）：WideRatio nil、SingleFirstPage/SingleLastPage nil。
type FolderSettings struct {
	Path             string
	SortMode         string
	SortDirection    string
	RegexPattern     string // regex 模式使用的正则表达式
	RegexConfig      string // 排序规则 JSON（{"rules":[...]}）
	PageMode         string  // single/spread；空串=未保存
	ReadOrder        string  // left_to_right/right_to_left；空串=未保存
	WideRatio        *float64 // spread 页宽比阈值；nil=未保存
	SingleFirstPage  *bool    // spread 首页独立单页；nil=未保存
	SingleLastPage   *bool    // spread 末页独立单页；nil=未保存
}
