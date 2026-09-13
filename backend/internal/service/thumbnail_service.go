package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
	"albumshelf/backend/internal/thumbnail"
)

// ThumbnailService 缩略图/预览图业务编排：
// 缓存命中判断（键已编码原图 mtime/size）→ 未命中 libvips 生成 → 原子落盘 → SQLite 索引登记。
// 同键并发生成用 singleflight 去重：仅一个请求执行生成，其余等待后直接读缓存文件。
type ThumbnailService struct {
	fs      *filesystem.LocalFilesystem
	dataDir string
	repo    *repository.ImageCacheRepository

	// inflight 进行中的生成任务：缓存键 → *flight
	inflight sync.Map
}

// flight 表示一次进行中的缓存生成；done 关闭即产物已落盘或生成已失败。
type flight struct {
	done chan struct{}
}

// NewThumbnailService 构造 ThumbnailService。
func NewThumbnailService(fs *filesystem.LocalFilesystem, dataDir string, repo *repository.ImageCacheRepository) *ThumbnailService {
	return &ThumbnailService{fs: fs, dataDir: dataDir, repo: repo}
}

// GetThumb 返回 thumb 变体缩略图：命中缓存直接读文件，未命中生成后落盘返回。
// width 非合法桶（200/300/500）时按 300 处理。
// 错误语义：路径类错误原样返回（handler 映射 4xx），生成失败返回包装错误（handler 映射 500）。
func (s *ThumbnailService) GetThumb(path string, width int) (model.Image, *os.File, error) {
	if !model.IsValidThumbBucket(width) {
		width = 300
	}

	info, cachePath, err := s.prepare(path, model.VariantThumb, width)
	if err != nil {
		return model.Image{}, nil, err
	}
	return s.deliver(path, model.VariantThumb, width, info, cachePath)
}

// GetPreview 返回 preview 变体预览图，流程同 GetThumb（bucket 固定 PreviewLongEdge）；
// 生成失败时回退返回原图（actualVariant=original），图片查看链路可用性优先。
func (s *ThumbnailService) GetPreview(path string) (model.Image, *os.File, string, error) {
	info, cachePath, err := s.prepare(path, model.VariantPreview, model.PreviewLongEdge)
	if err != nil {
		// 路径非法/不存在等：原样返回哨兵错误交由 handler 映射，不回退
		return model.Image{}, nil, "", err
	}

	img, file, err := s.deliver(path, model.VariantPreview, model.PreviewLongEdge, info, cachePath)
	if err == nil {
		return img, file, model.VariantPreview, nil
	}

	log.Printf("[thumbnail] preview 生成失败，回退原图 %s: %v", path, err)
	origImg, origFile, openErr := s.fs.OpenImage(path)
	if openErr != nil {
		return model.Image{}, nil, "", openErr
	}
	return origImg, origFile, VariantOriginal, nil
}

// prepare 路径安全校验并计算缓存路径：GetMetadata 校验路径在 IMAGE_ROOT 内且为图片文件，
// 同时取得进缓存键与 ETag 的原图 mtime/size；失败原样返回 filesystem 哨兵错误。
func (s *ThumbnailService) prepare(path, variant string, sizeBucket int) (model.Image, string, error) {
	info, err := s.fs.GetMetadata(path)
	if err != nil {
		return model.Image{}, "", err
	}

	cachePath, err := thumbnail.CachePath(s.dataDir, path, variant, sizeBucket,
		info.ModifiedAt.UnixNano(), info.Size)
	if err != nil {
		return model.Image{}, "", err
	}
	return info, cachePath, nil
}

// deliver 返回缓存产物：命中（该键下文件存在即新鲜）直接打开；
// 未命中经 singleflight 去重后生成——等待者等执行者完成即按 cachePath 开文件。
func (s *ThumbnailService) deliver(path, variant string, sizeBucket int, info model.Image, cachePath string) (model.Image, *os.File, error) {
	if thumbnail.IsFresh(cachePath) {
		file, err := os.Open(cachePath)
		if err == nil {
			return info, file, nil
		}
		// 极端竞态（缓存文件被外部清理）：视为 miss 转生成
		log.Printf("[thumbnail] 缓存命中但打开失败，转重新生成: %s", cachePath)
	}

	f := &flight{done: make(chan struct{})}
	actual, loaded := s.inflight.LoadOrStore(cachePath, f)
	if loaded {
		// 等待者：执行者 close(done) 前产物必已落盘，直接开文件
		<-actual.(*flight).done
		file, err := os.Open(cachePath)
		if err != nil {
			return model.Image{}, nil, fmt.Errorf("等待并发生成后打开缓存失败 %s: %w", cachePath, err)
		}
		return info, file, nil
	}

	// 执行者：生成 → 落盘 → 索引登记 → 唤醒等待者
	defer s.inflight.Delete(cachePath)
	file, err := s.generate(path, variant, sizeBucket, info, cachePath)
	close(f.done)
	return info, file, err
}

// generate libvips 生成变体产物并原子写入缓存文件，登记索引后打开缓存文件返回。
// 文件句柄交由调用方（handler）关闭。
func (s *ThumbnailService) generate(path, variant string, sizeBucket int, info model.Image, cachePath string) (*os.File, error) {
	// GetMetadata 已校验 info.Path 在 IMAGE_ROOT 内，Join 还原磁盘绝对路径供 libvips 读取
	data, width, height, err := thumbnail.Generate(filepath.Join(s.fs.Root(), info.Path), variant, sizeBucket)
	if err != nil {
		return nil, err
	}
	if err := thumbnail.WriteAtomic(cachePath, data); err != nil {
		return nil, err
	}
	s.index(path, variant, cachePath, info, width, height)

	file, err := os.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("打开缓存文件失败 %s: %w", cachePath, err)
	}
	return file, nil
}

// index 登记缩略图索引（best-effort）：先删同源同变体旧记录再 Upsert（顺序不能反，
// 否则会删掉新记录）；索引失败仅记日志不阻断响应——缓存文件已生成，图片可用性优先。
func (s *ThumbnailService) index(path, variant, cachePath string, info model.Image, width, height int) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := s.repo.DeleteBySource(ctx, path, variant); err != nil {
		log.Printf("[thumbnail] 清理旧索引失败（忽略）: %v", err)
	}
	record := model.ThumbnailCache{
		SourcePath:  path,
		Variant:     variant,
		CachePath:   cachePath,
		SourceMtime: info.ModifiedAt.UnixNano(),
		SourceSize:  info.Size,
		Width:       width,
		Height:      height,
		CreatedAt:   time.Now(),
	}
	if err := s.repo.Upsert(ctx, record); err != nil {
		log.Printf("[thumbnail] 缓存索引登记失败（忽略）: %v", err)
	}
}
