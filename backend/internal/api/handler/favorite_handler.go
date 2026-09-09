package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/request"
	"imageshelf/backend/internal/api/response"
	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/service"
)

// FavoriteHandler 处理收藏 API（PRD F010 Phase 2）。
// 两个端点均为个人数据端点，auth enabled 时挂 RequireAuth（游客完全隐藏）。
type FavoriteHandler struct {
	svc *service.FavoriteService
}

// NewFavoriteHandler 构造收藏 Handler。
func NewFavoriteHandler(svc *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{svc: svc}
}

// List 处理 GET /api/v1/favorites，返回收藏图片列表（收藏时间倒序）。
// 已删除/移动文件的失效收藏由 service 惰性清理，不出现在响应中。
func (h *FavoriteHandler) List(c *gin.Context) {
	images, err := h.svc.List()
	if err != nil {
		writeFavoriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": response.NewFavoriteData(images)})
}

// Toggle 处理 POST /api/v1/favorites/toggle，切换收藏状态并返回切换后的状态。
func (h *FavoriteHandler) Toggle(c *gin.Context) {
	var req request.FavoriteToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid request body"))
		return
	}

	favorited, err := h.svc.Toggle(req.Path)
	if err != nil {
		writeFavoriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"path": req.Path, "favorited": favorited}})
}

// writeFavoriteError 收藏错误映射：路径非法 → 400 INVALID_PATH；图片不存在 →
// 404 FILE_NOT_FOUND；格式不支持 → 400 UNSUPPORTED_FORMAT；数据库失败 → 500。
func writeFavoriteError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, filesystem.ErrInvalidPath):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid path"))
	case errors.Is(err, filesystem.ErrNotFound):
		c.JSON(http.StatusNotFound, response.NewError(response.CodeFileNotFound, "File not found"))
	case errors.Is(err, filesystem.ErrUnsupported):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeUnsupported, "Unsupported format"))
	case errors.Is(err, service.ErrDatabase):
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeDatabaseError, "Database error"))
	default:
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
	}
}
