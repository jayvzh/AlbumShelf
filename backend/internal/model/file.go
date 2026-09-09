package model

import "time"

// File 文件基础元信息。
type File struct {
	Name string
	// Path 相对 IMAGE_ROOT 的绝对风格路径，如 /Comics/001.jpg。
	Path       string
	Extension  string // 含点，如 .jpg
	Size       int64
	ModifiedAt time.Time
	// CreatedAt 排序用；取不到时回退 ModifiedAt 填充。
	CreatedAt time.Time
}
