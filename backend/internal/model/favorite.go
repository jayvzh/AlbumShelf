package model

import "time"

// Favorite 收藏记录（DATA_MODEL §2.3）；Path 为相对 IMAGE_ROOT 的绝对风格路径，
// 与 model.File.Path 同一形态，file_path 唯一约束保证去重。
type Favorite struct {
	Path      string
	CreatedAt time.Time
}
