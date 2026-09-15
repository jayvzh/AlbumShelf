package service

import (
	"context"
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
)

// warmIdleThreshold 前台静默多久后预热器才推进：前台高优先请求（缩略图/预览图）
// 持续出现 = 正在浏览，此时预热完全让路——优先级只决定排队顺序，挡不住在飞
// 任务与磁盘 IO 竞争，必须等静默。
const warmIdleThreshold = 30 * time.Second

// warmLevelKey 预热级别在 app_settings 表中的键。
const warmLevelKey = "warm_level"

// warmPhase 预热器运行阶段（进度展示用）。
type warmPhase int32

const (
	phaseIdle    warmPhase = iota // 空闲（级别 off / 一轮完成 / 未启动）
	phaseWarming                  // 预热进行中
	phasePaused                   // 前台浏览中，预热暂停
)

func (p warmPhase) String() string {
	switch p {
	case phaseWarming:
		return "warming"
	case phasePaused:
		return "paused"
	default:
		return "idle"
	}
}

// WarmStatus 预热器状态快照（设置页进度展示）。
type WarmStatus struct {
	Level         model.WarmLevel `json:"level"`
	Phase         string          `json:"phase"`
	DirsTotal     int64           `json:"dirs_total"`
	DirsDone      int64           `json:"dirs_done"`
	CurrentDir    string          `json:"current_dir"`
	ImagesPlanned int64           `json:"images_planned"`
	ImagesDone    int64           `json:"images_done"`
	LastPassEnd   int64           `json:"last_pass_end"` // unix 秒；0 = 尚未完成过一轮
	Enabled       bool            `json:"enabled"`       // THUMB_WARMER 总开关（env）
}

// warmDir 一个含图目录的预热输入：目录相对路径 + 目录内图片相对路径（文件名字典序）。
type warmDir struct {
	rel   string
	files []string
	mtime time.Time // 目录修改时间，决定预热顺序（新的先）
}

// CacheWarmer 后台缓存预热器：按级别在每个含图目录内以文件名序预热前缀——
// 先 preview 后 thumb（thumb 从新鲜 preview 派生，每张图只读一次原图、
// 只做一次大解码）。前台活跃时暂停推进；严格单任务串行，任一时刻至多占用
// 一个生成槽，另一个槽留给前台。级别运行时可改（持久化到 app_settings），
// 变更后 kick 提前唤醒等待中的循环按新级别重新扫描。
type CacheWarmer struct {
	fs       *filesystem.LocalFilesystem
	svc      *ThumbnailService
	settings *repository.AppSettingsRepository
	interval time.Duration
	enabled  bool
	kick     chan struct{}

	level       atomic.Value // model.WarmLevel
	phase       atomic.Int32
	dirsTotal   atomic.Int64
	dirsDone    atomic.Int64
	curDir      atomic.Value // string
	imgsPlanned atomic.Int64
	imgsDone    atomic.Int64
	lastPassEnd atomic.Int64 // unix 秒
}

// NewCacheWarmer 构造预热器。interval 为两轮扫描间隔（首轮启动立即执行）；
// initialLevel 为持久化的级别（LoadWarmLevel 读取），enabled 为 env 总开关。
func NewCacheWarmer(fs *filesystem.LocalFilesystem, svc *ThumbnailService, settings *repository.AppSettingsRepository, interval time.Duration, initialLevel model.WarmLevel, enabled bool) *CacheWarmer {
	w := &CacheWarmer{
		fs: fs, svc: svc, settings: settings,
		interval: interval, enabled: enabled,
		kick: make(chan struct{}, 1),
	}
	w.level.Store(initialLevel)
	w.curDir.Store("")
	return w
}

// Run 阻塞循环：立即跑第一轮 → 休眠 interval 或被 kick 提前唤醒 → 重复；ctx 取消即退出。
func (w *CacheWarmer) Run(ctx context.Context) {
	for {
		w.pass(ctx)
		select {
		case <-ctx.Done():
			return
		case <-w.kick:
		case <-time.After(w.interval):
		}
	}
}

// SetLevel 更新级别：持久化到 DB 并即时生效（Kick 提前唤醒等待中的循环重新扫描）。
func (w *CacheWarmer) SetLevel(ctx context.Context, level model.WarmLevel) error {
	if err := w.settings.Set(ctx, warmLevelKey, string(level)); err != nil {
		return err
	}
	w.level.Store(level)
	w.Kick()
	return nil
}

// Kick 提前唤醒等待中的循环立即开始新一轮扫描（级别变更/缓存清理后触发按级别重建）。
// 非阻塞：正在扫描时信号被丢弃（扫描自然覆盖新状态），不叠加重复轮次。
func (w *CacheWarmer) Kick() {
	select {
	case w.kick <- struct{}{}:
	default:
	}
}

// Level 当前生效级别。
func (w *CacheWarmer) Level() model.WarmLevel {
	return w.level.Load().(model.WarmLevel)
}

// Status 状态快照（进度 API 用）。
func (w *CacheWarmer) Status() WarmStatus {
	cur, _ := w.curDir.Load().(string)
	return WarmStatus{
		Level:         w.Level(),
		Phase:         warmPhase(w.phase.Load()).String(),
		DirsTotal:     w.dirsTotal.Load(),
		DirsDone:      w.dirsDone.Load(),
		CurrentDir:    cur,
		ImagesPlanned: w.imgsPlanned.Load(),
		ImagesDone:    w.imgsDone.Load(),
		LastPassEnd:   w.lastPassEnd.Load(),
		Enabled:       w.enabled,
	}
}

