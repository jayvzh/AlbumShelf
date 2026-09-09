package model

import "time"

// 缩略图变体：thumb 为列表小图（合法尺寸桶 200/300/500），preview 为查看大图。
const (
	VariantThumb   = "thumb"
	VariantPreview = "preview"
)

// preview 变体输出长边上限与 JPEG 输出质量（规格：长边 2560px JPEG q80）。
const (
	PreviewLongEdge = 2560
	JPEGQuality     = 80
)

// IsValidThumbBucket 判断是否为合法的 thumb 变体尺寸桶。
func IsValidThumbBucket(size int) bool {
	switch size {
	case 200, 300, 500:
		return true
	default:
		return false
	}
}

// ThumbnailCache 对应 image_cache 表一行（DATA_MODEL §2.5），
// 记录某张原图某变体的缩略图产物位置与失效判定依据。
type ThumbnailCache struct {
	SourcePath  string // 原图相对路径（/Comics/001.jpg 风格）
	Variant     string // thumb | preview
	CachePath   string // 缓存文件路径 dataDir/cache/{variant}/hash.jpg，表内 UNIQUE
	SourceMtime int64  // 生成时原图 mtime（UnixNano），失效判定依据
	SourceSize  int64  // 生成时原图字节数，失效判定依据
	Width       int    // 缩放后实际宽
	Height      int    // 缩放后实际高
	CreatedAt   time.Time
}
