package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"albumshelf/backend/internal/model"
)

// QuickAccessRepository quick_access 表访问（快捷访问：钉住的目录）。
// 目录钉住为全局用户数据，路径语义与 model.File.Path 一致。
type QuickAccessRepository struct {
	db *sql.DB
}

// NewQuickAccessRepository 构造 QuickAccessRepository。
func NewQuickAccessRepository(db *sql.DB) *QuickAccessRepository {
	return &QuickAccessRepository{db: db}
}

// List 返回全部固定目录，按固定时间倒序（新固定在前）。
func (r *QuickAccessRepository) List() ([]model.QuickAccess, error) {
	rows, err := r.db.Query(`
SELECT path, created_at FROM quick_access ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 quick_access 失败: %w", err)
	}
	defer rows.Close()

	var items []model.QuickAccess
	for rows.Next() {
		var q model.QuickAccess
		if err := rows.Scan(&q.Path, &q.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描 quick_access 失败: %w", err)
		}
		items = append(items, q)
	}
	return items, rows.Err()
}

// Add 新增固定目录；path 唯一约束下重复插入静默忽略（幂等）。
func (r *QuickAccessRepository) Add(path string) error {
	if _, err := r.db.Exec(`
INSERT INTO quick_access (path, created_at) VALUES (?, CURRENT_TIMESTAMP)
ON CONFLICT(path) DO NOTHING`, path); err != nil {
		return fmt.Errorf("写入 quick_access 失败: %w", err)
	}
	return nil
}

// Delete 删除指定固定目录；记录不存在时不报错（幂等）。
func (r *QuickAccessRepository) Delete(path string) error {
	if _, err := r.db.Exec(`DELETE FROM quick_access WHERE path = ?`, path); err != nil {
		return fmt.Errorf("删除 quick_access 失败: %w", err)
	}
	return nil
}

// Exists 判断路径是否已固定。
func (r *QuickAccessRepository) Exists(path string) (bool, error) {
	var one int
	err := r.db.QueryRow(`SELECT 1 FROM quick_access WHERE path = ?`, path).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("查询 quick_access 失败: %w", err)
	}
	return true, nil
}
