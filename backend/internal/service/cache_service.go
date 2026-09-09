package service

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"imageshelf/backend/internal/repository"
)

// 缓存管理（SPRINT7 §5.6）：统计与清理均不加锁，自用场景容忍与缩略图生成
// 并发时的竞态；清理期间浏览对应图片会触发缓存重建。

// VariantCacheStats 单变体缓存统计（以磁盘为准）。
type VariantCacheStats struct {
	Count int64 `json:"count"`
	Bytes int64 `json:"bytes"`
}

// CacheStats 缓存统计结果。
type CacheStats struct {
	Thumb   VariantCacheStats `json:"thumb"`
	Preview VariantCacheStats `json:"preview"`
}

// CacheCleanupResult 清理结果。
type CacheCleanupResult struct {
	RemovedFiles int64 `json:"removed_files"`
	RemovedBytes int64 `json:"removed_bytes"`
}

// CacheService 缩略图/预览图缓存管理：磁盘统计、孤儿清理、全部清理。
type CacheService struct {
	dataDir   string
	imageRoot string
	cache     *repository.ImageCacheRepository
}

// NewCacheService 构造 CacheService。
func NewCacheService(dataDir, imageRoot string, cache *repository.ImageCacheRepository) *CacheService {
	return &CacheService{dataDir: dataDir, imageRoot: imageRoot, cache: cache}
}

// Stats 扫描 dataDir/cache/{thumb,preview} 目录统计文件数与字节数（以磁盘为准）。
func (s *CacheService) Stats() (*CacheStats, error) {
	thumb, err := scanVariantDir(filepath.Join(s.dataDir, "cache", "thumb"))
	if err != nil {
		return nil, err
	}
	preview, err := scanVariantDir(filepath.Join(s.dataDir, "cache", "preview"))
	if err != nil {
		return nil, err
	}
	return &CacheStats{Thumb: thumb, Preview: preview}, nil
}

// CleanupOrphans 清理两类孤儿（SPRINT7 §5.6）：
//  1. 索引行的 source_path 在 IMAGE_ROOT 已不存在 → 删缓存文件 + 删索引行；
//  2. 磁盘缓存文件不在索引 cache_path 集合 → 删文件（索引行已删但旧文件遗留）。
func (s *CacheService) CleanupOrphans() (*CacheCleanupResult, error) {
	result := &CacheCleanupResult{}
	items, err := s.cache.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	// 第一类：源文件已不存在的索引行
	indexed := make(map[string]bool, len(items))
	var staleIDs []int64
	for _, it := range items {
		indexed[it.CachePath] = true
		if _, err := os.Lstat(filepath.Join(s.imageRoot, it.SourcePath)); errors.Is(err, fs.ErrNotExist) {
			size, err := removeFile(it.CachePath)
			if err != nil {
				return nil, err
			}
			result.RemovedFiles++
			result.RemovedBytes += size
			staleIDs = append(staleIDs, it.ID)
		}
	}
	if err := s.cache.DeleteByIDs(ctx, staleIDs); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	// 第二类：磁盘文件不在索引 cache_path 集合
	removed, bytes, err := removeUntrackedFiles(s.dataDir, indexed)
	if err != nil {
		return nil, err
	}
	result.RemovedFiles += removed
	result.RemovedBytes += bytes
	return result, nil
}

// CleanupAll 清空两个缓存目录内容 + 清空 image_cache 索引。
func (s *CacheService) CleanupAll() (*CacheCleanupResult, error) {
	result := &CacheCleanupResult{}
	// indexed 为 nil → 全部文件视为未跟踪，整目录清空
	removed, bytes, err := removeUntrackedFiles(s.dataDir, nil)
	if err != nil {
		return nil, err
	}
	result.RemovedFiles = removed
	result.RemovedBytes = bytes
	if err := s.cache.DeleteAll(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return result, nil
}

// scanVariantDir 统计单个缓存子目录（目录不存在视为空）。
func scanVariantDir(dir string) (VariantCacheStats, error) {
	var stats VariantCacheStats
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return stats, nil
		}
		return stats, fmt.Errorf("读取缓存目录 %s 失败: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue // 并发生成中的文件可能已消失，跳过
		}
		stats.Count++
		stats.Bytes += info.Size()
	}
	return stats, nil
}

// removeFile 删除文件并返回其大小；文件已不存在视为成功。
func removeFile(path string) (int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("读取缓存文件 %s 失败: %w", path, err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return 0, fmt.Errorf("删除缓存文件 %s 失败: %w", path, err)
	}
	return info.Size(), nil
}

// removeUntrackedFiles 删除两个缓存目录中不在索引 cache_path 集合内的文件。
func removeUntrackedFiles(dataDir string, indexed map[string]bool) (int64, int64, error) {
	var removedFiles, removedBytes int64
	for _, sub := range []string{"thumb", "preview"} {
		dir := filepath.Join(dataDir, "cache", sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return 0, 0, fmt.Errorf("读取缓存目录 %s 失败: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			full := filepath.Join(dir, e.Name())
			if indexed != nil && indexed[full] {
				continue
			}
			size, err := removeFile(full)
			if err != nil {
				return 0, 0, err
			}
			removedFiles++
			removedBytes += size
		}
	}
	return removedFiles, removedBytes, nil
}
