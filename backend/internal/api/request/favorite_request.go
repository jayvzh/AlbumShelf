package request

// FavoriteToggleRequest 收藏切换请求体（POST /favorites/toggle）。
type FavoriteToggleRequest struct {
	Path string `json:"path"`
}
