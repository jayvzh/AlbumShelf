package repository

import (
	"context"
	"database/sql"
	"fmt"

	"albumshelf/backend/internal/model"
)

// ImageCacheRepository image_cache 表（缩略图/预览图缓存索引）的访问封装。
type ImageCacheRepository struct {
	db *sql.DB
}

// NewImageCacheRepository 构造 ImageCacheRepository。
func NewImageCacheRepository(db *sql.DB) *ImageCacheRepository {
	return &ImageCacheRepository{db: db}
}

// Upsert 插入一条缓存记录；cache_path 已存在时整行覆盖更新。
func (r *ImageCacheRepository) Upsert(ctx context.Context, record model.ThumbnailCache) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO image_cache
    (source_path, variant, cache_path, source_mtime, source_size, width, height, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(cache_path) DO UPDATE SET
    source_path  = excluded.source_path,
    variant      = excluded.variant,
    source_mtime = excluded.source_mtime,
    source_size  = excluded.source_size,
    width        = excluded.width,
    height       = excluded.height,
    created_at   = excluded.created_at`,
		record.SourcePath, record.Variant, record.CachePath,
		record.SourceMtime, record.SourceSize,
		record.Width, record.Height, record.CreatedAt)
	if err != nil {
		return fmt.Errorf("写入 image_cache 失败: %w", err)
	}
	return nil
}

// DeleteBySource 删除指定原图某变体的全部缓存记录（原图删除/失效重建前调用）。
func (r *ImageCacheRepository) DeleteBySource(ctx context.Context, sourcePath, variant string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM image_cache WHERE source_path = ? AND variant = ?`,
		sourcePath, variant)
	if err != nil {
		return fmt.Errorf("删除 image_cache 失败: %w", err)
	}
	return nil
}

// ImageCacheRow image_cache 表行视图（含 id，缓存管理清理用）。
type ImageCacheRow struct {
	ID int64
	model.ThumbnailCache
}

// List 返回全部缓存索引行（缓存管理统计/孤儿清理用，SPRINT7 §5.6）。
func (r *ImageCacheRepository) List(ctx context.Context) ([]ImageCacheRow, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, source_path, variant, cache_path, source_mtime, source_size, width, height, created_at
FROM image_cache`)
	if err != nil {
		return nil, fmt.Errorf("查询 image_cache 失败: %w", err)
	}
	defer rows.Close()

	var items []ImageCacheRow
	for rows.Next() {
		var it ImageCacheRow
		if err := rows.Scan(&it.ID, &it.SourcePath, &it.Variant, &it.CachePath,
			&it.SourceMtime, &it.SourceSize, &it.Width, &it.Height, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描 image_cache 失败: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// DeleteByIDs 按 id 批量删除索引行（孤儿清理用）。
func (r *ImageCacheRepository) DeleteByIDs(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `DELETE FROM image_cache WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("预编译删除语句失败: %w", err)
	}
	defer stmt.Close()
	for _, id := range ids {
		if _, err := stmt.ExecContext(ctx, id); err != nil {
			return fmt.Errorf("删除 image_cache 行失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

// DeleteAll 清空 image_cache 表（全部清理用）。
func (r *ImageCacheRepository) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM image_cache`)
	if err != nil {
		return fmt.Errorf("清空 image_cache 失败: %w", err)
	}
	return nil
}

// DeleteByVariant 删除指定变体（thumb/preview）的全部索引行（按变体清理用）。
func (r *ImageCacheRepository) DeleteByVariant(ctx context.Context, variant string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM image_cache WHERE variant = ?`, variant)
	if err != nil {
		return fmt.Errorf("按变体删除 image_cache 失败: %w", err)
	}
	return nil
}
