package model

import "time"

// 缩略图变体：thumb 为列表小图（合法尺寸桶 200/300/500），preview 为查看大图。
const (
	VariantThumb   = "thumb"
	VariantPreview = "preview"
)

// preview 变体输出长边上限与 JPEG 输出质量（规格：长边 1920px JPEG q80）。
// 长边取 1920 而非 2560：JPEG shrink-on-load 仅支持 1/2/4/8 倍且不允许放大，
// 主流 12MP 图（长边 4032）在 2560 盒下 4032/2560 < 2 只能整幅解码（NAS 上单张 5~42s，
// 见 logs/albumshelf_2026_9_14.log）；1920 使 4032/1920 = 2.1 触发 factor 2 快速解码
//（解码像素 ÷4），4K 全屏 Fit 仅约 12.5% 放大，照片浏览可接受，放大场景走 original 变体。
const (
	PreviewLongEdge = 1920
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
