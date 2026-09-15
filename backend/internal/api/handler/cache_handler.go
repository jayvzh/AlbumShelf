package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"albumshelf/backend/internal/api/response"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/service"
)

// CacheHandler 处理缓存管理 API（SPRINT7 §5.6 + 预热管理）。
type CacheHandler struct {
	svc    *service.CacheService
	warmer *service.CacheWarmer
}

// NewCacheHandler 构造缓存管理 Handler。
func NewCacheHandler(svc *service.CacheService, warmer *service.CacheWarmer) *CacheHandler {
	return &CacheHandler{svc: svc, warmer: warmer}
}

// Stats 处理 GET /api/v1/cache/stats，返回 thumb/preview 磁盘统计。
func (h *CacheHandler) Stats(c *gin.Context) {
	stats, err := h.svc.Stats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// CleanupOrphans 处理 POST /api/v1/cache/cleanup/orphan，清理两类孤儿缓存。
func (h *CacheHandler) CleanupOrphans(c *gin.Context) {
	h.cleanup(c, h.svc.CleanupOrphans)
}

// CleanupAll 处理 POST /api/v1/cache/cleanup/all，清空全部缓存。
// 清理成功后 Kick 预热器立即按当前级别开始重建（闲时推进）。
func (h *CacheHandler) CleanupAll(c *gin.Context) {
	h.cleanup(c, func() (*service.CacheCleanupResult, error) {
		result, err := h.svc.CleanupAll()
		if err == nil {
			h.warmer.Kick()
		}
		return result, err
	})
}

// CleanupVariant 处理 POST /api/v1/cache/cleanup/variant，清空单个变体
//（thumb/preview）的全部缓存文件与索引行。清理成功后 Kick 预热器重建。
func (h *CacheHandler) CleanupVariant(c *gin.Context) {
	var req struct {
		Variant string `json:"variant"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Variant != model.VariantThumb && req.Variant != model.VariantPreview) {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidRequest, "Invalid variant"))
		return
	}
	h.cleanup(c, func() (*service.CacheCleanupResult, error) {
		result, err := h.svc.CleanupVariant(req.Variant)
		if err == nil {
			h.warmer.Kick()
		}
		return result, err
	})
}

// WarmStatus 处理 GET /api/v1/cache/warm/status，返回预热级别与进度快照。
func (h *CacheHandler) WarmStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": h.warmer.Status()})
}

// UpdateWarmSettings 处理 PUT /api/v1/cache/warm/settings，
// 更新预热级别（持久化并即时生效，触发按新级别重新扫描）。
func (h *CacheHandler) UpdateWarmSettings(c *gin.Context) {
	var req struct {
		Level string `json:"level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidRequest, "Invalid level"))
		return
	}
	level, ok := model.ParseWarmLevel(req.Level)
	if !ok {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidRequest, "Invalid level"))
		return
	}
	if err := h.warmer.SetLevel(c.Request.Context(), level); err != nil {
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.warmer.Status()})
}

// cleanup 执行清理动作并写响应（各清理端点共用）。
func (h *CacheHandler) cleanup(c *gin.Context, fn func() (*service.CacheCleanupResult, error)) {
	result, err := fn()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
