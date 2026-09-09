package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/request"
	"imageshelf/backend/internal/api/response"
	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/middleware"
	"imageshelf/backend/internal/model"
	"imageshelf/backend/internal/service"
	"imageshelf/backend/internal/sorting"
)

// AuthChecker 目录 Handler 依赖的认证探测接口（由 *auth.Service 实现）。
type AuthChecker interface{ Enabled() bool }

// FolderHandler 处理目录相关 API。
type FolderHandler struct {
	svc       *service.FolderService
	authSvc   AuthChecker
	protected *service.AppSettingsService
}

// NewFolderHandler 构造目录 Handler。authSvc/protected 用于游客私有目录过滤（Sprint 7）。
func NewFolderHandler(svc *service.FolderService, authSvc AuthChecker, protected *service.AppSettingsService) *FolderHandler {
	return &FolderHandler{svc: svc, authSvc: authSvc, protected: protected}
}

// List 处理 GET /api/v1/folders，返回目录内子目录与图片列表。
// auth enabled 且游客时剔除私有子树条目（SPRINT7 §5.5）。
func (h *FolderHandler) List(c *gin.Context) {
	req := request.NewFolderListRequest(c)

	// regex_rules 仅做结构与取值解析，格式非法直接拒绝（INVALID_REGEX 权威校验在后端）
	rules, err := model.ParseSortRules(req.RegexRules)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidRegex, "Invalid regex rules"))
		return
	}
	opts := model.SortOptions{Mode: req.Sort, Direction: req.Direction, Regex: req.Regex, Rules: rules}

	guard := &guestFilterGuard{
		active:    h.authSvc.Enabled() && !isAuthenticated(c),
		protected: h.protected,
	}
	result, err := h.svc.List(req.Path, opts, guard)
	if err != nil {
		writeFolderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response.NewFolderData(result.Path, result.Folders, result.Images)})
}

// guestFilterGuard service.GuestGuard 的请求级实现：仅游客且 auth enabled 时激活。
type guestFilterGuard struct {
	active    bool
	protected *service.AppSettingsService
}

func (g *guestFilterGuard) FilterActive() bool { return g.active }

func (g *guestFilterGuard) IsProtectedRelPath(relPath string) bool {
	return g.protected.IsProtectedRelPath(relPath)
}

// isAuthenticated 读取会话解析中间件写入的登录标志。
func isAuthenticated(c *gin.Context) bool {
	v, _ := c.Get(middleware.AuthenticatedKey)
	return v == true
}

// PreviewSort 处理 POST /api/v1/sort/preview，返回 Regex 排序预览（API.md §3.7）。
func (h *FolderHandler) PreviewSort(c *gin.Context) {
	var req request.SortPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid request body"))
		return
	}

	preview, err := h.svc.PreviewSort(req.Files, model.SortOptions{
		Mode:  req.Mode,
		Regex: req.Regex,
		Rules: req.Rules,
	})
	if err != nil {
		writeFolderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": preview})
}

// writeFolderError 把 filesystem/sorting sentinel 错误映射为统一错误响应。
func writeFolderError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, filesystem.ErrInvalidPath):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid path"))
	case errors.Is(err, filesystem.ErrNotFound):
		c.JSON(http.StatusNotFound, response.NewError(response.CodeFolderNotFound, "Folder not found"))
	case errors.Is(err, sorting.ErrInvalidRegex):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidRegex, "Invalid regex"))
	default:
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
	}
}
