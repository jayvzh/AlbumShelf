// Package auth 提供单管理员登录认证与会话校验（SPRINT7 §5.2）。
// 凭证来自环境变量（config.Config），token 为 crypto/rand 32 字节 hex，
// 持久化于 sessions 表，固定过期不滑动续期。
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"time"

	"imageshelf/backend/internal/model"
	"imageshelf/backend/internal/repository"
)

// ctx 后台上下文（自用单管理员场景，无请求级取消需求）。
var ctx = context.Background()

// CookieName 会话 Cookie 名。
const CookieName = "session_token"

// tokenBytes 随机 token 字节数（hex 编码后 64 字符）。
const tokenBytes = 32

// 认证服务 sentinel 错误。
var (
	// ErrInvalidCredentials 用户名或密码错误（未知用户与错误密码同码，防时序区分）。
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrDisabled 认证未启用时调用登录。
	ErrDisabled = errors.New("认证未启用")
)

// Service 单管理员认证服务。
type Service struct {
	username string
	password string
	maxAge   time.Duration
	sessions *repository.SessionRepository
}

// New 构造认证服务。password 为空表示认证未启用。
func New(username, password string, maxAge time.Duration, sessions *repository.SessionRepository) *Service {
	return &Service{
		username: username,
		password: password,
		maxAge:   maxAge,
		sessions: sessions,
	}
}

// Enabled 返回认证是否启用（AUTH_PASSWORD 非空）。
func (s *Service) Enabled() bool {
	return s.password != ""
}

// MaxAgeSeconds 返回 Cookie MaxAge（秒）。
func (s *Service) MaxAgeSeconds() int {
	return int(s.maxAge.Seconds())
}

// Login 校验凭证；成功时生成 token 写入 sessions 表并返回 (token, nil)。
// 用户名与密码均用常量时间比较，未知用户也执行密码比较，避免时序侧信道。
func (s *Service) Login(username, password string) (string, error) {
	if !s.Enabled() {
		return "", ErrDisabled
	}
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(s.password)) == 1
	if !userOK || !passOK {
		return "", ErrInvalidCredentials
	}

	token, err := newToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	if err := s.sessions.Create(ctx, model.Session{
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(s.maxAge),
	}); err != nil {
		return "", err
	}
	return token, nil
}

// Logout 删除指定会话（Cookie 清理由 handler 负责）。
func (s *Service) Logout(token string) error {
	return s.sessions.Delete(ctx, token)
}

// Validate 判断 token 是否为有效未过期会话；命中的过期记录惰性删除。
func (s *Service) Validate(token string) bool {
	if token == "" {
		return false
	}
	sess, err := s.sessions.GetByToken(ctx, token)
	if err != nil || sess == nil {
		return false
	}
	if !sess.IsValid(time.Now()) {
		_ = s.sessions.Delete(ctx, token)
		return false
	}
	return true
}

// newToken 生成 32 字节 crypto/rand 的 hex 编码 token。
func newToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
