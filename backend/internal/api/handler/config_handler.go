package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/request"
	"imageshelf/backend/internal/api/response"
	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/model"
	"imageshelf/backend/internal/service"
)

// ConfigHandler 处理配置导入导出 API（SPRINT7 §5.7）。
type ConfigHandler struct {
	svc *service.ConfigService
}

// NewConfigHandler 构造配置 Handler。
func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

// Export 处理 GET /api/v1/config/export：JSON 附件下载。
func (h *ConfigHandler) Export(c *gin.Context) {
	payload, err := h.svc.Export()
	if err != nil {
		writeConfigError(c, err)
		return
	}
	filename := fmt.Sprintf("albumshelf-config-%s.json", time.Now().Format("20060102"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.JSON(http.StatusOK, payload)
}

// Import 处理 POST /api/v1/config/import：JSON 解析失败或 version 不符 → 400 CONFIG_INVALID。
func (h *ConfigHandler) Import(c *gin.Context) {
	var req request.ConfigImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeConfigInvalid, "Invalid config"))
		return
	}

	payload := model.ConfigPayload{
		Version:          req.Version,
		ExportedAt:       req.ExportedAt,
		ProtectedFolders: req.ProtectedFolders,
	}
	for _, fav := range req.Favorites {
		payload.Favorites = append(payload.Favorites, model.ConfigFavoriteEntry{
			Path:      fav.Path,
			CreatedAt: fav.CreatedAt,
		})
	}
	for _, f := range req.FolderSettings {
		payload.FolderSettings = append(payload.FolderSettings, model.ConfigFolderEntry{
			Path:            f.Path,
			SortMode:        configStringPtr(f.SortMode),
			SortDirection:   configStringPtr(f.SortDirection),
			RegexPattern:    configStringPtr(f.RegexPattern),
			RegexConfig:     configStringPtr(f.RegexConfig),
			PageMode:        f.PageMode,
			ReadOrder:       f.ReadOrder,
			WideRatio:       f.WideRatio,
			SingleFirstPage: f.SingleFirstPage,
			SingleLastPage:  f.SingleLastPage,
		})
	}

	result, err := h.svc.Import(payload)
	if err != nil {
		writeConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// writeConfigError 错误映射：坏配置 → 400 CONFIG_INVALID；引用了非法/不存在路径 → 400 INVALID_PATH。
func writeConfigError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrConfigInvalid):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeConfigInvalid, "Invalid config"))
	case errors.Is(err, filesystem.ErrInvalidPath), errors.Is(err, filesystem.ErrNotFound):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid path"))
	case errors.Is(err, service.ErrDatabase):
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeDatabaseError, "Database error"))
	default:
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
	}
}

// configStringPtr 空串映射 nil（wire format null = 未保存），非空取地址。
func configStringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
