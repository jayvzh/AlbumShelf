package model

import "time"

// QuickAccess 对应 quick_access 表一行（快捷访问：钉住的目录）。
// Path 为 IMAGE_ROOT 相对绝对风格路径，唯一约束去重。
type QuickAccess struct {
	Path      string
	CreatedAt time.Time
}
