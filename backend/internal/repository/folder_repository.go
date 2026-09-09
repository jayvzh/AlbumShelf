package repository

import (
	"database/sql"
	"errors"
	"fmt"
)

// FolderRepository folders 表访问（DATA_MODEL §2.1）：随浏览行为懒创建目录记录。
type FolderRepository struct {
	db *sql.DB
}

// NewFolderRepository 构造 FolderRepository。
func NewFolderRepository(db *sql.DB) *FolderRepository {
	return &FolderRepository{db: db}
}

// GetOrCreateID 返回 path 对应目录记录 ID；不存在时先插入（懒创建，DATA_MODEL §2.1）。
func (r *FolderRepository) GetOrCreateID(path string) (int64, error) {
	var id int64
	err := r.db.QueryRow(`SELECT id FROM folders WHERE path = ?`, path).Scan(&id)
	switch {
	case err == nil:
		return id, nil
	case !errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("查询 folders 失败: %w", err)
	}

	if _, err := r.db.Exec(
		`INSERT INTO folders (path, created_at, updated_at) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		path,
	); err != nil {
		return 0, fmt.Errorf("插入 folders 失败: %w", err)
	}
	if err := r.db.QueryRow(`SELECT id FROM folders WHERE path = ?`, path).Scan(&id); err != nil {
		return 0, fmt.Errorf("回读 folders id 失败: %w", err)
	}
	return id, nil
}
