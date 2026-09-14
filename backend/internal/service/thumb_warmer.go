package service

import (
	"context"
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"strings"
	"time"

	"albumshelf/backend/internal/filesystem"
)

// ThumbWarmer 后台缩略图预热器：周期性递归扫描 IMAGE_ROOT，对缺失 thumb 缓存的
// 图片经调度器以低优先级逐张补齐——用户请求恒为高优先级，预热只在空闲槽位推进，
// 不与浏览体验抢占。thumb 生成成功后由 ThumbnailService 的级联逻辑趁热补 preview，
// 因此一轮预热完成后用户首次进入任何目录 thumbs/preview 双热。
//
// 跳过以 . 开头的隐藏目录/文件（与目录扫描规则一致）；坏图依赖负缓存 TTL
// 内快速失败，每轮扫描自动重试。
type ThumbWarmer struct {
	fs       *filesystem.LocalFilesystem
	svc      *ThumbnailService
	interval time.Duration // 两轮全库扫描间隔
}

// NewThumbWarmer 构造预热器。interval 为两轮扫描间隔（首轮启动立即执行）。
func NewThumbWarmer(fs *filesystem.LocalFilesystem, svc *ThumbnailService, interval time.Duration) *ThumbWarmer {
	return &ThumbWarmer{fs: fs, svc: svc, interval: interval}
}

// Run 阻塞循环：立即跑第一轮 → 休眠 interval → 重复；ctx 取消即退出。
func (w *ThumbWarmer) Run(ctx context.Context) {
	for {
		w.pass(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(w.interval):
		}
	}
}

// pass 单轮扫描：WalkDir 字典序确定性遍历，串行逐张预热（WarmThumb 同步等待
// 生成完成，天然限流为队列中至多一个预热任务）。已缓存图片仅付 stat 成本。
func (w *ThumbWarmer) pass(ctx context.Context) {
	start := time.Now()
	root := w.fs.Root()
	var generated, cached, failed int

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// 单目录读取失败（权限等）跳过该子树，继续遍历其余部分
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !filesystem.IsImageExt(filepath.Ext(name)) {
			return nil
		}

		// root（无尾斜杠）截断后自带前导 /，用 path.Join 清洗避免 // 双斜杠
		rel := path.Join("/", filepath.ToSlash(strings.TrimPrefix(p, root)))
		g, err := w.svc.WarmThumb(ctx, rel)
		switch {
		case err != nil:
			failed++
		case g:
			generated++
		default:
			cached++
		}
		return nil
	})
	if err != nil {
		log.Printf("[warmer] 预热扫描异常终止: %v", err)
	}
	log.Printf("[warmer] 预热扫描完成: 新增 %d，已缓存 %d，失败 %d，耗时 %s",
		generated, cached, failed, time.Since(start).Round(time.Millisecond))
}
