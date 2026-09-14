package service

import (
	"context"
	"errors"
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
// 同键并发生成用 singleflight 去重：仅一个请求执行生成，其余等待后直接读缓存文件；
// 生成动作经全局 Scheduler 排队（并发上限 + 优先级 + 排队中随 ctx 取消）。
type ThumbnailService struct {
	fs      *filesystem.LocalFilesystem
	dataDir string
	repo    *repository.ImageCacheRepository
	sched   *thumbnail.Scheduler

	// inflight 进行中的生成任务：缓存键 → *flight
	inflight sync.Map

	// negCache 生成失败负缓存：缓存键（cachePath）→ negativeEntry。
	// TTL 内同键请求直接返回记录的错误，不再进入调度器触发注定失败的 libvips 解码；
	// TTL 过期自动失效允许重试。取消类错误不入缓存（见 negativeCacheTTL 注释）。
	negCache sync.Map

	// negTTL 负缓存存活时长，测试可注入短值。仅进程内状态，不影响正确性。
	negTTL time.Duration
}

// negativeCacheTTL 生成失败负缓存时长：覆盖连续翻页窗口内对同一坏图的反复请求，
// 过短则退化为无负缓存，过长则短暂故障（如磁盘瞬满）后迟迟不重试。
const negativeCacheTTL = 60 * time.Second

// negativeEntry 负缓存条目：失败时间起算，expireAt 后视为过期。
type negativeEntry struct {
	err      error
	expireAt time.Time
}

// flight 表示一次进行中的缓存生成；done 关闭即产物已落盘、生成已失败或被取消。
type flight struct {
	done chan struct{}
}

// NewThumbnailService 构造 ThumbnailService。
func NewThumbnailService(fs *filesystem.LocalFilesystem, dataDir string, repo *repository.ImageCacheRepository, sched *thumbnail.Scheduler) *ThumbnailService {
	return &ThumbnailService{fs: fs, dataDir: dataDir, repo: repo, sched: sched, negTTL: negativeCacheTTL}
}

// GetThumb 返回 thumb 变体缩略图：命中缓存直接读文件，未命中生成后落盘返回。
// width 非合法桶（200/300/500）时按 300 处理。用户正在浏览的页面，恒为高优先级。
// 错误语义：路径类错误原样返回（handler 映射 4xx），生成失败返回包装错误（handler 映射 500）。
func (s *ThumbnailService) GetThumb(ctx context.Context, path string, width int) (model.Image, *os.File, error) {
	if !model.IsValidThumbBucket(width) {
		width = 300
	}

	info, cachePath, err := s.prepare(path, model.VariantThumb, width)
	if err != nil {
		return model.Image{}, nil, err
	}
	return s.deliver(ctx, path, model.VariantThumb, width, info, cachePath, thumbnail.PriorityHigh)
}

// GetPreview 返回 preview 变体预览图，流程同 GetThumb（bucket 固定 PreviewLongEdge）；
// prio 由调用方按场景传入（当前查看=高，后台预热=低）。
// 生成失败时回退返回原图（actualVariant=original），图片查看链路可用性优先。
func (s *ThumbnailService) GetPreview(ctx context.Context, path string, prio thumbnail.Priority) (model.Image, *os.File, string, error) {
	info, cachePath, err := s.prepare(path, model.VariantPreview, model.PreviewLongEdge)
	if err != nil {
		// 路径非法/不存在等：原样返回哨兵错误交由 handler 映射，不回退
		return model.Image{}, nil, "", err
	}

	img, file, err := s.deliver(ctx, path, model.VariantPreview, model.PreviewLongEdge, info, cachePath, prio)
	if err == nil {
		return img, file, model.VariantPreview, nil
	}
	// 请求已被取消（翻页/切目录）：调用方已不关心结果，直接返回，不再浪费一次原图打开
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return model.Image{}, nil, "", err
	}

	// 首次失败已在 deliver 记录负缓存时打过日志；负缓存 TTL 内的重复请求
	// 快速失败走此回退，保持静默，避免坏图反复浏览时刷屏。
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
// 未命中经 singleflight 去重后经调度器生成——等待者等执行者收尾后开缓存文件。
// 执行者被取消（翻页/切目录丢弃）或生成失败时，等待者回到循环顶部重新竞选执行者，
// 因此循环必然终止：只要本请求 ctx 未取消，最终要么命中缓存要么自己执行生成。
func (s *ThumbnailService) deliver(ctx context.Context, path, variant string, sizeBucket int, info model.Image, cachePath string, prio thumbnail.Priority) (model.Image, *os.File, error) {
	for {
		if thumbnail.IsFresh(cachePath) {
			file, err := os.Open(cachePath)
			if err == nil {
				return info, file, nil
			}
			// 极端竞态（缓存文件被外部清理）：视为 miss 转生成
			log.Printf("[thumbnail] 缓存命中但打开失败，转重新生成: %s", cachePath)
		}

		// 负缓存命中（该键近期刚生成失败且未过期）：直接返回记录的错误，
		// 不再入队/竞选执行者，避免反复调度注定失败的 libvips 解码。
		if err := s.checkNegative(cachePath); err != nil {
			return model.Image{}, nil, err
		}

		f := &flight{done: make(chan struct{})}
		actual, loaded := s.inflight.LoadOrStore(cachePath, f)
		if loaded {
			// 等待者：等执行者收尾（落盘 / 失败 / 被取消）
			<-actual.(*flight).done
			if file, err := os.Open(cachePath); err == nil {
				return info, file, nil
			}
			continue
		}

		// 执行者：经调度器排队执行 生成 → 落盘 → 唤醒等待者；索引登记移出生成槽，
		// 由 Submit 返回（槽位已释放）后的 fire-and-forget goroutine best-effort 补记。
		// Delete 必须先于 close(done)：若反过来，等待者可能 load 到已关闭的旧 flight
		// 并与新 flight 竞争前空转（热自旋）。
		enqueuedAt := time.Now()
		var genFile *os.File
		var genW, genH int
		var genErr error
		err := s.sched.Submit(ctx, prio, func() {
			queueWait := time.Since(enqueuedAt)
			start := time.Now()
			genFile, genW, genH, genErr = s.generate(path, variant, sizeBucket, info, cachePath)
			log.Printf("[thumbnail] %s %s 生成 %s（排队 %s）",
				variant, path, time.Since(start).Round(time.Millisecond), queueWait.Round(time.Millisecond))
		})
		s.inflight.Delete(cachePath)
		// 记录须先于 close(done)：等待者被唤醒后回循环顶部即查负缓存，须能看到该条目。
		// 首次生成失败在此打一次日志（含回退语义：preview 由 GetPreview 回退原图、
		// thumb 由 handler 返回 500）；负缓存 TTL 内的重复请求快速失败，不再刷屏。
		if err == nil && genErr != nil {
			log.Printf("[thumbnail] %s %s 生成失败（负缓存 TTL 内同键请求快速失败）: %v",
				variant, path, genErr)
			s.recordNegative(cachePath, genErr)
		}
		close(f.done)

		if err != nil {
			// 排队中被 ctx 取消：run 未执行，直接把取消带给调用方
			return model.Image{}, nil, err
		}
		if genErr == nil {
			// 索引登记不占用生成槽：失败仅记日志，不阻断请求（缓存文件已生成，可用性优先）
			go s.index(path, variant, cachePath, info, genW, genH)
		}
		return info, genFile, genErr
	}
}

// recordNegative 记录生成失败负缓存；取消类错误不入缓存——它们不代表图坏，
// 请求取消（翻页/切目录）后同键的其他请求仍应正常尝试生成。
func (s *ThumbnailService) recordNegative(cachePath string, err error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	ttl := s.negTTL
	if ttl <= 0 {
		ttl = negativeCacheTTL
	}
	s.negCache.Store(cachePath, negativeEntry{err: err, expireAt: time.Now().Add(ttl)})
}

// checkNegative 返回未过期负缓存记录的错误；过期即清除并放行重新生成。
func (s *ThumbnailService) checkNegative(cachePath string) error {
	v, ok := s.negCache.Load(cachePath)
	if !ok {
		return nil
	}
	entry := v.(negativeEntry)
	if time.Now().After(entry.expireAt) {
		s.negCache.Delete(cachePath)
		return nil
	}
	return entry.err
}

// generate 生成变体产物并原子写入缓存文件，打开缓存文件返回（索引由槽外补记）。
// 文件句柄交由调用方（handler）关闭。
// 槽内仅保留 libvips 解码 + 落盘 + 打开；thumb 变体优先从同源 preview 缓存派生，
// 避免重复解码原图大图。
func (s *ThumbnailService) generate(path, variant string, sizeBucket int, info model.Image, cachePath string) (*os.File, int, int, error) {
	data, width, height, err := s.render(path, variant, sizeBucket, info)
	if err != nil {
		return nil, 0, 0, err
	}
	if err := thumbnail.WriteAtomic(cachePath, data); err != nil {
		return nil, 0, 0, err
	}

	file, err := os.Open(cachePath)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("打开缓存文件失败 %s: %w", cachePath, err)
	}
	return file, width, height, nil
}

