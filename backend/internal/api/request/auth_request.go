package request

// LoginRequest 登录请求体（SPRINT7 §5.2）。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
