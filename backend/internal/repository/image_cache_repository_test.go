package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"albumshelf/backend/internal/model"
)

// 同一 cache_path 重复 Upsert 不产生重复行，且字段被覆盖更新。
func TestImageCacheUpsertOverwrites(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)
	repo := NewImageCacheRepository(db)
	ctx := context.Background()

	cachePath := filepath.Join(dataDir, "cache", "thumb", "a.jpg")
	rec1 := model.ThumbnailCache{
		SourcePath: "/Comics/001.jpg", Variant: model.VariantThumb, CachePath: cachePath,
		SourceMtime: 100, SourceSize: 10, Width: 300, Height: 200, CreatedAt: time.Now(),
	}
	rec2 := rec1
	rec2.SourceMtime = 200
	rec2.Width = 299

	if err := repo.Upsert(ctx, rec1); err != nil {
		t.Fatalf("第一次 Upsert 失败: %v", err)
	}
	if err := repo.Upsert(ctx, rec2); err != nil {
		t.Fatalf("第二次 Upsert 失败: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM image_cache WHERE cache_path = ?`, cachePath).Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("期望 1 行，实际 %d 行", count)
	}

	var mtime, width int64
	if err := db.QueryRow(`SELECT source_mtime, width FROM image_cache WHERE cache_path = ?`, cachePath).Scan(&mtime, &width); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if mtime != 200 || width != 299 {
		t.Fatalf("覆盖更新未生效: mtime=%d width=%d，期望 200/299", mtime, width)
	}
}

// DeleteBySource 只删除指定原图对应变体的记录。
func TestImageCacheDeleteBySource(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)
	repo := NewImageCacheRepository(db)
	ctx := context.Background()

	mk := func(source, variant, cache string) model.ThumbnailCache {
		return model.ThumbnailCache{
			SourcePath: source, Variant: variant, CachePath: cache,
			SourceMtime: 1, SourceSize: 2, Width: 3, Height: 4, CreatedAt: time.Now(),
		}
	}
	for _, r := range []model.ThumbnailCache{
		mk("/Comics/001.jpg", model.VariantThumb, "/data/cache/thumb/1.jpg"),
		mk("/Comics/001.jpg", model.VariantPreview, "/data/cache/preview/1.jpg"),
		mk("/Comics/002.jpg", model.VariantThumb, "/data/cache/thumb/2.jpg"),
	} {
		if err := repo.Upsert(ctx, r); err != nil {
			t.Fatalf("Upsert 失败: %v", err)
		}
	}

	if err := repo.DeleteBySource(ctx, "/Comics/001.jpg", model.VariantThumb); err != nil {
		t.Fatalf("DeleteBySource 失败: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM image_cache`).Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 2 {
		t.Fatalf("删除后期望剩 2 行，实际 %d 行", count)
	}
	var remains int
	if err := db.QueryRow(`SELECT COUNT(*) FROM image_cache WHERE source_path = ? AND variant = ?`,
		"/Comics/001.jpg", model.VariantThumb).Scan(&remains); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if remains != 0 {
		t.Fatalf("目标记录未删除，仍剩 %d 行", remains)
	}
}

// Close 后对同一 dataDir 重新 Open：schema 幂等不报错，且可继续写入。
func TestReopenSchemaIdempotent(t *testing.T) {
	dataDir := t.TempDir()
	db1, err := Open(dataDir)
	if err != nil {
		t.Fatalf("第一次 Open 失败: %v", err)
	}
	if err := Close(db1); err != nil {
		t.Fatalf("Close 失败: %v", err)
	}

	db2, err := Open(dataDir)
	if err != nil {
		t.Fatalf("重新 Open 失败: %v", err)
	}
	defer Close(db2)

	repo := NewImageCacheRepository(db2)
	rec := model.ThumbnailCache{
		SourcePath: "/a.jpg", Variant: model.VariantThumb, CachePath: "/data/cache/thumb/x.jpg",
		SourceMtime: 1, SourceSize: 2, Width: 10, Height: 20, CreatedAt: time.Now(),
	}
	if err := repo.Upsert(context.Background(), rec); err != nil {
		t.Fatalf("重开后 Upsert 失败: %v", err)
	}
}
