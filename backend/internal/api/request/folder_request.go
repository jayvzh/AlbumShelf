package request

import (
	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/model"
)

// FolderListRequest 目录列表请求参数。
type FolderListRequest struct {
	Path string
	// Sort/Direction/Regex 缺省为空串，表示请求未提供（service 层回退已保存设置或默认值）。
	Sort      string
	Direction string
	Regex     string
	// RegexRules 多规则 JSON 配置（{"rules":[...]}），仅 sort=regex 时提供。
	RegexRules string
}

// NewFolderListRequest 从 query 参数构造请求对象。
func NewFolderListRequest(c *gin.Context) FolderListRequest {
	return FolderListRequest{
		Path:       c.Query("path"),
		Sort:       c.Query("sort"),
		Direction:  c.Query("direction"),
		Regex:      c.Query("regex"),
		RegexRules: c.Query("regex_rules"),
	}
}

// SortPreviewRequest POST /sort/preview 请求体（API.md §3.7，字段顶层平铺）。
type SortPreviewRequest struct {
	Files []string         `json:"files"`
	Mode  string           `json:"mode"`
	Regex string           `json:"regex"`
	Rules []model.SortRule `json:"rules"`
}
