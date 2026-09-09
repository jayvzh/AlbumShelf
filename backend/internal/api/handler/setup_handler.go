package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/service"
)

// SetupHandler 处理初始化检测 API（SPRINT8 §5.3）。
type SetupHandler struct {
	svc *service.SetupService
}

// NewSetupHandler 构造初始化检测 Handler。
func NewSetupHandler(svc *service.SetupService) *SetupHandler {
	return &SetupHandler{svc: svc}
}

// Status 处理 GET /api/v1/setup/status：公开端点（无需登录），
// 只读检测 IMAGE_ROOT 可用性，不写任何状态。
func (h *SetupHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": h.svc.Status(c.Request.Context())})
}
