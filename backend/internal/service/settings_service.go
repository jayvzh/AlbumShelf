package service

import (
	"errors"
	"fmt"

	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/model"
	"imageshelf/backend/internal/repository"
)

// ErrDatabase 数据库操作失败 sentinel，handler 用 errors.Is 映射 DATABASE_ERROR。
var ErrDatabase = errors.New("database error")

// SettingsService 文件夹设置业务：路径校验 → 参数规范化 → 持久化（API.md §3.6）。
type SettingsService struct {
	fs       *filesystem.LocalFilesystem
	folders  *repository.FolderRepository
	settings *repository.SettingsRepository
}

// NewSettingsService 构造 SettingsService。
func NewSettingsService(fs *filesystem.LocalFilesystem, folders *repository.FolderRepository, settings *repository.SettingsRepository) *SettingsService {
	return &SettingsService{fs: fs, folders: folders, settings: settings}
}

// Get 读取目录已保存设置；无记录返回空字段对象（Path 已填，空字段=未保存）。
// 路径错误透传 filesystem sentinel errors；数据库失败包装为 ErrDatabase。
func (s *SettingsService) Get(path string) (*model.FolderSettings, error) {
	_, relPath, err := filesystem.Resolve(s.fs.Root(), path)
	if err != nil {
		return nil, err
	}

	saved, err := s.settings.GetByPath(relPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	if saved == nil {
		return &model.FolderSettings{Path: relPath}, nil
	}
	return saved, nil
}

// Save 保存目录设置：非法字段容错规范化为默认值而非报错，返回规范化后的设置。
// 排序：sort_mode 非法 → filename，sort_direction 非法 → asc；
// regex 字段仅在 sort_mode=regex 时持久化，其余模式清空（DATA_MODEL §2.2）；
// 正则合法性不在此校验，由使用方（/folders、sort/preview）容错处理。
// spread：page_mode 非法 → single，read_order 非法 → left_to_right，
// wide_ratio 缺省/≤0 → 1.0，single_first_page 缺省 → true，single_last_page 缺省 → false。
func (s *SettingsService) Save(path string, settings model.FolderSettings) (*model.FolderSettings, error) {
	_, relPath, err := filesystem.Resolve(s.fs.Root(), path)
	if err != nil {
		return nil, err
	}

	mode := settings.SortMode
	if !validSortMode(mode) {
		mode = model.SortModeFilename
	}
	direction := settings.SortDirection
	if direction != model.SortDirectionAsc && direction != model.SortDirectionDesc {
		direction = model.SortDirectionAsc
	}
	regexPattern, regexConfig := "", ""
	if mode == model.SortModeRegex {
		regexPattern, regexConfig = settings.RegexPattern, settings.RegexConfig
	}

	pageMode := settings.PageMode
	if pageMode != model.PageModeSingle && pageMode != model.PageModeSpread {
		pageMode = model.PageModeSingle
	}
	readOrder := settings.ReadOrder
	if readOrder != model.ReadOrderLeftToRight && readOrder != model.ReadOrderRightToLeft {
		readOrder = model.ReadOrderLeftToRight
	}
	wideRatio := 1.0
	if settings.WideRatio != nil && *settings.WideRatio > 0 {
		wideRatio = *settings.WideRatio
	}
	singleFirstPage := true
	if settings.SingleFirstPage != nil {
		singleFirstPage = *settings.SingleFirstPage
	}
	singleLastPage := false
	if settings.SingleLastPage != nil {
		singleLastPage = *settings.SingleLastPage
	}

	saved := model.FolderSettings{
		Path:            relPath,
		SortMode:        mode,
		SortDirection:   direction,
		RegexPattern:    regexPattern,
		RegexConfig:     regexConfig,
		PageMode:        pageMode,
		ReadOrder:       readOrder,
		WideRatio:       &wideRatio,
		SingleFirstPage: &singleFirstPage,
		SingleLastPage:  &singleLastPage,
	}

	folderID, err := s.folders.GetOrCreateID(relPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	if err := s.settings.Upsert(folderID, saved); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return &saved, nil
}

// validSortMode 判断是否为受支持的排序模式。
func validSortMode(mode string) bool {
	switch mode {
	case model.SortModeFilename, model.SortModeNatural, model.SortModeModifiedTime,
		model.SortModeCreatedTime, model.SortModeFileSize, model.SortModeRegex:
		return true
	}
	return false
}