// render 产出变体图像数据：thumb 变体在同源 preview 缓存存在且新鲜时优先从 preview
// 派生（省去原图解码），preview 缺失或派生失败均回退原图生成（记日志，不向上抛错）；
// 缓存键与失效语义不变——派生用与原图同一 mtime/size 构造的 preview 缓存键定位。
func (s *ThumbnailService) render(path, variant string, sizeBucket int, info model.Image) ([]byte, int, int, error) {
	if variant == model.VariantThumb {
		if previewPath, ok := s.freshPreviewPath(path, info); ok {
			data, w, h, err := thumbnail.GenerateThumbFromPreview(previewPath, sizeBucket)
			if err == nil {
				return data, w, h, nil
			}
			log.Printf("[thumbnail] preview 派生 thumb 失败，回退原图生成 %s: %v", path, err)
		}
	}
	// GetMetadata 已校验 info.Path 在 IMAGE_ROOT 内，Join 还原磁盘绝对路径供 libvips 读取
	return thumbnail.Generate(filepath.Join(s.fs.Root(), info.Path), variant, sizeBucket)
}

// freshPreviewPath 判断同源 preview 缓存是否可用于派生：用与 prepare 一致的
// mtime/size/变体/桶参数构造 preview 缓存键（原图变化 → 键变 → 必然 miss），
// 再按 IsFresh 判定文件存在且完整。
func (s *ThumbnailService) freshPreviewPath(path string, info model.Image) (string, bool) {
	previewPath, err := thumbnail.CachePath(s.dataDir, path, model.VariantPreview,
		model.PreviewLongEdge, info.ModifiedAt.UnixNano(), info.Size)
	if err != nil {
		return "", false
	}
	return previewPath, thumbnail.IsFresh(previewPath)
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
