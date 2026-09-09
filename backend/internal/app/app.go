// Package app 负责应用全部依赖的组装与生命周期管理。
package app

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api"
	"imageshelf/backend/internal/api/handler"
	"imageshelf/backend/internal/auth"
	"imageshelf/backend/internal/config"
	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/repository"
	"imageshelf/backend/internal/service"
)

// App 持有应用级依赖。
type App struct {
	cfg    *config.Config
	router *gin.Engine
	db     *sql.DB
}

// New 完成依赖组装并构建路由。
//
// main.go 只负责 config.Load() → app.New() → app.Run()，
// 所有初始化与组装逻辑集中在本文件。
func New(cfg *config.Config) *App {
	// 依赖组装：Config → Filesystem/SQLite → Service → Handler → Router
	fs := filesystem.NewLocalFilesystem(cfg.ImageRoot)

	db, err := repository.Open(cfg.DataDir)
	if err != nil {
		// 数据库不可用属致命错误：缩略图缓存索引无法工作，直接终止启动
		log.Fatalf("初始化 SQLite 失败: %v", err)
	}

	folderRepo := repository.NewFolderRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// 认证服务：凭证来自 env（SPRINT7 §5.1），AUTH_PASSWORD 未设置时全部新语义关闭
	sessionRepo := repository.NewSessionRepository(db)
	authSvc := auth.New(cfg.AuthUsername, cfg.AuthPassword, time.Duration(cfg.SessionMaxAge)*time.Second, sessionRepo)

	folderSvc := service.NewFolderService(fs, settingsRepo)

	appSettingsSvc := service.NewAppSettingsService(fs, repository.NewProtectedFolderRepository(db))
	folderH := handler.NewFolderHandler(folderSvc, authSvc, appSettingsSvc)
	appSettingsH := handler.NewAppSettingsHandler(appSettingsSvc)

	settingsSvc := service.NewSettingsService(fs, folderRepo, settingsRepo)
	settingsH := handler.NewSettingsHandler(settingsSvc)

	cacheRepo := repository.NewImageCacheRepository(db)
	thumbnailSvc := service.NewThumbnailService(fs, cfg.DataDir, cacheRepo)
	imageSvc := service.NewImageService(fs, thumbnailSvc)
	imageH := handler.NewImageHandler(imageSvc)
	thumbnailH := handler.NewThumbnailHandler(thumbnailSvc)

	cacheSvc := service.NewCacheService(cfg.DataDir, cfg.ImageRoot, cacheRepo)
	cacheH := handler.NewCacheHandler(cacheSvc)

	favoriteRepo := repository.NewFavoriteRepository(db)
	favoriteSvc := service.NewFavoriteService(fs, favoriteRepo)
	favoriteH := handler.NewFavoriteHandler(favoriteSvc)

	configSvc := service.NewConfigService(fs, folderRepo, settingsRepo, appSettingsSvc, favoriteRepo)
	configH := handler.NewConfigHandler(configSvc)

	authH := handler.NewAuthHandler(authSvc, cfg.AuthUsername, cfg.ImageRoot, cfg.DataDir)

	// 初始化检测（Sprint 8）：只读探测 IMAGE_ROOT，公开端点供前端引导页使用
	setupH := handler.NewSetupHandler(service.NewSetupService(cfg.ImageRoot, cfg.AuthPassword))

	a := &App{
		cfg: cfg,
		router: api.NewRouter(folderH, imageH, thumbnailH, settingsH,
			authH, appSettingsH, cacheH, configH, favoriteH, setupH, authSvc, appSettingsSvc),
		db: db,
	}
	a.registerStatic()
	return a
}

// registerStatic 当 ./dist（前端构建产物）存在时由后端托管静态文件，
// 实现 MVP 单容器部署；本地开发模式下前端由 Vite dev server 提供，不启用。
func (a *App) registerStatic() {
	const distDir = "./dist"

	info, err := os.Stat(distDir)
	if err != nil || !info.IsDir() {
		return
	}

	a.router.Static("/assets", filepath.Join(distDir, "assets"))
	a.router.StaticFile("/", filepath.Join(distDir, "index.html"))

	// SPA history fallback（Sprint 8）：非 API 路径未命中路由时回 index.html，
	// 使 /setup、/settings 等深层路径可直接访问与刷新；API 路径维持 404。
	a.router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			http.NotFound(c.Writer, c.Request)
			return
		}
		c.File(filepath.Join(distDir, "index.html"))
	})
}

// Run 启动 HTTP 服务并阻塞，直至退出。
func (a *App) Run() error {
	// 进程退出路径关闭数据库连接
	defer func() {
		if err := repository.Close(a.db); err != nil {
			log.Printf("关闭数据库失败: %v", err)
		}
	}()
	return a.router.Run(a.cfg.Addr())
}
