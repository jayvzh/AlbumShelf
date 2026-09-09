package request

// ProtectedFoldersRequest 私有目录请求体（PUT /protected-folders，全量替换语义）。
type ProtectedFoldersRequest struct {
	Paths []string `json:"paths"`
}

// ConfigFavoriteEntryRequest 配置导入中的单条收藏；created_at 仅展示用途，导入不保留。
type ConfigFavoriteEntryRequest struct {
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

// ConfigImportRequest 配置导入请求体（SPRINT7 §5.7）。
// folder_settings 元素与 FolderSettingsRequest 同形（含 path）。
type ConfigImportRequest struct {
	Version          int                          `json:"version"`
	ExportedAt       string                       `json:"exported_at"`
	FolderSettings   []FolderSettingsRequest      `json:"folder_settings"`
	ProtectedFolders []string                     `json:"protected_folders"`
	Favorites        []ConfigFavoriteEntryRequest `json:"favorites"`
}
