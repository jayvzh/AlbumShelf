package service

import (
	"errors"
	"fmt"
	"log"
	"sort"

	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/model"
	"imageshelf/backend/internal/repository"
	"imageshelf/backend/internal/sorting"
)

// GuestGuard 游客目录过滤守卫（SPRINT7 §5.5）：auth enabled 且游客时，
// 目录列表剔除命中私有子树的条目。由 app 组装层注入实现。
type GuestGuard interface {
	// FilterActive 当前请求是否需要过滤（auth enabled && 游客）。
	FilterActive() bool
	// IsProtectedRelPath 判断已规范化相对路径是否命中私有子树。
	IsProtectedRelPath(relPath string) bool
}

// FolderListResult 目录列表业务结果。
type FolderListResult struct {
	Path    string
	Folders []model.Folder
	Images  []model.Image
}

// FolderService 目录浏览业务编排：扫描 → 排序。
// images 生效排序优先级：请求参数 > 该目录已保存的 folder_settings > 默认（API.md §3.2）。
type FolderService struct {
	fs       *filesystem.LocalFilesystem
	settings *repository.SettingsRepository
	engine   *sorting.Engine
}

// NewFolderService 构造 FolderService。
func NewFolderService(fs *filesystem.LocalFilesystem, settings *repository.SettingsRepository) *FolderService {
	return &FolderService{fs: fs, settings: settings, engine: sorting.NewEngine()}
}

// List 返回目录内容：folders 按名称自然排序固定返回；images 按生效排序参数返回。
// 可变参数 guards 提供游客私有目录过滤（管理请求不传）；
// 错误透传 filesystem 的 sentinel errors，由 handler 映射 HTTP 状态码。
func (s *FolderService) List(path string, opts model.SortOptions, guards ...GuestGuard) (*FolderListResult, error) {
	_, relPath, err := filesystem.Resolve(s.fs.Root(), path)
	if err != nil {
		return nil, err
	}

	images, folders, err := s.fs.ListDirectory(path)
	if err != nil {
		return nil, err
	}

	folders = filterGuestFolders(folders, relPath, guards)
	sort.SliceStable(folders, func(i, j int) bool {
		return sorting.NaturalCompare(folders[i].Name, folders[j].Name) < 0
	})
	images = s.sortImages(images, opts, relPath)

	return &FolderListResult{Path: relPath, Folders: folders, Images: images}, nil
}

// filterGuestFolders 游客视角剔除私有子树条目：子目录完整相对路径 =
// 父目录 relPath + 名称（根为 /Name 风格）。无激活 guard 时原样返回。
func filterGuestFolders(folders []model.Folder, parentRel string, guards []GuestGuard) []model.Folder {
	active := false
	for _, g := range guards {
		if g != nil && g.FilterActive() {
			active = true
			break
		}
	}
	if !active {
		return folders
	}

	out := make([]model.Folder, 0, len(folders))
	for _, f := range folders {
		child := parentRel + "/" + f.Name
		if parentRel == "/" {
			child = "/" + f.Name
		}
		hit := false
		for _, g := range guards {
			if g != nil && g.FilterActive() && g.IsProtectedRelPath(child) {
				hit = true
				break
			}
		}
		if !hit {
			out = append(out, f)
		}
	}
	return out
}

// sortImages 确定 images 的生效排序并执行：
// 请求未带 sort 参数（Mode 为空）时读已保存设置，读库失败仅记日志并回退默认
// （不阻断浏览）；请求带参数时直接使用，Engine 内部负责规范化非法值。
// 正则编译失败（保存了非法正则或请求参数非法）时记日志回退 filename asc，不阻断浏览。
func (s *FolderService) sortImages(images []model.Image, opts model.SortOptions, relPath string) []model.Image {
	if opts.Mode == "" {
		saved, err := s.settings.GetByPath(relPath)
		switch {
		case err != nil:
			log.Printf("读取目录 %s 已保存排序设置失败，回退默认排序: %v", relPath, err)
			opts = model.SortOptions{Mode: model.SortModeFilename, Direction: model.SortDirectionAsc}
		case saved != nil && saved.SortMode != "":
			direction := saved.SortDirection
			if direction == "" {
				direction = model.SortDirectionAsc
			}
			opts = model.SortOptions{
				Mode:      saved.SortMode,
				Direction: direction,
				Regex:     saved.RegexPattern,
				Rules:     parseSavedRules(saved.RegexConfig),
			}
		default:
			opts = model.SortOptions{Mode: model.SortModeFilename, Direction: model.SortDirectionAsc}
		}
	}
	if opts.Direction == "" {
		opts.Direction = model.SortDirectionAsc
	}

	sorted, err := s.engine.Sort(images, opts)
	if errors.Is(err, sorting.ErrInvalidRegex) {
		log.Printf("目录 %s 正则排序失败，回退 filename asc: %v", relPath, err)
		sorted, err = s.engine.Sort(images, model.SortOptions{
			Mode:      model.SortModeFilename,
			Direction: model.SortDirectionAsc,
		})
	}
	if err != nil {
		// 理论不可达（filename 排序无编译错误）；保底返回输入原序
		return images
	}
	return sorted
}

// parseSavedRules 解析已保存的 regex_config（容错：非法 JSON 仅记日志，
// 返回 nil 回退单规则简写），保证损坏的存量数据不阻断浏览。
func parseSavedRules(config string) []model.SortRule {
	rules, err := model.ParseSortRules(config)
	if err != nil {
		log.Printf("解析已保存 regex_config 失败，回退单规则简写: %v", err)
		return nil
	}
	return rules
}

// PreviewSort Regex 排序预览（API.md §3.7）：无状态纯函数预览，不落库、不校验路径。
// 仅支持 regex 模式；非法正则返回包装 sorting.ErrInvalidRegex 的错误（400 INVALID_REGEX）。
func (s *FolderService) PreviewSort(files []string, opts model.SortOptions) (*sorting.SortPreview, error) {
	if opts.Mode != model.SortModeRegex {
		return nil, fmt.Errorf("%w: mode 必须为 regex", sorting.ErrInvalidRegex)
	}
	return sorting.RegexPreview(files, opts)
}
