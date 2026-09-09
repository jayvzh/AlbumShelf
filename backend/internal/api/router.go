package api

import (
	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/handler"
	"imageshelf/backend/internal/middleware"
)

// NewRouter 构建 API 路由。只负责注册路由，不含业务逻辑。
// Sprint 7：会话解析全局挂载；内容端点挂 ProtectedAccess，管理端点挂 RequireAuth（SPRINT7 §5.8）。
func NewRouter(
	folderH *handler.FolderHandler,
	imageH *handler.ImageHandler,
	thumbnailH *handler.ThumbnailHandler,
	settingsH *handler.SettingsHandler,
	authH *handler.AuthHandler,
	appSettingsH *handler.AppSettingsHandler,
	cacheH *handler.CacheHandler,
	configH *handler.ConfigHandler,
	favoriteH *handler.FavoriteHandler,
	setupH *handler.SetupHandler,
	authSvc middleware.AuthService,
	protectedSvc middleware.ProtectedService,
) *gin.Engine {
	r := gin.Default()

	// 会话解析：每请求读 session_token cookie → Validate → c.Set(authenticated)
	r.Use(middleware.Session(authSvc))

	// 内容端点游客访问控制（auth enabled && 游客 && 命中私有目录 → 401）
	protectedAccess := middleware.ProtectedAccess(authSvc, protectedSvc)
	// 管理端点登录要求（auth enabled && 游客 → 401）
	requireAuth := middleware.RequireAuth(authSvc)

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", handler.Health)

		// 初始化检测（Sprint 8）：公开端点（不挂 RequireAuth / ProtectedAccess），
		// 未登录 + 配置有误时仍可访问，否则引导页死锁
		v1.GET("/setup/status", setupH.Status)

		// 认证（Sprint 7）
		v1.POST("/auth/login", authH.Login)
		v1.POST("/auth/logout", authH.Logout)
		v1.GET("/auth/status", authH.Status)

		// 私有目录（Sprint 7；PUT 为管理端点需登录）
		v1.GET("/protected-folders", appSettingsH.List)
		v1.PUT("/protected-folders", requireAuth, appSettingsH.Replace)

		// 缓存管理（Sprint 7；清理为管理端点需登录）
		v1.GET("/cache/stats", cacheH.Stats)
		v1.POST("/cache/cleanup/orphan", requireAuth, cacheH.CleanupOrphans)
		v1.POST("/cache/cleanup/all", requireAuth, cacheH.CleanupAll)

		// 配置导入导出（Sprint 7；均为管理端点需登录）
		v1.GET("/config/export", requireAuth, configH.Export)
		v1.POST("/config/import", requireAuth, configH.Import)

		// 收藏（Phase 2：个人数据端点需登录，游客完全隐藏）
		v1.GET("/favorites", requireAuth, favoriteH.List)
		v1.POST("/favorites/toggle", requireAuth, favoriteH.Toggle)

		// 目录内容（Sprint 1：子目录 + 图片列表；Sprint 7：游客列表过滤私有条目）
		v1.GET("/folders", protectedAccess, folderH.List)
		// 图片查看（Sprint 2：原图流式输出 + 元信息；Sprint 3：preview 真实生成）
		v1.GET("/image", protectedAccess, imageH.Get)
		v1.GET("/image/info", protectedAccess, imageH.Info)
		// 缩略图（Sprint 3：libvips 生成 + 磁盘缓存）
		v1.GET("/thumbnail", protectedAccess, thumbnailH.Get)
		// 文件夹设置（Sprint 4/5：排序设置持久化含 regex 字段；view/page 随 Sprint 6 扩展）
		v1.GET("/folder/settings", protectedAccess, settingsH.Get)
		v1.PUT("/folder/settings", protectedAccess, settingsH.Save)
		// Regex 排序预览（Sprint 5：无状态纯函数预览，无 path 语义不挂访问控制）
		v1.POST("/sort/preview", folderH.PreviewSort)
	}

	return r
}
