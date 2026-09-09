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

// ThumbnailHandler 处理缩略图 API。
type ThumbnailHandler struct {
	svc *service.ThumbnailService
}

// NewThumbnailHandler 构造缩略图 Handler。
func NewThumbnailHandler(svc *service.ThumbnailService) *ThumbnailHandler {
	return &ThumbnailHandler{svc: svc}
}

// Get 处理 GET /api/v1/thumbnail，流式返回缩略图 JPEG。
func (h *ThumbnailHandler) Get(c *gin.Context) {
	req := request.NewThumbnailRequest(c)
	if req.Path == "" {
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid path"))
		return
	}

	info, file, err := h.svc.GetThumb(req.Path, req.Width)
	if err != nil {
		switch {
		case errors.Is(err, filesystem.ErrInvalidPath),
			errors.Is(err, filesystem.ErrNotFound),
			errors.Is(err, filesystem.ErrUnsupported):
			// 路径类错误沿用图片查看的统一映射（400/404/415）
			writeImageError(c, err)
		default:
			// 其余为生成阶段失败（解码失败、写缓存失败等）
			c.JSON(http.StatusInternalServerError, response.NewError(response.CodeThumbnailFailed, "Thumbnail generation failed"))
		}
		return
	}
	defer file.Close()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	// ETag 用源文件 mtime/size（与 /image 一致的强校验规范），原图更新后浏览器缓存自然失效
	c.Header("ETag", service.ETag(info))
	c.Header("X-Image-Variant", model.VariantThumb)

	// 传缩略文件名辅助 ServeContent 的 Range 语义
	http.ServeContent(c.Writer, c.Request, "thumb_"+info.Name+".jpg", info.ModifiedAt, file)
}
