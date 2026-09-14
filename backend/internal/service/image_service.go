package service

import (
	"context"
	"fmt"
	"os"

	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/thumbnail"
)

// VariantOriginal 原图变体标识（X-Image-Variant 响应头取值之一）。
const VariantOriginal = "original"

// ImageService 图片查看业务编排：读取原图与元信息；
// preview 变体委托 ThumbnailService 生成（Sprint 3），其余变体一律返回原图。
type ImageService struct {
	fs         filesystem.Filesystem
	thumbnails *ThumbnailService
}

// NewImageService 构造 ImageService。
func NewImageService(fs filesystem.Filesystem, thumbnails *ThumbnailService) *ImageService {
	return &ImageService{fs: fs, thumbnails: thumbnails}
}

// Get 打开图片文件，返回元信息、文件句柄与实际变体；调用方负责关闭句柄。
// variant=preview → 走缩略图服务预览链路（生成失败自动回退原图并标注 original），
// prio 透传给生成调度器（当前查看=高，后台预热=低）；
// original/空及其他任意值均按原图处理，不报错。
func (s *ImageService) Get(ctx context.Context, path, variant string, prio thumbnail.Priority) (model.Image, *os.File, string, error) {
	if variant == model.VariantPreview {
		return s.thumbnails.GetPreview(ctx, path, prio)
	}

	img, file, err := s.fs.OpenImage(path)
	if err != nil {
		return model.Image{}, nil, "", err
	}
	return img, file, VariantOriginal, nil
}

// GetInfo 返回图片元信息（不打开文件）。
func (s *ImageService) GetInfo(path string) (model.Image, error) {
	return s.fs.GetMetadata(path)
}

// ETag 基于 mtime+size 生成强 ETag（含双引号），/image 与 /thumbnail 共用同一规范。
func ETag(img model.Image) string {
	return fmt.Sprintf("\"%d-%d\"", img.ModifiedAt.UnixNano(), img.Size)
}

// ContentType 按扩展名（含点小写，来自 model.File.Extension）映射 MIME 类型；
// 未知扩展名兜底为通用二进制流。
func ContentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
