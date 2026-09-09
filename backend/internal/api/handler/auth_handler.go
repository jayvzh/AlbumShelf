package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/request"
	"imageshelf/backend/internal/api/response"
	"imageshelf/backend/internal/auth"
	"imageshelf/backend/internal/middleware"
)

// AuthHandler 处理认证 API（SPRINT7 §5.8）。
type AuthHandler struct {
	svc       *auth.Service
	username  string
	imageRoot string
	dataDir   string
}

// NewAuthHandler 构造认证 Handler。imageRoot/dataDir 仅在已登录的 status 响应中返回。
func NewAuthHandler(svc *auth.Service, username, imageRoot, dataDir string) *AuthHandler {
	return &AuthHandler{svc: svc, username: username, imageRoot: imageRoot, dataDir: dataDir}
}

// Login 处理 POST /api/v1/auth/login：校验凭证并种会话 Cookie。
// auth 未启用时 400 UNAUTHORIZED（SPRINT7 §5.1）。
func (h *AuthHandler) Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeUnauthorized, "Invalid request body"))
		return
	}
	if !h.svc.Enabled() {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeUnauthorized, "Auth disabled"))
		return
	}

	token, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, response.NewError(response.CodeInvalidCredentials, "Invalid credentials"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
		return
	}

	setSessionCookie(c, token, h.svc.MaxAgeSeconds())
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"authenticated": true, "username": h.username}})
}

// Logout 处理 POST /api/v1/auth/logout：删除会话记录并清 Cookie。
func (h *AuthHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie(auth.CookieName); err == nil && token != "" {
		if err := h.svc.Logout(token); err != nil {
			c.JSON(http.StatusInternalServerError, response.NewError(response.CodeDatabaseError, "Database error"))
			return
		}
	}
	setSessionCookie(c, "", -1)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"authenticated": false}})
}

// Status 处理 GET /api/v1/auth/status：enabled 恒返回；
// 已登录时附 username / image_root / data_dir（游客不泄露服务器路径）。
func (h *AuthHandler) Status(c *gin.Context) {
	authenticated, _ := c.Get(middleware.AuthenticatedKey)
	status := gin.H{"enabled": h.svc.Enabled(), "authenticated": authenticated == true}
	if authenticated == true {
		status["username"] = h.username
		status["image_root"] = h.imageRoot
		status["data_dir"] = h.dataDir
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

// setSessionCookie 写入/清除会话 Cookie（HttpOnly + SameSite=Lax + Path=/）。
func setSessionCookie(c *gin.Context, token string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
