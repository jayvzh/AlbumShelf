package model

import (
	"encoding/json"
	"fmt"
)

// 排序模式常量（SORT_ENGINE §3）
const (
	SortModeFilename     = "filename"
	SortModeNatural      = "natural"
	SortModeModifiedTime = "modified_time"
	SortModeCreatedTime  = "created_time"
	SortModeFileSize     = "file_size"
	SortModeRegex        = "regex"
)

// 排序方向常量
const (
	SortDirectionAsc  = "asc"
	SortDirectionDesc = "desc"
)

// 排序规则类型常量（SORT_ENGINE §2）
const (
	SortRuleTypeNumber = "number"
	SortRuleTypeString = "string"
)

// SortOptions 排序参数（API.md §3.2 query 参数的领域表示；零值 Mode="" 表示未提供）。
// Regex 仅在 Mode=regex 时生效；Rules 为多规则链式比较配置（空时回退单规则简写）。
type SortOptions struct {
	Mode      string
	Direction string
	Regex     string
	Rules     []SortRule
}

// SortRule 单条排序规则（SORT_ENGINE §2）：按捕获组序号 group 提取内容，
// 以 type 方式比较，direction 决定升降序。
type SortRule struct {
	Group     int    `json:"group"`
	Type      string `json:"type"`
	Direction string `json:"direction"`
}

// sortRulesEnvelope ParseSortRules 的 JSON 信封结构：{"rules":[...]}
type sortRulesEnvelope struct {
	Rules []SortRule `json:"rules"`
}

// ParseSortRules 解析 regex_rules 参数（API.md §3.2）。
// 空串视为未提供（返回 nil）；格式非法或含空 rules 数组时返回错误。
// 注意：仅做结构与取值校验，规则合法性（group 范围等）由 sorting 层裁剪。
func ParseSortRules(raw string) ([]SortRule, error) {
	if raw == "" {
		return nil, nil
	}
	var env sortRulesEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return nil, fmt.Errorf("regex_rules 格式非法: %w", err)
	}
	if env.Rules == nil {
		return nil, fmt.Errorf("regex_rules 缺少 rules 数组")
	}
	return env.Rules, nil
}
