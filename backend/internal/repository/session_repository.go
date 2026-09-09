package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"imageshelf/backend/internal/model"
)

// SessionRepository sessions 表访问（SPRINT7 §5.3）。
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository 构造 SessionRepository。
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create 写入新会话（token 为主键）。
func (r *SessionRepository) Create(ctx context.Context, s model.Session) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO sessions (token, created_at, expires_at) VALUES (?, ?, ?)`,
		s.Token, s.CreatedAt, s.ExpiresAt)
	if err != nil {
		return fmt.Errorf("写入 sessions 失败: %w", err)
	}
	return nil
}

// GetByToken 按 token 查询会话；不存在返回 (nil, nil)。
func (r *SessionRepository) GetByToken(ctx context.Context, token string) (*model.Session, error) {
	var s model.Session
	err := r.db.QueryRowContext(ctx,
		`SELECT token, created_at, expires_at FROM sessions WHERE token = ?`, token).
		Scan(&s.Token, &s.CreatedAt, &s.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 sessions 失败: %w", err)
	}
	return &s, nil
}

// Delete 删除指定 token 的会话（登出 / 过期惰性删除）。
func (r *SessionRepository) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("删除 sessions 失败: %w", err)
	}
	return nil
}

// DeleteExpired 删除全部已过期会话，返回删除行数。
func (r *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, now)
	if err != nil {
		return 0, fmt.Errorf("清理过期 sessions 失败: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("读取清理行数失败: %w", err)
	}
	return n, nil
}
