package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health 处理 GET /api/v1/health，返回健康状态。
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"status": "ok"},
	})
}
