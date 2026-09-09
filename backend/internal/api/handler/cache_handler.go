package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/response"
	"imageshelf/backend/internal/service"
)

// CacheHandler 处理缓存管理 API（SPRINT7 §5.6）。
type CacheHandler struct {
	svc *service.CacheService
}

// NewCacheHandler 构造缓存管理 Handler。
func NewCacheHandler(svc *service.CacheService) *CacheHandler {
	return &CacheHandler{svc: svc}
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
func (h *CacheHandler) CleanupAll(c *gin.Context) {
	h.cleanup(c, h.svc.CleanupAll)
}

// cleanup 执行清理动作并写响应（两类清理共用）。
func (h *CacheHandler) cleanup(c *gin.Context, fn func() (*service.CacheCleanupResult, error)) {
	result, err := fn()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
