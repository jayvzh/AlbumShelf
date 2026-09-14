// Package app 负责应用全部依赖的组装与生命周期管理。
package app

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"albumshelf/backend/internal/api"
	"albumshelf/backend/internal/api/handler"
	"albumshelf/backend/internal/auth"
	"albumshelf/backend/internal/config"
	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/repository"
	"albumshelf/backend/internal/service"
	"albumshelf/backend/internal/thumbnail"
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
	thumbnailSvc := service.NewThumbnailService(fs, cfg.DataDir, cacheRepo,
		thumbnail.NewScheduler(cfg.ThumbConcurrency))

	// 并发配置自检：缩略图调度器就绪后输出生效值（与 libvips 实际取值同源：
	// thumbnail.EffectiveVipsConcurrency——env 显式覆盖优先，否则按
	// 可用核数 ÷ 调度器并发放量配平）。乘积超过可用核数时提示超订阅。
	procs := runtime.GOMAXPROCS(0)
	vipsWorkers := thumbnail.EffectiveVipsConcurrency()
	log.Printf("缩略图并发配置: THUMB_CONCURRENCY=%d, VIPS_CONCURRENCY=%d, 可用核数=%d",
		cfg.ThumbConcurrency, vipsWorkers, procs)
	if cfg.ThumbConcurrency*vipsWorkers > procs {
		log.Printf("[WARN] 缩略图并发超订阅: THUMB_CONCURRENCY(%d) × VIPS_CONCURRENCY(%d) = %d > 可用核数 %d，建议两者乘积约等于可用核数",
			cfg.ThumbConcurrency, vipsWorkers, cfg.ThumbConcurrency*vipsWorkers, procs)
	}

	// 后台预热器：空闲低优先级补齐全库 thumbs（成功后级联趁热补 preview），
	// 首次进入任何目录即命中缓存；随进程生命周期运行，无需优雅退出。
	if cfg.ThumbWarmer {
		go service.NewThumbWarmer(fs, thumbnailSvc, 30*time.Minute).Run(context.Background())
	}

	imageSvc := service.NewImageService(fs, thumbnailSvc)
	imageH := handler.NewImageHandler(imageSvc)
	thumbnailH := handler.NewThumbnailHandler(thumbnailSvc)

	cacheSvc := service.NewCacheService(cfg.DataDir, cfg.ImageRoot, cacheRepo)
	cacheH := handler.NewCacheHandler(cacheSvc)

	favoriteRepo := repository.NewFavoriteRepository(db)
	favoriteSvc := service.NewFavoriteService(fs, favoriteRepo)
	favoriteH := handler.NewFavoriteHandler(favoriteSvc)

	quickAccessRepo := repository.NewQuickAccessRepository(db)
	quickAccessSvc := service.NewQuickAccessService(fs, quickAccessRepo)
	quickAccessH := handler.NewQuickAccessHandler(quickAccessSvc)

	configSvc := service.NewConfigService(fs, folderRepo, settingsRepo, appSettingsSvc, favoriteRepo, quickAccessRepo)
	configH := handler.NewConfigHandler(configSvc)

	authH := handler.NewAuthHandler(authSvc, cfg.AuthUsername, cfg.ImageRoot, cfg.DataDir)

	// 初始化检测（Sprint 8）：只读探测 IMAGE_ROOT，公开端点供前端引导页使用
	setupH := handler.NewSetupHandler(service.NewSetupService(cfg.ImageRoot, cfg.AuthPassword))

	a := &App{
		cfg: cfg,
		router: api.NewRouter(folderH, imageH, thumbnailH, settingsH,
			authH, appSettingsH, cacheH, configH, favoriteH, quickAccessH, setupH, authSvc, appSettingsSvc),
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
		// 先匹配 dist 根目录下真实存在的静态文件（如 public 资源 favicon.svg），
		// 命中则按文件返回（Content-Type 由扩展名推断）；未命中再回 index.html
		// 交给前端路由处理。filepath.Join 会清洗路径，避免目录穿越。
		if p := strings.TrimPrefix(filepath.Clean(c.Request.URL.Path), "/"); p != "" {
			if fp := filepath.Join(distDir, p); isRegularFile(fp) {
				c.File(fp)
				return
			}
		}
		c.File(filepath.Join(distDir, "index.html"))
	})
}

// isRegularFile 判断路径是否为存在的普通文件（非目录），用于区分静态资源与前端路由。
func isRegularFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
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
