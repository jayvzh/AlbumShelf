package service

import (
	"errors"
	"fmt"
	"time"

	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/model"
	"imageshelf/backend/internal/repository"
)

// ErrConfigInvalid 配置文件非法（version 不符等），handler 映射 400 CONFIG_INVALID。
var ErrConfigInvalid = errors.New("config invalid")

// ConfigService 配置导入导出业务（SPRINT7 §5.7）。
// 主题等 localStorage 偏好不参与。
type ConfigService struct {
	fs        *filesystem.LocalFilesystem
	folders   *repository.FolderRepository
	settings  *repository.SettingsRepository
	protected *AppSettingsService
	favorites *repository.FavoriteRepository
}

// NewConfigService 构造 ConfigService。
func NewConfigService(
	fs *filesystem.LocalFilesystem,
	folders *repository.FolderRepository,
	settings *repository.SettingsRepository,
	protected *AppSettingsService,
	favorites *repository.FavoriteRepository,
) *ConfigService {
	return &ConfigService{fs: fs, folders: folders, settings: settings, protected: protected, favorites: favorites}
}

// Export 导出全量配置：folder_settings + protected_folders + favorites。
func (s *ConfigService) Export() (*model.ConfigPayload, error) {
	settings, err := s.settings.ListAll()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	entries := make([]model.ConfigFolderEntry, 0, len(settings))
	for _, st := range settings {
		entries = append(entries, model.NewConfigFolderEntry(st))
	}
	paths, err := s.protected.ListPaths()
	if err != nil {
		return nil, err
	}
	favs, err := s.favorites.List()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	favEntries := make([]model.ConfigFavoriteEntry, 0, len(favs))
	for _, f := range favs {
		favEntries = append(favEntries, model.ConfigFavoriteEntry{
			Path:      f.Path,
			CreatedAt: f.CreatedAt.Format(time.RFC3339),
		})
	}
	return &model.ConfigPayload{
		Version:          model.ConfigVersion,
		ExportedAt:       time.Now().Format(time.RFC3339),
		FolderSettings:   entries,
		ProtectedFolders: paths,
		Favorites:        favEntries,
	}, nil
}

// Import 导入配置（合并语义）：folder_settings 按目录 path upsert（保留未保存字段的
// null 语义，不走 Save 规范化）；protected_folders 按 path 合并；favorites 按 path
// INSERT OR IGNORE 合并。返回各类导入条数。
func (s *ConfigService) Import(payload model.ConfigPayload) (*model.ConfigImportResult, error) {
	if payload.Version != model.ConfigVersion {
		return nil, fmt.Errorf("%w: version 必须为 %d", ErrConfigInvalid, model.ConfigVersion)
	}

	result := &model.ConfigImportResult{}
	for _, entry := range payload.FolderSettings {
		_, relPath, err := filesystem.Resolve(s.fs.Root(), entry.Path)
		if err != nil {
			return nil, err
		}
		folderID, err := s.folders.GetOrCreateID(relPath)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
		}
		st := entry.ToSettings()
		st.Path = relPath
		if err := s.settings.Upsert(folderID, st); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
		}
		result.FolderSettings++
	}

	if len(payload.ProtectedFolders) > 0 {
		n, err := s.protected.MergePaths(payload.ProtectedFolders)
		if err != nil {
			return nil, err
		}
		result.ProtectedFolders = n
	}

	// 收藏合并导入（INSERT OR IGNORE，不覆盖不删除现有收藏）。
	// 仅做路径安全校验、不要求文件存在（迁移到新 NAS 时文件可能暂缺，
	// 收藏列表的惰性清理会兜底）；created_at 不保留，入库记为导入时间。
	for _, fav := range payload.Favorites {
		if fav.Path == "" {
			continue
		}
		relPath, err := filesystem.ValidatePath(s.fs.Root(), fav.Path)
		if err != nil {
			return nil, err
		}
		if err := s.favorites.Add(relPath); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
		}
		result.Favorites++
	}
	return result, nil
}
