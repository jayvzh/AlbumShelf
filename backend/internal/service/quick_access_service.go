package service

import (
	"fmt"
	"log"
	"os"

	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/repository"
)

// QuickAccessService 快捷访问业务：钉住目录的列表与切换。
// 路径语义与 model.File.Path 一致（IMAGE_ROOT 相对绝对风格路径，根为 /）。
type QuickAccessService struct {
	fs   *filesystem.LocalFilesystem
	repo *repository.QuickAccessRepository
}

// NewQuickAccessService 构造 QuickAccessService。
func NewQuickAccessService(fs *filesystem.LocalFilesystem, repo *repository.QuickAccessRepository) *QuickAccessService {
	return &QuickAccessService{fs: fs, repo: repo}
}

// List 返回全部固定目录路径（固定时间倒序）。
// 已删除/移动的条目惰性清理（记日志后从库中移除），不阻断列表返回。
func (s *QuickAccessService) List() ([]string, error) {
	entries, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	paths := make([]string, 0, len(entries))
	for _, q := range entries {
		if _, err := s.resolveDir(q.Path); err != nil {
			log.Printf("快捷访问条目失效，惰性清理 %s: %v", q.Path, err)
			if err := s.repo.Delete(q.Path); err != nil {
				// 清理失败仅记日志，不影响本次列表
				log.Printf("惰性清理失效快捷访问失败: %v", err)
			}
			continue
		}
		paths = append(paths, q.Path)
	}
	return paths, nil
}

// Toggle 切换指定目录的固定状态，返回切换后的状态。
// 路径非法或目标非目录时透传 filesystem sentinel 错误（handler 映射 400/404）。
func (s *QuickAccessService) Toggle(path string) (pinned bool, err error) {
	relPath, err := s.resolveDir(path)
	if err != nil {
		return false, err
	}

	exists, err := s.repo.Exists(relPath)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	if exists {
		if err := s.repo.Delete(relPath); err != nil {
			return false, fmt.Errorf("%w: %v", ErrDatabase, err)
		}
		return false, nil
	}
	if err := s.repo.Add(relPath); err != nil {
		return false, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return true, nil
}

// resolveDir 路径安全校验与规范化（防目录穿越），并校验目标存在且为目录。
func (s *QuickAccessService) resolveDir(path string) (string, error) {
	absPath, relPath, err := filesystem.Resolve(s.fs.Root(), path)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(absPath)
	if err != nil {
		return "", filesystem.ErrNotFound
	}
	if !fi.IsDir() {
		return "", filesystem.ErrInvalidPath
	}
	return relPath, nil
}
