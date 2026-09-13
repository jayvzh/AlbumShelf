package request

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"albumshelf/backend/internal/model"
)

// ImageGetRequest 图片读取请求参数。variant 一律回退原图（Sprint 2）。
type ImageGetRequest struct {
	Path    string
	Variant string
}

// NewImageGetRequest 从 query 参数构造请求对象。
func NewImageGetRequest(c *gin.Context) ImageGetRequest {
	return ImageGetRequest{Path: c.Query("path"), Variant: c.Query("variant")}
}

// ImageInfoRequest 图片元信息请求参数。
type ImageInfoRequest struct {
	Path string
}

// NewImageInfoRequest 从 query 参数构造请求对象。
func NewImageInfoRequest(c *gin.Context) ImageInfoRequest {
	return ImageInfoRequest{Path: c.Query("path")}
}

// ThumbnailRequest 缩略图请求参数。
type ThumbnailRequest struct {
	Path  string
	Width int
}

// NewThumbnailRequest 从 query 参数构造请求对象；
// width 缺省、解析失败或不在合法桶（200/300/500）内时一律按 300 处理。
func NewThumbnailRequest(c *gin.Context) ThumbnailRequest {
	width := 300
	if raw := c.Query("width"); raw != "" {
		if w, err := strconv.Atoi(raw); err == nil && model.IsValidThumbBucket(w) {
			width = w
		}
	}
	return ThumbnailRequest{Path: c.Query("path"), Width: width}
}
