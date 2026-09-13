package response

import (
	"time"

	"albumshelf/backend/internal/model"
)

// Resolution 像素尺寸对象；解析失败时 width/height 为 null。
type Resolution struct {
	Width  *int `json:"width"`
	Height *int `json:"height"`
}

// ImageData GET /api/v1/image/info 成功响应（API.md §3.4）。
type ImageData struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Resolution Resolution `json:"resolution"`
	Size       int64      `json:"size"`
	ModifiedAt string     `json:"modified_at"` // UTC RFC3339
}

// NewImageData 把领域模型转换为响应 DTO。
func NewImageData(img model.Image) ImageData {
	return ImageData{
		Name:       img.Name,
		Path:       img.Path,
		Resolution: Resolution{Width: img.Width, Height: img.Height},
		Size:       img.Size,
		ModifiedAt: img.ModifiedAt.UTC().Format(time.RFC3339),
	}
}
