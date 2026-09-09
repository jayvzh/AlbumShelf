package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/repository"
)

// ctx 后台上下文（自用单管理员场景，无请求级取消需求）。
var ctx = context.Background()

// AppSettingsService 应用级设置业务：私有目录管理（SPRINT7 §5.5）。
type AppSettingsService struct {
	fs        *filesystem.LocalFilesystem
	protected *repository.ProtectedFolderRepository
}

// NewAppSettingsService 构造 AppSettingsService。
func NewAppSettingsService(fs *filesystem.LocalFilesystem, protected *repository.ProtectedFolderRepository) *AppSettingsService {
	return &AppSettingsService{fs: fs, protected: protected}
}

// ListPaths 返回全部私有目录（规范化相对路径，按字典序）。
func (s *AppSettingsService) ListPaths() ([]string, error) {
	return s.ProtectedRelPaths()
}

// ReplacePaths 全量替换私有目录（PUT 语义）：每项校验 + 规范化 + 去重排序，
// 返回规范化后的路径列表。非法/不存在路径透传 filesystem sentinel errors。
func (s *AppSettingsService) ReplacePaths(paths []string) ([]string, error) {
	normalized, err := s.normalize(paths)
	if err != nil {
		return nil, err
	}
	if err := s.protected.ReplaceAll(ctx, normalized, time.Now()); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return normalized, nil
}

// MergePaths 按 path 合并追加私有目录（配置导入语义），返回合并条数。
func (s *AppSettingsService) MergePaths(paths []string) (int, error) {
	normalized, err := s.normalize(paths)
	if err != nil {
		return 0, err
	}
	if err := s.protected.UpsertPaths(ctx, normalized, time.Now()); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return len(normalized), nil
}

// IsProtectedPath 判断用户输入路径（未规范化）是否命中私有子树；
// Resolve 失败（越界/不存在）返回 false——内容端点自身会返回 404/400。
func (s *AppSettingsService) IsProtectedPath(userPath string) bool {
	_, relPath, err := filesystem.Resolve(s.fs.Root(), userPath)
	if err != nil {
		return false
	}
	return s.IsProtectedRelPath(relPath)
}

// IsProtectedRelPath 判断已规范化相对路径是否命中私有子树：
// relPath == p || strings.HasPrefix(relPath, p+"/")，/Private 保护自身与全部子级，不误伤 /PrivateX。
func (s *AppSettingsService) IsProtectedRelPath(relPath string) bool {
	paths, err := s.ProtectedRelPaths()
	if err != nil {
		return false
	}
	for _, p := range paths {
		if relPath == p || strings.HasPrefix(relPath, p+"/") {
			return true
		}
	}
	return false
}

// ProtectedRelPaths 返回全部私有相对路径（数据库失败包装 ErrDatabase）。
func (s *AppSettingsService) ProtectedRelPaths() ([]string, error) {
	items, err := s.protected.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	paths := make([]string, 0, len(items))
	for _, it := range items {
		paths = append(paths, it.Path)
	}
	return paths, nil
}

// normalize 校验并规范化路径列表：空串与 "/" 拒绝（INVALID_PATH）；
// 每项 Resolve（非法/不存在透传 sentinel）；返回去重排序后的相对路径。
func (s *AppSettingsService) normalize(paths []string) ([]string, error) {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" || p == "/" {
			return nil, fmt.Errorf("%w: 空路径与根目录不允许", filesystem.ErrInvalidPath)
		}
		_, relPath, err := filesystem.Resolve(s.fs.Root(), p)
		if err != nil {
			return nil, err
		}
		if seen[relPath] {
			continue
		}
		seen[relPath] = true
		out = append(out, relPath)
	}
	sort.Strings(out)
	return out, nil
}
