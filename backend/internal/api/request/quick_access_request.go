package request

// QuickAccessToggleRequest 快捷访问切换请求体（POST /quick-access/toggle）。
type QuickAccessToggleRequest struct {
	Path string `json:"path"`
}
