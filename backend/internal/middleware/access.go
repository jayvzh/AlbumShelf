// Package middleware 提供 Gin 中间件。本文件实现 Sprint 7 访问控制（SPRINT7 §5.4）。
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/response"
	"imageshelf/backend/internal/auth"
)

// AuthenticatedKey gin.Context 中会话状态的 key。
const AuthenticatedKey = "authenticated"

// AuthService 中间件依赖的认证接口（由 internal/auth.Service 实现）。
type AuthService interface {
	Validate(token string) bool
	Enabled() bool
}

// ProtectedService 中间件依赖的私有目录判断接口（由 appsettings service 实现）。
type ProtectedService interface {
	IsProtectedPath(userPath string) bool
}

// Session 会话解析中间件：每请求读取 session_token cookie 并校验，
// 结果写入 c.Set(AuthenticatedKey, bool)。
func Session(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(auth.CookieName)
		authenticated := err == nil && authSvc.Validate(token)
		c.Set(AuthenticatedKey, authenticated)
		c.Next()
	}
}

// RequireAuth 管理端点守卫：auth enabled 且游客 → 401 UNAUTHORIZED；
// auth disabled 直通。
func RequireAuth(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authSvc.Enabled() {
			c.Next()
			return
		}
		if authenticated, _ := c.Get(AuthenticatedKey); authenticated != true {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				response.NewError(response.CodeUnauthorized, "Login required"))
			return
		}
		c.Next()
	}
}

// ProtectedAccess 内容端点守卫：auth enabled && 游客 && path 命中私有目录 → 401 UNAUTHORIZED；
// auth disabled 或已登录直通。
func ProtectedAccess(authSvc AuthService, protected ProtectedService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authSvc.Enabled() {
			c.Next()
			return
		}
		if authenticated, _ := c.Get(AuthenticatedKey); authenticated == true {
			c.Next()
			return
		}
		if protected.IsProtectedPath(c.Query("path")) {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				response.NewError(response.CodeUnauthorized, "Login required"))
			return
		}
		c.Next()
	}
}
