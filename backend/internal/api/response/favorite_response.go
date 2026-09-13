package response

import (
	"albumshelf/backend/internal/model"
)

// FavoriteData GET /api/v1/favorites 成功响应（API.md §3.x）。
// images 与 GET /folders 的 images 同构，按收藏时间倒序。
type FavoriteData struct {
	Images []ImageFile `json:"images"`
}

// NewFavoriteData 把收藏图片列表转换为响应 DTO。
func NewFavoriteData(images []model.Image) FavoriteData {
	return FavoriteData{Images: NewImageFiles(images)}
}
