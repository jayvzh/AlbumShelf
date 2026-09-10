package model

import "time"

// ConfigVersion 配置导入导出格式版本。
const ConfigVersion = 1

// ProtectedFolder 对应 protected_folders 表一行（SPRINT7 §5.3）。
// Path 为 IMAGE_ROOT 相对路径，统一以 / 前缀规范化（如 /Private）。
type ProtectedFolder struct {
	ID        int64
	Path      string
	CreatedAt time.Time
}

// ConfigFolderEntry 配置导入导出中的单目录设置条目（path + 设置字段平铺）。
// 指针字段为 wire format：nil = 未保存（JSON null）。
type ConfigFolderEntry struct {
	Path            string   `json:"path"`
	SortMode        *string  `json:"sort_mode"`
	SortDirection   *string  `json:"sort_direction"`
	RegexPattern    *string  `json:"regex_pattern"`
	RegexConfig     *string  `json:"regex_config"`
	PageMode        *string  `json:"page_mode"`
	ReadOrder       *string  `json:"read_order"`
	WideRatio       *float64 `json:"wide_ratio"`
	SingleFirstPage *bool    `json:"single_first_page"`
	SingleLastPage  *bool    `json:"single_last_page"`
}

// ToSettings 转为 FolderSettings：nil 指针映射为空串/nil（未保存语义）。
func (e ConfigFolderEntry) ToSettings() FolderSettings {
	return FolderSettings{
		Path:            e.Path,
		SortMode:        derefString(e.SortMode),
		SortDirection:   derefString(e.SortDirection),
		RegexPattern:    derefString(e.RegexPattern),
		RegexConfig:     derefString(e.RegexConfig),
		PageMode:        derefString(e.PageMode),
		ReadOrder:       derefString(e.ReadOrder),
		WideRatio:       e.WideRatio,
		SingleFirstPage: e.SingleFirstPage,
		SingleLastPage:  e.SingleLastPage,
	}
}

// NewConfigFolderEntry 从 FolderSettings 构造：空串映射为 nil 指针（JSON null）。
func NewConfigFolderEntry(s FolderSettings) ConfigFolderEntry {
	return ConfigFolderEntry{
		Path:            s.Path,
		SortMode:        emptyToNil(s.SortMode),
		SortDirection:   emptyToNil(s.SortDirection),
		RegexPattern:    emptyToNil(s.RegexPattern),
		RegexConfig:     emptyToNil(s.RegexConfig),
		PageMode:        emptyToNil(s.PageMode),
		ReadOrder:       emptyToNil(s.ReadOrder),
		WideRatio:       s.WideRatio,
		SingleFirstPage: s.SingleFirstPage,
		SingleLastPage:  s.SingleLastPage,
	}
}

// ConfigFavoriteEntry 配置导入导出中的单条收藏（path + 收藏时间）。
// favorites 为可选字段（omitempty）：旧配置无此字段导入不受影响，
// 新配置被旧版程序导入时忽略未知字段，双向兼容。
type ConfigFavoriteEntry struct {
	Path      string `json:"path"`
	CreatedAt string `json:"created_at,omitempty"`
}

// ConfigPayload 配置导出/导入的顶层结构（SPRINT7 §5.7）。
type ConfigPayload struct {
	Version          int                   `json:"version"`
	ExportedAt       string                `json:"exported_at"`
	FolderSettings   []ConfigFolderEntry   `json:"folder_settings"`
	ProtectedFolders []string              `json:"protected_folders"`
	Favorites        []ConfigFavoriteEntry `json:"favorites,omitempty"`
	QuickAccess      []string              `json:"quick_access,omitempty"`
}

// ConfigImportResult 导入结果：各类导入条数（SPRINT7 §5.7）。
type ConfigImportResult struct {
	FolderSettings   int `json:"folder_settings"`
	ProtectedFolders int `json:"protected_folders"`
	Favorites        int `json:"favorites"`
	QuickAccess      int `json:"quick_access"`
}

// derefString nil 指针映射为空串。
func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// emptyToNil 空串映射为 nil 指针。
func emptyToNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
