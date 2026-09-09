package service

import (
	"fmt"
	"log"

	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/model"
	"imageshelf/backend/internal/repository"
)

// FavoriteService 收藏业务（PRD F010 Phase 2）：全局图片收藏的列表与切换。
// 路径语义与 model.File.Path 一致（IMAGE_ROOT 相对绝对风格路径）。
type FavoriteService struct {
	fs   *filesystem.LocalFilesystem
	repo *repository.FavoriteRepository
}

// NewFavoriteService 构造 FavoriteService。
func NewFavoriteService(fs *filesystem.LocalFilesystem, repo *repository.FavoriteRepository) *FavoriteService {
	return &FavoriteService{fs: fs, repo: repo}
}

// List 返回全部收藏对应的图片信息，按收藏时间倒序。
// 已删除/移动的收藏条目惰性清理（记日志后从库中移除），不阻断列表返回。
func (s *FavoriteService) List() ([]model.Image, error) {
	favorites, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	images := make([]model.Image, 0, len(favorites))
	var stale []string
	for _, f := range favorites {
		img, err := s.fs.GetMetadata(f.Path)
		if err != nil {
			// 不存在/非法路径：记日志并纳入惰性清理，跳过该条目
			log.Printf("收藏条目失效，惰性清理 %s: %v", f.Path, err)
			stale = append(stale, f.Path)
			continue
		}
		images = append(images, img)
	}

	if len(stale) > 0 {
		if _, err := s.repo.DeleteMany(stale); err != nil {
			// 清理失败仅记日志，不影响本次列表
			log.Printf("惰性清理失效收藏失败: %v", err)
		}
	}
	return images, nil
}

// Toggle 切换指定图片的收藏状态，返回切换后的状态。
// 图片不存在或非图片格式时透传 filesystem sentinel 错误（handler 映射 404/400）。
func (s *FavoriteService) Toggle(path string) (favorited bool, err error) {
	// Resolve 校验路径安全并规范化（防目录穿越），GetMetadata 校验存在且为图片
	normalized, err := s.resolvePath(path)
	if err != nil {
		return false, err
	}
	if _, err := s.fs.GetMetadata(normalized); err != nil {
		return false, err
	}

	exists, err := s.repo.Exists(normalized)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	if exists {
		if err := s.repo.Delete(normalized); err != nil {
			return false, fmt.Errorf("%w: %v", ErrDatabase, err)
		}
		return false, nil
	}
	if err := s.repo.Add(normalized); err != nil {
		return false, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return true, nil
}

// resolvePath 路径安全校验与规范化（防目录穿越）。
func (s *FavoriteService) resolvePath(path string) (string, error) {
	_, relPath, err := filesystem.Resolve(s.fs.Root(), path)
	return relPath, err
}
