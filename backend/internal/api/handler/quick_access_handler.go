package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"albumshelf/backend/internal/api/request"
	"albumshelf/backend/internal/api/response"
	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/service"
)

// QuickAccessHandler 处理快捷访问 API（目录钉住）。
// 两个端点均为个人数据端点，auth enabled 时挂 RequireAuth（游客完全隐藏）。
type QuickAccessHandler struct {
	svc *service.QuickAccessService
}

// NewQuickAccessHandler 构造快捷访问 Handler。
func NewQuickAccessHandler(svc *service.QuickAccessService) *QuickAccessHandler {
	return &QuickAccessHandler{svc: svc}
}

// List 处理 GET /api/v1/quick-access，返回固定目录路径列表（固定时间倒序）。
// 已删除/移动目录的失效条目由 service 惰性清理，不出现在响应中。
func (h *QuickAccessHandler) List(c *gin.Context) {
	paths, err := h.svc.List()
	if err != nil {
		writeQuickAccessError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"paths": paths}})
}

// Toggle 处理 POST /api/v1/quick-access/toggle，切换指定目录的固定状态并返回切换后的状态。
func (h *QuickAccessHandler) Toggle(c *gin.Context) {
	var req request.QuickAccessToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid request body"))
		return
	}

	pinned, err := h.svc.Toggle(req.Path)
	if err != nil {
		writeQuickAccessError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"path": req.Path, "pinned": pinned}})
}

// writeQuickAccessError 快捷访问错误映射：路径非法/非目录 → 400 INVALID_PATH；
// 目录不存在 → 404 FILE_NOT_FOUND；数据库失败 → 500。
func writeQuickAccessError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, filesystem.ErrInvalidPath):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid path"))
	case errors.Is(err, filesystem.ErrNotFound):
		c.JSON(http.StatusNotFound, response.NewError(response.CodeFileNotFound, "Folder not found"))
	case errors.Is(err, service.ErrDatabase):
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeDatabaseError, "Database error"))
	default:
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
	}
}
