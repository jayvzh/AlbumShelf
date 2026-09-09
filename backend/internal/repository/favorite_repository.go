package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"imageshelf/backend/internal/model"
)

// FavoriteRepository favorites 表访问（DATA_MODEL §2.3）。
// 收藏为全局用户数据，路径语义与 model.File.Path 一致。
type FavoriteRepository struct {
	db *sql.DB
}

// NewFavoriteRepository 构造 FavoriteRepository。
func NewFavoriteRepository(db *sql.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

// List 返回全部收藏，按收藏时间倒序（新收藏在前）。
func (r *FavoriteRepository) List() ([]model.Favorite, error) {
	rows, err := r.db.Query(`
SELECT file_path, created_at FROM favorites ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 favorites 失败: %w", err)
	}
	defer rows.Close()

	var items []model.Favorite
	for rows.Next() {
		var f model.Favorite
		if err := rows.Scan(&f.Path, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描 favorites 失败: %w", err)
		}
		items = append(items, f)
	}
	return items, rows.Err()
}

// Add 新增收藏；file_path 唯一约束下重复插入静默忽略（幂等）。
func (r *FavoriteRepository) Add(path string) error {
	if _, err := r.db.Exec(`
INSERT INTO favorites (file_path, created_at) VALUES (?, CURRENT_TIMESTAMP)
ON CONFLICT(file_path) DO NOTHING`, path); err != nil {
		return fmt.Errorf("写入 favorites 失败: %w", err)
	}
	return nil
}

// Delete 删除指定收藏；记录不存在时不报错（幂等）。
func (r *FavoriteRepository) Delete(path string) error {
	if _, err := r.db.Exec(`DELETE FROM favorites WHERE file_path = ?`, path); err != nil {
		return fmt.Errorf("删除 favorites 失败: %w", err)
	}
	return nil
}

// Exists 判断路径是否已收藏。
func (r *FavoriteRepository) Exists(path string) (bool, error) {
	var one int
	err := r.db.QueryRow(`SELECT 1 FROM favorites WHERE file_path = ?`, path).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("查询 favorites 失败: %w", err)
	}
	return true, nil
}

// DeleteMany 批量删除收藏（惰性清理失效记录用）；返回删除行数。
func (r *FavoriteRepository) DeleteMany(paths []string) (int64, error) {
	if len(paths) == 0 {
		return 0, nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启 favorites 清理事务失败: %w", err)
	}
	defer tx.Rollback()

	var total int64
	for _, p := range paths {
		res, err := tx.Exec(`DELETE FROM favorites WHERE file_path = ?`, p)
		if err != nil {
			return 0, fmt.Errorf("批量删除 favorites 失败: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("读取 favorites 删除行数失败: %w", err)
		}
		total += n
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交 favorites 清理事务失败: %w", err)
	}
	return total, nil
}
