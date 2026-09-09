package model

// Folder 目录条目。
type Folder struct {
	Name string
	// Path 相对 IMAGE_ROOT 的绝对风格路径，如 /Comics/Chapter01。
	Path string
}
