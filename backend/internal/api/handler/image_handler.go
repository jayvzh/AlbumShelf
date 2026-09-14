package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"albumshelf/backend/internal/api/request"
	"albumshelf/backend/internal/api/response"
	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/service"
	"albumshelf/backend/internal/thumbnail"
)

// ImageHandler 处理图片查看相关 API。
type ImageHandler struct {
	svc *service.ImageService
}

// NewImageHandler 构造图片 Handler。
func NewImageHandler(svc *service.ImageService) *ImageHandler {
	return &ImageHandler{svc: svc}
}

// Get 处理 GET /api/v1/image，流式返回图片二进制。
// variant=preview 走预览生成链路（service 内失败自动回退原图）；
// 请求头 X-Load-Priority: low（前端空闲预热）将生成任务降为低优先级——
// 同源自定义头不改变 URL，浏览器 HTTP 缓存键不受影响；
// X-Image-Variant 如实标注实际返回的变体（preview 或 original 回退）。
func (h *ImageHandler) Get(c *gin.Context) {
	req := request.NewImageGetRequest(c)

	prio := thumbnail.PriorityHigh
	if c.GetHeader("X-Load-Priority") == "low" {
		prio = thumbnail.PriorityLow
	}

	info, file, actualVariant, err := h.svc.Get(c.Request.Context(), req.Path, req.Variant, prio)
	if err != nil {
		writeImageError(c, err)
		return
	}
	defer file.Close()

	// 预览产物统一 JPEG 输出；原图（含 preview 回退情形）保持按扩展名映射
	contentType := service.ContentType(info.Extension)
	if actualVariant == model.VariantPreview {
		contentType = "image/jpeg"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("ETag", service.ETag(info))
	c.Header("X-Image-Variant", actualVariant)

	// ServeContent 自动支持 Range 请求与 If-None-Match → 304（读取已设置的 ETag）
	http.ServeContent(c.Writer, c.Request, info.Name, info.ModifiedAt, file)
}

// Info 处理 GET /api/v1/image/info，返回图片元信息。
func (h *ImageHandler) Info(c *gin.Context) {
	req := request.NewImageInfoRequest(c)

	info, err := h.svc.GetInfo(req.Path)
	if err != nil {
		writeImageError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response.NewImageData(info)})
}

// writeImageError 把 filesystem sentinel 错误映射为统一错误响应。
func writeImageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, filesystem.ErrInvalidPath):
		c.JSON(http.StatusBadRequest, response.NewError(response.CodeInvalidPath, "Invalid path"))
	case errors.Is(err, filesystem.ErrNotFound):
		c.JSON(http.StatusNotFound, response.NewError(response.CodeFileNotFound, "Image not found"))
	case errors.Is(err, filesystem.ErrUnsupported):
		c.JSON(http.StatusUnsupportedMediaType, response.NewError(response.CodeUnsupported, "Unsupported image format"))
	default:
		c.JSON(http.StatusInternalServerError, response.NewError(response.CodeInternalError, "Internal error"))
	}
}
