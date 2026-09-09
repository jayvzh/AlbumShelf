package model

import "time"

// Session 对应 sessions 表一行（SPRINT7 §5.3）。
// token 为 crypto/rand 32 字节的 hex 编码，固定有效期不滑动续期。
type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// IsValid 判断会话是否未过期。
func (s *Session) IsValid(now time.Time) bool {
	return now.Before(s.ExpiresAt)
}
