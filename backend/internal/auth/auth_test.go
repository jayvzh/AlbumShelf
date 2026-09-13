package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
)

// newTestService 构造真实 SQLite 会话存储的认证服务（SPRINT7 §5.2 测试要求）。
func newTestService(t *testing.T) (*Service, *repository.SessionRepository) {
	t.Helper()
	db, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	sessions := repository.NewSessionRepository(db)
	return New("admin", "secret", time.Hour, sessions), sessions
}

func TestLoginSuccess(t *testing.T) {
	svc, sessions := newTestService(t)

	token, err := svc.Login("admin", "secret")
	if err != nil {
		t.Fatalf("正确凭证应登录成功，got %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("token 应为 32 字节 hex（64 字符），got %d", len(token))
	}
	if !svc.Validate(token) {
		t.Fatal("登录后的 token 应有效")
	}
	row, err := sessions.GetByToken(context.Background(), token)
	if err != nil || row == nil {
		t.Fatalf("token 应已写入 sessions 表: %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Login("admin", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("错误密码应返回 ErrInvalidCredentials，got %v", err)
	}
}

func TestLoginUnknownUserSameError(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Login("nobody", "secret")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("未知用户应与错误密码同码（ErrInvalidCredentials），got %v", err)
	}
}

func TestValidateExpiredToken(t *testing.T) {
	svc, sessions := newTestService(t)

	// 直接插入一条已过期会话
	if err := sessions.Create(context.Background(), model.Session{
		Token:     "expired-token",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("插入过期会话失败: %v", err)
	}
	if svc.Validate("expired-token") {
		t.Fatal("过期 token 应无效")
	}
	// 惰性删除：过期记录应已被清理
	row, _ := sessions.GetByToken(context.Background(), "expired-token")
	if row != nil {
		t.Fatal("过期记录应被惰性删除")
	}
}

func TestLoginDisabled(t *testing.T) {
	db, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := New("admin", "", time.Hour, repository.NewSessionRepository(db))

	if svc.Enabled() {
		t.Fatal("密码为空时认证应未启用")
	}
	if _, err := svc.Login("admin", "secret"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("未启用时登录应返回 ErrDisabled，got %v", err)
	}
}

func TestLogoutInvalidates(t *testing.T) {
	svc, _ := newTestService(t)

	token, err := svc.Login("admin", "secret")
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if err := svc.Logout(token); err != nil {
		t.Fatalf("登出失败: %v", err)
	}
	if svc.Validate(token) {
		t.Fatal("登出后的 token 应无效")
	}
}
