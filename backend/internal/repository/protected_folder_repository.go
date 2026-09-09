package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"imageshelf/backend/internal/model"
)

// ProtectedFolderRepository protected_folders 表访问（SPRINT7 §5.3）。
type ProtectedFolderRepository struct {
	db *sql.DB
}

// NewProtectedFolderRepository 构造 ProtectedFolderRepository。
func NewProtectedFolderRepository(db *sql.DB) *ProtectedFolderRepository {
	return &ProtectedFolderRepository{db: db}
}

// List 返回全部私有目录（按 path 排序）。
func (r *ProtectedFolderRepository) List(ctx context.Context) ([]model.ProtectedFolder, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, path, created_at FROM protected_folders ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("查询 protected_folders 失败: %w", err)
	}
	defer rows.Close()

	var items []model.ProtectedFolder
	for rows.Next() {
		var it model.ProtectedFolder
		if err := rows.Scan(&it.ID, &it.Path, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描 protected_folders 失败: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ReplaceAll 事务内全量替换（DELETE 全部 + INSERT，SPRINT7 §5.5）。
func (r *ProtectedFolderRepository) ReplaceAll(ctx context.Context, paths []string, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM protected_folders`); err != nil {
		return fmt.Errorf("清空 protected_folders 失败: %w", err)
	}
	for _, p := range paths {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO protected_folders (path, created_at) VALUES (?, ?)`, p, now); err != nil {
			return fmt.Errorf("写入 protected_folders 失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

// UpsertPaths 按 path 合并追加（配置导入用，已存在则跳过）。
func (r *ProtectedFolderRepository) UpsertPaths(ctx context.Context, paths []string, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, p := range paths {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO protected_folders (path, created_at) VALUES (?, ?)
ON CONFLICT(path) DO NOTHING`, p, now); err != nil {
			return fmt.Errorf("合并 protected_folders 失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}
