package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// AppSettingsRepository app_settings 键值表（全局应用设置）的访问封装。
// 当前仅存缓存预热级别等少量键；值统一为字符串，类型解析由调用方负责。
type AppSettingsRepository struct {
	db *sql.DB
}

// NewAppSettingsRepository 构造 AppSettingsRepository。
func NewAppSettingsRepository(db *sql.DB) *AppSettingsRepository {
	return &AppSettingsRepository{db: db}
}

// Get 读取键值；键不存在时返回 ("", false, nil)。
func (r *AppSettingsRepository) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM app_settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("查询 app_settings 失败: %w", err)
	}
	return value, true, nil
}

// Set 写入键值（UPSERT）。
func (r *AppSettingsRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO app_settings (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("写入 app_settings 失败: %w", err)
	}
	return nil
}
