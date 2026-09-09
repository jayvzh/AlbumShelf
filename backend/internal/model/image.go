package model

// Image 在 File 基础上附加像素尺寸；解析失败时 Width/Height 为 nil（JSON null）。
type Image struct {
	File
	Width  *int
	Height *int
}