// LoadWarmLevel 启动时从 DB 读预热级别；缺省或非法值回退默认档。
func LoadWarmLevel(ctx context.Context, settings *repository.AppSettingsRepository) model.WarmLevel {
	if v, ok, err := settings.Get(ctx, warmLevelKey); err == nil && ok {
		if l, valid := model.ParseWarmLevel(v); valid {
			return l
		}
	}
	return model.WarmLevelDefault
}

// pass 单轮：级别 off 或 env 总开关关闭时直接空转；否则收集含图目录，
// 逐目录按级别前缀预热。目录数为进度分母；图片计划数 = 各目录 preview 前缀 + thumb 前缀之和。
func (w *CacheWarmer) pass(ctx context.Context) {
	level := w.Level()
	if level == model.WarmLevelOff || !w.enabled {
		w.setPhase(phaseIdle)
		return
	}
	limits := level.Limits()
	dirs := w.collect()
	w.dirsTotal.Store(int64(len(dirs)))
	w.dirsDone.Store(0)
	var planned int64
	for _, d := range dirs {
		planned += int64(min(len(d.files), limits.Preview)) + int64(min(len(d.files), limits.Thumb))
	}
	w.imgsPlanned.Store(planned)
	w.imgsDone.Store(0)
	w.setPhase(phaseWarming)

	var gen, cached, failed int64
	start := time.Now()
	for _, d := range dirs {
		w.curDir.Store(d.rel)
		if !w.warmDir(ctx, d, limits, &gen, &cached, &failed) {
			return // ctx 取消：保留进度字段供状态查询
		}
		w.dirsDone.Add(1)
	}
	w.setPhase(phaseIdle)
	w.lastPassEnd.Store(time.Now().Unix())
	w.curDir.Store("")
	log.Printf("[warmer] 预热完成（%s）: 目录 %d，生成 %d，已缓存 %d，失败 %d，耗时 %s",
		level, len(dirs), gen, cached, failed, time.Since(start).Round(time.Second))
}

// warmDir 目录内预热：先 preview 前 previewN 张，再 thumb 前 thumbN 张
//（此时 preview 已新鲜，render 自动派生，仅小 IO）。返回 false 表示 ctx 已取消。
func (w *CacheWarmer) warmDir(ctx context.Context, d warmDir, limits model.WarmLimits, gen, cached, failed *int64) bool {
	warm := func(rel string, fn func(context.Context, string) (bool, error)) bool {
		if !w.waitIdle(ctx) {
			return false
		}
		g, err := fn(ctx, rel)
		w.imgsDone.Add(1)
		switch {
		case err != nil:
			*failed++
		case g:
			*gen++
		default:
			*cached++
		}
		return true
	}
	for i, f := range d.files {
		if limits.Preview > 0 && i >= limits.Preview {
			break
		}
		if !warm(f, w.svc.WarmPreview) {
			return false
		}
	}
	for i, f := range d.files {
		if limits.Thumb > 0 && i >= limits.Thumb {
			break
		}
		if !warm(f, w.svc.WarmThumb) {
			return false
		}
	}
	return true
}

// waitIdle 等待前台与前端预热双双静默：
//  1. 高优先级任务（用户正在浏览的图）≥ warmIdleThreshold 未出现；
//  2. 前端低优先级请求（查看器滑窗 WARMUP / 目录预热 DIRWARM，X-Load-Priority: low）
//     ≥ warmIdleThreshold 未出现——同为低优先级会在调度队列与磁盘 IO 上交错，
//     必须等前端预热完成（静默）后再推进，落实"当前目录预热完了最后才是全库后台构建"。
//
// 等待期间 phase 置 paused、恢复时回 warming（仅在这两态间切换，不覆盖 idle）。
// 返回 false 表示 ctx 已取消。
func (w *CacheWarmer) waitIdle(ctx context.Context) bool {
	idle := func() bool {
		return w.svc.FrontendIdleFor() >= warmIdleThreshold && w.svc.FrontendWarmIdleFor() >= warmIdleThreshold
	}
	if idle() {
		return true
	}
	w.phase.CompareAndSwap(int32(phaseWarming), int32(phasePaused))
	defer w.phase.CompareAndSwap(int32(phasePaused), int32(phaseWarming))
	for {
		if idle() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(2 * time.Second):
		}
	}
}

func (w *CacheWarmer) setPhase(p warmPhase) {
	w.phase.Store(int32(p))
}

// collect 递归收集含图目录（跳过以 . 开头的隐藏项，规则同目录扫描）；
// 按目录修改时间降序返回（新目录优先预热），目录内文件按文件名字典序（前缀语义）。
// IMAGE_ROOT 根目录直下的图片归入虚拟目录 "/"。
func (w *CacheWarmer) collect() []warmDir {
	root := w.fs.Root()
	var dirs []warmDir
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 单目录读取失败（权限等）跳过，继续其余部分
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			rel := path.Join("/", filepath.ToSlash(strings.TrimPrefix(p, root)))
			wd := warmDir{rel: rel}
			if info, infoErr := d.Info(); infoErr == nil {
				wd.mtime = info.ModTime()
			}
			dirs = append(dirs, wd)
			return nil
		}
		if strings.HasPrefix(name, ".") || !filesystem.IsImageExt(filepath.Ext(name)) {
			return nil
		}
		if n := len(dirs); n > 0 {
			rel := path.Join("/", filepath.ToSlash(strings.TrimPrefix(p, root)))
			dirs[n-1].files = append(dirs[n-1].files, rel)
		}
		return nil
	})
	out := dirs[:0]
	for _, d := range dirs {
		if len(d.files) > 0 {
			out = append(out, d)
		}
	}
	// 目录修改时间降序：新入库/新更新的目录优先预热
	sort.Slice(out, func(i, j int) bool { return out[i].mtime.After(out[j].mtime) })
	return out
}
