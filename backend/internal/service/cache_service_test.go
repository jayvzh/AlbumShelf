package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
	"albumshelf/backend/internal/thumbnail"
)

// newTestCacheEnv 构造含「正常缓存 + 两类孤儿」的缓存环境：
//   - 正常：/Keep/ok.jpg 源存在，thumb 索引与磁盘文件齐全；
//   - 孤儿一：/Gone/deleted.jpg 源已删除，索引与磁盘文件遗留；
//   - 孤儿二：untracked.jpg 磁盘存在但无索引行（原图变更后旧文件遗留形态）。
func newTestCacheEnv(t *testing.T) (*CacheService, *repository.ImageCacheRepository, string) {
	t.Helper()

	imageRoot := t.TempDir()
	dataDir := t.TempDir()

	keepSrc := filepath.Join(imageRoot, "Keep", "ok.jpg")
	if err := os.MkdirAll(filepath.Dir(keepSrc), 0o755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}
	if err := os.WriteFile(keepSrc, []byte("jpg"), 0o644); err != nil {
		t.Fatalf("写入源文件失败: %v", err)
	}

	db, err := repository.Open(dataDir)
	if err != nil {
		t.Fatalf("打开临时数据库失败: %v", err)
	}
	t.Cleanup(func() { repository.Close(db) })

	cacheRepo := repository.NewImageCacheRepository(db)
	svc := NewCacheService(dataDir, imageRoot, cacheRepo)

	now := time.Now()
	writeCache := func(sourcePath, cachePath string, inIndex bool) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
			t.Fatalf("创建缓存目录失败: %v", err)
		}
		if err := os.WriteFile(cachePath, []byte("jpeg-data"), 0o644); err != nil {
			t.Fatalf("写入缓存文件失败: %v", err)
		}
		if !inIndex {
			return
		}
		if err := cacheRepo.Upsert(ctx, model.ThumbnailCache{
			SourcePath:  sourcePath,
			Variant:     model.VariantThumb,
			CachePath:   cachePath,
			SourceMtime: 100,
			SourceSize:  3,
			Width:       10,
			Height:      10,
			CreatedAt:   now,
		}); err != nil {
			t.Fatalf("写入索引失败: %v", err)
		}
	}

	okPath, err := thumbnail.CachePath(dataDir, "/Keep/ok.jpg", model.VariantThumb, 200, 100, 3)
	if err != nil {
		t.Fatalf("计算缓存路径失败: %v", err)
	}
	gonePath, err := thumbnail.CachePath(dataDir, "/Gone/deleted.jpg", model.VariantThumb, 200, 100, 3)
	if err != nil {
		t.Fatalf("计算缓存路径失败: %v", err)
	}
	writeCache("/Keep/ok.jpg", okPath, true)                        // 正常
	writeCache("/Gone/deleted.jpg", gonePath, true)                 // 孤儿一：源不存在
	writeCache("", filepath.Join(dataDir, "cache", "thumb", "untracked.jpg"), false) // 孤儿二：无索引

	return svc, cacheRepo, dataDir
}

// 统计以磁盘为准：thumb 3 个文件、preview 空目录为 0。
func TestCacheStatsCountsDiskFiles(t *testing.T) {
	svc, _, _ := newTestCacheEnv(t)

	// preview 目录尚未创建 → 统计为 0 不报错
	stats, err := svc.Stats()
	if err != nil {
		t.Fatalf("Stats 失败: %v", err)
	}
	if stats.Thumb.Count != 3 || stats.Thumb.Bytes != 3*int64(len("jpeg-data")) {
		t.Fatalf("thumb 统计错误: %+v", stats.Thumb)
	}
	if stats.Preview.Count != 0 || stats.Preview.Bytes != 0 {
		t.Fatalf("preview 统计应为空: %+v", stats.Preview)
	}
}

// 孤儿清理：两类孤儿均被清理，正常缓存不受影响。
func TestCacheCleanupOrphans(t *testing.T) {
	svc, cacheRepo, dataDir := newTestCacheEnv(t)

	result, err := svc.CleanupOrphans()
	if err != nil {
		t.Fatalf("CleanupOrphans 失败: %v", err)
	}
	if result.RemovedFiles != 2 || result.RemovedBytes != 2*int64(len("jpeg-data")) {
		t.Fatalf("清理结果错误: %+v", result)
	}

	// 磁盘：仅正常缓存保留
	entries, err := os.ReadDir(filepath.Join(dataDir, "cache", "thumb"))
	if err != nil {
		t.Fatalf("读取 thumb 目录失败: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("应仅剩 1 个正常缓存文件，got %d", len(entries))
	}

	// 索引：孤儿一的行已删除
	items, err := cacheRepo.List(ctx)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(items) != 1 || items[0].SourcePath != "/Keep/ok.jpg" {
		t.Fatalf("索引应仅剩正常行: %+v", items)
	}
}

// 全部清理：目录与索引皆空。
func TestCacheCleanupAll(t *testing.T) {
	svc, cacheRepo, _ := newTestCacheEnv(t)

	result, err := svc.CleanupAll()
	if err != nil {
		t.Fatalf("CleanupAll 失败: %v", err)
	}
	if result.RemovedFiles != 3 {
		t.Fatalf("应清理 3 个文件，got %d", result.RemovedFiles)
	}

	stats, err := svc.Stats()
	if err != nil {
		t.Fatalf("Stats 失败: %v", err)
	}
	if stats.Thumb.Count != 0 || stats.Preview.Count != 0 {
		t.Fatalf("清理后统计应为 0: %+v", stats)
	}
	items, err := cacheRepo.List(ctx)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("清理后索引应为空，got %d 行", len(items))
	}
}
