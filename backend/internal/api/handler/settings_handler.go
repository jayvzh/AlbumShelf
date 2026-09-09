package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/request"
	"imageshelf/backend/internal/api/response"
	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/model"
	"imageshelf/backend/internal/service"
)

// SettingsHandler 处理文件夹设置 API（API.md §3.6）。
type SettingsHandler struct {
	svc *service.SettingsService
}

// NewSettingsHandler 构造设置 Handler。
func NewSettingsHandler(svc *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

// Get 处理 GET /api/v1/folder/settings，读取目录已保存设置（未保存字段序列化为 null）。
func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.svc.Get(c.Query("path"))
	if err != nil {
		writeSettingsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settingsData(settings)})
}

// Save 处理 PUT /api/v1/folder/settings，保存目录设置并返回规范化后的结果。
func (h *SettingsHandler) Save(c *gin.Context) {
	var req request.FolderSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid request body"))
		return
	}

	settings, err := h.svc.Save(req.Path, model.FolderSettings{
		Path:            req.Path,
		SortMode:        req.SortMode,
		SortDirection:   req.SortDirection,
		RegexPattern:    req.RegexPattern,
		RegexConfig:     req.RegexConfig,
		PageMode:        nilOrStringPtr(req.PageMode),
		ReadOrder:       nilOrStringPtr(req.ReadOrder),
		WideRatio:       req.WideRatio,
		SingleFirstPage: req.SingleFirstPage,
		SingleLastPage:  req.SingleLastPage,
	})
	if err != nil {
		writeSettingsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settingsData(settings)})
}

// settingsData 构造响应体：未保存字段（空串/nil 指针）序列化为 null（API.md §3.6）。
func settingsData(s *model.FolderSettings) gin.H {
	return gin.H{
		"path":               s.Path,
		"sort_mode":          nilOrString(s.SortMode),
		"sort_direction":     nilOrString(s.SortDirection),
		"regex_pattern":      nilOrString(s.RegexPattern),
		"regex_config":       nilOrString(s.RegexConfig),
		"page_mode":          nilOrString(s.PageMode),
		"read_order":         nilOrString(s.ReadOrder),
		"wide_ratio":         nilOrFloat64(s.WideRatio),
		"single_first_page":  nilOrBool(s.SingleFirstPage),
		"single_last_page":   nilOrBool(s.SingleLastPage),
	}
}

// nilOrString 空串映射为 null（未保存），其余原样返回。
func nilOrString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// nilOrStringPtr 请求指针解引用；nil 映射为空串（未提供语义），空串再由 nilOrString 转 null。
func nilOrStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// nilOrFloat64 nil 指针映射为 null（未保存），其余取值。
func nilOrFloat64(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

// nilOrBool nil 指针映射为 null（未保存），其余取值。
func nilOrBool(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

// writeSettingsError 把 filesystem/service sentinel 错误映射为统一错误响应。
func writeSettingsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, filesystem.ErrInvalidPath):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid path"))
	case errors.Is(err, filesystem.ErrNotFound):
		c.JSON(http.StatusNotFound, response.NewError(response.CodeFolderNotFound, "Folder not found"))
	case errors.Is(err, service.ErrDatabase):
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeDatabaseError, "Database error"))
	default:
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
	}
}
