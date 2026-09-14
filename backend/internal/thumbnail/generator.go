// Package thumbnail 提供缩略图/预览图生成（libvips）、缓存文件管理与新鲜度判断。
package thumbnail

import (
	"fmt"

	vips "github.com/davidbyttow/govips/v2/vips"

	"albumshelf/backend/internal/model"
)

// Generate 按变体从源图生成 JPEG：
// thumb 变体 fit 进 sizeBucket×sizeBucket 盒（sizeBucket 须为合法桶）；
// preview 变体 fit 进 PreviewLongEdge×PreviewLongEdge 盒（忽略 sizeBucket）。
// 不裁剪、不放大：原图长边已 ≤ 盒尺寸时保持原尺寸仅转 JPEG。
// 返回导出的 JPEG 字节与缩放后实际宽高。
func Generate(sourcePath string, variant string, sizeBucket int) ([]byte, int, int, error) {
	box, err := boxSize(variant, sizeBucket)
	if err != nil {
		return nil, 0, 0, err
	}

	if err := ensureVipsInitialized(); err != nil {
		return nil, 0, 0, fmt.Errorf("初始化 libvips 失败: %w", err)
	}

	// NewThumbnailWithSizeFromFile 加载并同时完成缩放：
	// InterestingNone 表示不智能裁剪（保持整图 fit 盒内），SizeDown 表示仅缩小不放大。
	img, err := vips.NewThumbnailWithSizeFromFile(sourcePath, box, box, vips.InterestingNone, vips.SizeDown)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("加载图片失败 %s: %w", sourcePath, err)
	}
	defer img.Close()

	// StripMetadata 去除 EXIF 等元数据，减小缓存体积；
	// preview 启用渐进式编码（浏览器边传边显，弱网首屏更快）；
	// thumb 为 200~500px 小图，渐进编码体积反增，保持 baseline。
	jpegBytes, _, err := img.ExportJpeg(&vips.JpegExportParams{
		StripMetadata: true,
		Quality:       model.JPEGQuality,
		Interlace:     variant == model.VariantPreview,
	})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("导出 JPEG 失败 %s: %w", sourcePath, err)
	}
	return jpegBytes, img.Width(), img.Height(), nil
}

// GenerateThumbFromPreview 从已生成的 preview 缓存 JPEG 二次缩放生成 thumb 桶产物：
// preview 已是长边 ≤ PreviewLongEdge 的小图，从中派生 thumb 可避免对原图的重复大图解码。
// vips_thumbnail 从文件路径加载时对 JPEG 自动 shrink-on-load（按 preview 长边与
// 目标桶的比例计算整数预缩因子，仅解码必要像素），缩放/导出参数与 Generate 完全一致
//（fit 盒、不放大、q80、baseline）。
func GenerateThumbFromPreview(previewPath string, sizeBucket int) ([]byte, int, int, error) {
	return Generate(previewPath, model.VariantThumb, sizeBucket)
}

// boxSize 解析变体对应的 fit 盒边长。
func boxSize(variant string, sizeBucket int) (int, error) {
	switch variant {
	case model.VariantThumb:
		if !model.IsValidThumbBucket(sizeBucket) {
			return 0, fmt.Errorf("非法缩略图尺寸桶: %d", sizeBucket)
		}
		return sizeBucket, nil
	case model.VariantPreview:
		return model.PreviewLongEdge, nil
	default:
		return 0, fmt.Errorf("未知缩略图变体: %s", variant)
	}
}
