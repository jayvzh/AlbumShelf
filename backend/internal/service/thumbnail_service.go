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
// prio 为发起者（执行者）的优先级：更高优先级的请求撞上时可绕过等待双跑插队。
type flight struct {
	done chan struct{}
	prio thumbnail.Priority
}

// NewThumbnailService 构造 ThumbnailService。
func NewThumbnailService(fs *filesystem.LocalFilesystem, dataDir string, repo *repository.ImageCacheRepository, sched *thumbnail.Scheduler) *ThumbnailService {
	return &ThumbnailService{fs: fs, dataDir: dataDir, repo: repo, sched: sched, negTTL: negativeCacheTTL}
}

// GetThumb 返回 thumb 变体缩略图：命中缓存直接读文件，未命中生成后落盘返回。
// width 非合法桶（200/300/500）时按 300 处理。prio 由调用方按场景传入：
// 用户可见请求（胶片条/网格 <img>）恒为高优先级，前端目录预热（X-Load-Priority: low）为低。
// 错误语义：路径类错误原样返回（handler 映射 4xx），生成失败返回包装错误（handler 映射 500）。
func (s *ThumbnailService) GetThumb(ctx context.Context, path string, width int, prio thumbnail.Priority) (model.Image, *os.File, error) {
	if !model.IsValidThumbBucket(width) {
		width = 300
	}

	info, cachePath, err := s.prepare(path, model.VariantThumb, width)
	if err != nil {
		return model.Image{}, nil, err
	}
	return s.deliver(ctx, path, model.VariantThumb, width, info, cachePath, prio)
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

		f := &flight{done: make(chan struct{}), prio: prio}
		actual, loaded := s.inflight.LoadOrStore(cachePath, f)
		if loaded {
			fl := actual.(*flight)
			if prio >= fl.prio {
				// 等待者（优先级不高于在飞任务）：等执行者收尾（落盘 / 失败 / 被取消）
				<-fl.done
				if file, err := os.Open(cachePath); err == nil {
					return info, file, nil
				}
				continue
			}
			// 高优先级请求撞上低优先级在飞任务（用户点击 vs 后台预热同一张图）：
			// 等待低优任务等于把用户请求降级到它的队列位置，不可接受——
			// 以自身优先级双跑一次生成，WriteAtomic 保证并发写一致，多付一次
			// 生成换用户即时响应；不触碰 flight（收尾/Delete/close 仍归原低优
			// 执行者），其余低优等待者不受影响。
			oc := s.enqueue(ctx, prio, path, variant, sizeBucket, info, cachePath)
			if oc.cancelled {
				return model.Image{}, nil, ctx.Err()
			}
			return s.settleGen(path, variant, cachePath, info, oc)
		}

		// 执行者：经调度器排队执行 生成 → 落盘 → 唤醒等待者；索引登记移出生成槽，
		// 由 Submit 返回（槽位已释放）后的 fire-and-forget goroutine best-effort 补记。
		// Delete 必须先于 close(done)：若反过来，等待者可能 load 到已关闭的旧 flight
		// 并与新 flight 竞争前空转（热自旋）。
		oc := s.enqueue(ctx, prio, path, variant, sizeBucket, info, cachePath)
		s.inflight.Delete(cachePath)
		// 负缓存已在 enqueue 内记录（须先于 close(done)：等待者被唤醒后回循环顶部
		// 即查负缓存，须能看到该条目）。
		close(f.done)

		if oc.cancelled {
			// 排队中被 ctx 取消：run 未执行，直接把取消带给调用方
			return model.Image{}, nil, ctx.Err()
		}
		return s.settleGen(path, variant, cachePath, info, oc)
	}
}

// settleGen 处理一次生成结果并返回交付三元组：成功时槽外补记索引，
// thumb 变体触发趁热级联 preview（源文件仍在 page cache）。
// deliver 执行者与高优先级双跑路径共用。
func (s *ThumbnailService) settleGen(path, variant, cachePath string, info model.Image, oc genOutcome) (model.Image, *os.File, error) {
	if oc.genErr == nil {
		// 索引登记不占用生成槽：失败仅记日志，不阻断请求（缓存文件已生成，可用性优先）
		go s.index(path, variant, cachePath, info, oc.width, oc.height)
		if variant == model.VariantThumb {
			s.cascadePreview(path, info)
		}
	}
	return info, oc.file, oc.genErr
}

// genOutcome 一次排队生成的结果。cancelled 表示排队中被 ctx 取消（run 未执行），
// 与 genErr（libvips/落盘失败，已记负缓存与日志）互斥。
type genOutcome struct {
	file          *os.File
	width, height int
	genErr        error
	cancelled     bool
}

