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

// AppSettingsHandler 处理私有目录 API（SPRINT7 §5.5）。
type AppSettingsHandler struct {
	svc *service.AppSettingsService
}

// NewAppSettingsHandler 构造应用设置 Handler。
func NewAppSettingsHandler(svc *service.AppSettingsService) *AppSettingsHandler {
	return &AppSettingsHandler{svc: svc}
}

// List 处理 GET /api/v1/protected-folders，返回全部私有目录。
func (h *AppSettingsHandler) List(c *gin.Context) {
	paths, err := h.svc.ListPaths()
	if err != nil {
		writeAppSettingsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"paths": paths}})
}

// Replace 处理 PUT /api/v1/protected-folders（全量替换语义），返回规范化后的列表。
func (h *AppSettingsHandler) Replace(c *gin.Context) {
	var req request.ProtectedFoldersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid request body"))
		return
	}

	paths, err := h.svc.ReplacePaths(req.Paths)
	if err != nil {
		writeAppSettingsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"paths": paths}})
}

// writeAppSettingsError 错误映射：非法/不存在路径 → 400 INVALID_PATH（§5.5）。
func writeAppSettingsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, filesystem.ErrInvalidPath), errors.Is(err, filesystem.ErrNotFound):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid path"))
	case errors.Is(err, service.ErrDatabase):
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeDatabaseError, "Database error"))
	default:
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
	}
}