// enqueue 经调度器排队执行一次生成并记录耗时/失败日志与负缓存。
// 只做单次 Submit——供 deliver 执行者与后台级联复用；调用方自行管理 singleflight。
func (s *ThumbnailService) enqueue(ctx context.Context, prio thumbnail.Priority, path, variant string, sizeBucket int, info model.Image, cachePath string) genOutcome {
	enqueuedAt := time.Now()
	var file *os.File
	var w, h int
	var genErr error
	err := s.sched.Submit(ctx, prio, func() {
		queueWait := time.Since(enqueuedAt)
		start := time.Now()
		file, w, h, genErr = s.generate(path, variant, sizeBucket, info, cachePath)
		log.Printf("[thumbnail] %s %s 生成 %s（排队 %s）",
			variant, path, time.Since(start).Round(time.Millisecond), queueWait.Round(time.Millisecond))
	})
	if err != nil {
		return genOutcome{cancelled: true}
	}
	if genErr != nil {
		log.Printf("[thumbnail] %s %s 生成失败（负缓存 TTL 内同键请求快速失败）: %v",
			variant, path, genErr)
		s.recordNegative(cachePath, genErr)
	}
	return genOutcome{file: file, width: w, height: h, genErr: genErr}
}

// cascadePreview 趁热级联生成 preview：紧随 thumb 生成调用（此刻原图刚被完整读取，
// 仍在 page cache，生成 preview 只花 CPU 不再付冷读代价）。以低优先级入队，
// 不与用户当前查看的图抢占；background ctx 不受请求取消影响。
// 已有同键生成在飞（singleflight 竞选失败）则放弃，由在飞任务兜底。
func (s *ThumbnailService) cascadePreview(path string, info model.Image) {
	previewPath, err := thumbnail.CachePath(s.dataDir, path, model.VariantPreview,
		model.PreviewLongEdge, info.ModifiedAt.UnixNano(), info.Size)
	if err != nil || thumbnail.IsFresh(previewPath) {
		return
	}
	f := &flight{done: make(chan struct{})}
	if _, loaded := s.inflight.LoadOrStore(previewPath, f); loaded {
		return
	}
	go func() {
		oc := s.enqueue(context.Background(), thumbnail.PriorityLow,
			path, model.VariantPreview, model.PreviewLongEdge, info, previewPath)
		s.inflight.Delete(previewPath)
		close(f.done)
		if oc.file != nil {
			oc.file.Close()
		}
		if !oc.cancelled && oc.genErr == nil {
			go s.index(path, model.VariantPreview, previewPath, info, oc.width, oc.height)
		}
	}()
}

// WarmThumb 后台预热：确保 path 的 thumb 缓存存在（低优先级），返回是否实际生成。
// 供 CacheWarmer 调用；preview 新鲜时 render 自动从 preview 派生（小 IO），
// 复用 deliver 全部语义（singleflight/负缓存/成功后级联 preview）。
// 已缓存或负缓存命中时不做任何生成；调用方以串行节奏调用即可天然限流。
func (s *ThumbnailService) WarmThumb(ctx context.Context, path string) (bool, error) {
	info, cachePath, err := s.prepare(path, model.VariantThumb, 300)
	if err != nil {
		return false, err
	}
	if thumbnail.IsFresh(cachePath) {
		return false, nil
	}
	_, file, err := s.deliver(ctx, path, model.VariantThumb, 300, info, cachePath, thumbnail.PriorityLow)
	if file != nil {
		file.Close()
	}
	return true, err
}

// WarmPreview 后台预热：确保 path 的 preview 缓存存在（低优先级），返回是否实际生成。
// 预热顺序"先 preview 后 thumb"：preview 落盘后 WarmThumb 自动从它派生，
// 每张图只读一次原图、只做一次大解码。
func (s *ThumbnailService) WarmPreview(ctx context.Context, path string) (bool, error) {
	info, cachePath, err := s.prepare(path, model.VariantPreview, model.PreviewLongEdge)
	if err != nil {
		return false, err
	}
	if thumbnail.IsFresh(cachePath) {
		return false, nil
	}
	_, file, err := s.deliver(ctx, path, model.VariantPreview, model.PreviewLongEdge, info, cachePath, thumbnail.PriorityLow)
	if file != nil {
		file.Close()
	}
	return true, err
}

// FrontendIdleFor 返回距最近一次前台（高优先级）生成请求的时长，供预热器闲时判定。
func (s *ThumbnailService) FrontendIdleFor() time.Duration {
	return s.sched.FrontendIdleFor()
}

// NoteFrontendWarmTraffic 记录一次前端低优先级（X-Load-Priority: low）HTTP 请求到达，
// 供后台预热器判定前端闲时预热（WARMUP/DIRWARM）是否仍在推进。仅 handler 层调用。
func (s *ThumbnailService) NoteFrontendWarmTraffic() {
	s.sched.NoteFrontendWarmTraffic()
}

// FrontendWarmIdleFor 返回距最近一次前端低优先级请求的时长，供预热器闲时判定。
func (s *ThumbnailService) FrontendWarmIdleFor() time.Duration {
	return s.sched.FrontendWarmIdleFor()
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
