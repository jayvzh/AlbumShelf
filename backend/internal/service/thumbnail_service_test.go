package service

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
	"albumshelf/backend/internal/thumbnail"
)

// newTestThumbnailEnv 构造基于临时目录的 ThumbnailService：
// 返回服务与其图片根目录（源图以 /xxx.jpg 相对路径访问）。
func newTestThumbnailEnv(t *testing.T) (*ThumbnailService, string) {
	t.Helper()
	imageRoot := t.TempDir()
	dataDir := t.TempDir()

	db, err := repository.Open(dataDir)
	if err != nil {
		t.Fatalf("打开临时数据库失败: %v", err)
	}
	t.Cleanup(func() { repository.Close(db) })

	svc := NewThumbnailService(filesystem.NewLocalFilesystem(imageRoot), dataDir,
		repository.NewImageCacheRepository(db), thumbnail.NewScheduler(1))
	return svc, imageRoot
}

// writeTestJPEG 用标准库在 imageRoot 下生成 w×h 渐变色 JPEG，返回相对路径。
func writeTestJPEG(t *testing.T, imageRoot, relPath string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	abs := filepath.Join(imageRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}
	f, err := os.Create(abs)
	if err != nil {
		t.Fatalf("创建源图失败: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatalf("编码源图失败: %v", err)
	}
	return relPath
}

// writeBrokenImage 写入扩展名为 .jpg 但内容非图像的文件：
// GetMetadata 仅读文件头（尺寸解析失败不报错），libvips 解码必然失败。
func writeBrokenImage(t *testing.T, imageRoot, relPath string) string {
	t.Helper()
	abs := filepath.Join(imageRoot, relPath)
	if err := os.WriteFile(abs, []byte("definitely not an image body"), 0o644); err != nil {
		t.Fatalf("写入损坏源图失败: %v", err)
	}
	return relPath
}

// previewCachePath 以源图当前 mtime/size 构造 preview 缓存键路径（与派生判定同参）。
func previewCachePath(t *testing.T, svc *ThumbnailService, sourcePath string) string {
	t.Helper()
	info, err := svc.fs.GetMetadata(sourcePath)
	if err != nil {
		t.Fatalf("读取源图元信息失败: %v", err)
	}
	previewPath, err := thumbnail.CachePath(svc.dataDir, sourcePath, model.VariantPreview,
		model.PreviewLongEdge, info.ModifiedAt.UnixNano(), info.Size)
	if err != nil {
		t.Fatalf("构造 preview 缓存路径失败: %v", err)
	}
	return previewPath
}

// preview 存在且新鲜 → thumb 从 preview 派生。以损坏原图为证明：
// 若未走派生路径，回退的原图解码必然失败，GetThumb 不可能成功。
func TestGetThumbDerivesFromFreshPreview(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeBrokenImage(t, imageRoot, "broken.jpg")

	// 种植同源新鲜 preview：由合法供体图生成，写入 /broken.jpg 的 preview 缓存键下
	donor := writeTestJPEG(t, imageRoot, "donor.jpg", 1600, 1200)
	data, _, _, err := thumbnail.Generate(filepath.Join(imageRoot, donor), model.VariantPreview, 0)
	if err != nil {
		t.Fatalf("生成种植用 preview 失败: %v", err)
	}
	if err := thumbnail.WriteAtomic(previewCachePath(t, svc, "/broken.jpg"), data); err != nil {
		t.Fatalf("写入种植 preview 失败: %v", err)
	}

	_, file, err := svc.GetThumb(ctx, "/broken.jpg", 300, thumbnail.PriorityHigh)
	if err != nil {
		t.Fatalf("preview 新鲜时应从 preview 派生成功: %v", err)
	}
	defer file.Close()

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatalf("派生产物应为合法 JPEG: %v", err)
	}
	if cfg.Width > 300 || cfg.Height > 300 {
		t.Fatalf("派生产物超出 300×300 盒: %dx%d", cfg.Width, cfg.Height)
	}
}

// preview 缺失 → thumb 保持现行为从原图生成。
func TestGetThumbFallsBackToOriginalWithoutPreview(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeTestJPEG(t, imageRoot, "ok.jpg", 800, 600)

	_, file, err := svc.GetThumb(ctx, "/ok.jpg", 300, thumbnail.PriorityHigh)
	if err != nil {
		t.Fatalf("原图生成 thumb 应成功: %v", err)
	}
	defer file.Close()

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatalf("产物应为合法 JPEG: %v", err)
	}
	if cfg.Width > 300 || cfg.Height > 300 {
		t.Fatalf("产物超出 300×300 盒: %dx%d", cfg.Width, cfg.Height)
	}
}

// preview 存在但内容损坏 → 派生失败回退原图生成，请求仍成功（错误仅记日志）。
func TestGetThumbFallsBackToOriginalWhenDeriveFails(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeTestJPEG(t, imageRoot, "ok.jpg", 800, 600)

	// 种植损坏 preview：以 EOI 尾标记骗过 IsFresh，但内容非合法 JPEG
	brokenPreview := append([]byte("junk"), 0xFF, 0xD9)
	if err := thumbnail.WriteAtomic(previewCachePath(t, svc, "/ok.jpg"), brokenPreview); err != nil {
		t.Fatalf("写入损坏 preview 失败: %v", err)
	}

	_, file, err := svc.GetThumb(ctx, "/ok.jpg", 300, thumbnail.PriorityHigh)
	if err != nil {
		t.Fatalf("派生失败应回退原图生成: %v", err)
	}
	file.Close()
}

// 负缓存：生成失败后 TTL 内同键快速失败（返回记录的同一错误实例，不再触发生成），
// TTL 过期后允许重新生成（产生新错误实例）。TTL 注入短值便于测试。
func TestNegativeCacheFastFailAndRetry(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	svc.negTTL = 30 * time.Millisecond
	writeBrokenImage(t, imageRoot, "broken.jpg")

	_, _, err1 := svc.GetThumb(ctx, "/broken.jpg", 300, thumbnail.PriorityHigh)
	if err1 == nil {
		t.Fatal("损坏图应生成失败")
	}

	// TTL 内：命中负缓存，返回记录的同一错误实例（而非重新生成的新实例）
	_, _, err2 := svc.GetThumb(ctx, "/broken.jpg", 300, thumbnail.PriorityHigh)
	if err2 == nil {
		t.Fatal("TTL 内应命中负缓存快速失败")
	}
	if err1 != err2 {
		t.Fatalf("TTL 内应返回负缓存记录的同一错误实例")
	}

	// TTL 过期：负缓存失效，重新生成并得到新错误实例
	time.Sleep(60 * time.Millisecond)
	_, _, err3 := svc.GetThumb(ctx, "/broken.jpg", 300, thumbnail.PriorityHigh)
	if err3 == nil {
		t.Fatal("TTL 过期后应重新生成并仍失败")
	}
	if err3 == err1 {
		t.Fatal("TTL 过期后应重新生成（错误实例应更新）")
	}
}

// 取消类错误不得记入负缓存：取消不代表图坏，后续同键请求仍应正常尝试生成。
func TestNegativeCacheSkipsCancellation(t *testing.T) {
	svc, _ := newTestThumbnailEnv(t)
	const key = "neg-key"

	svc.recordNegative(key, context.Canceled)
	if err := svc.checkNegative(key); err != nil {
		t.Fatalf("context.Canceled 不应记入负缓存: %v", err)
	}
	svc.recordNegative(key, fmt.Errorf("wrap: %w", context.DeadlineExceeded))
	if err := svc.checkNegative(key); err != nil {
		t.Fatalf("context.DeadlineExceeded 不应记入负缓存: %v", err)
	}
}

// 索引登记在生成槽外异步补记：产物返回即可用，索引行随后可见。
func TestIndexRecordedAfterSlotRelease(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeTestJPEG(t, imageRoot, "ok.jpg", 800, 600)

	_, file, err := svc.GetThumb(ctx, "/ok.jpg", 300, thumbnail.PriorityHigh)
	if err != nil {
		t.Fatalf("生成 thumb 应成功: %v", err)
	}
	file.Close()

	deadline := time.Now().Add(2 * time.Second)
	for {
		items, err := svc.repo.List(ctx)
		if err != nil {
			t.Fatalf("查询索引失败: %v", err)
		}
		if len(items) == 1 && items[0].SourcePath == "/ok.jpg" && items[0].Variant == model.VariantThumb {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("索引未在超时前补记: %+v", items)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// thumbCachePath 以源图当前 mtime/size 构造 thumb(300) 缓存键路径（与派生判定同参）。
func thumbCachePath(t *testing.T, svc *ThumbnailService, sourcePath string) string {
	t.Helper()
	info, err := svc.fs.GetMetadata(sourcePath)
	if err != nil {
		t.Fatalf("读取源图元信息失败: %v", err)
	}
	thumbPath, err := thumbnail.CachePath(svc.dataDir, sourcePath, model.VariantThumb,
		300, info.ModifiedAt.UnixNano(), info.Size)
	if err != nil {
		t.Fatalf("构造 thumb 缓存路径失败: %v", err)
	}
	return thumbPath
}

// 级联：GetThumb 从原图生成成功后，异步低优先级趁热补 preview（轮询等待落盘）。
func TestCascadePreviewAfterThumbGeneration(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeTestJPEG(t, imageRoot, "ok.jpg", 800, 600)

	_, file, err := svc.GetThumb(ctx, "/ok.jpg", 300, thumbnail.PriorityHigh)
	if err != nil {
		t.Fatalf("生成 thumb 应成功: %v", err)
	}
	file.Close()

	previewPath := previewCachePath(t, svc, "/ok.jpg")
	deadline := time.Now().Add(3 * time.Second)
	for {
		if thumbnail.IsFresh(previewPath) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("级联 preview 未在超时内生成")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// 前端低优流量打点透传：NoteFrontendWarmTraffic 后 FrontendWarmIdleFor 立即接近零，
// 且不影响高优先级活跃判定——后台预热器据此区分"用户在浏览"与"前端预热进行中"。
func TestFrontendWarmTrafficPassthrough(t *testing.T) {
	svc, _ := newTestThumbnailEnv(t)

	if svc.FrontendWarmIdleFor() < time.Hour {
		t.Fatal("从未打点应视为一直空闲（返回最大时长）")
	}
	svc.NoteFrontendWarmTraffic()
	if idle := svc.FrontendWarmIdleFor(); idle > time.Second {
		t.Fatalf("打点后空闲时长应接近零: %v", idle)
	}
	if svc.FrontendIdleFor() < time.Hour {
		t.Fatal("低优打点不应影响高优先级活跃判定")
	}
}

// WarmThumb：缺失时生成并返回 true；已缓存时返回 false 且不再生成；非法路径报错。
func TestWarmThumbIdempotent(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeTestJPEG(t, imageRoot, "ok.jpg", 800, 600)

	generated, err := svc.WarmThumb(ctx, "/ok.jpg")
	if err != nil || !generated {
		t.Fatalf("首次预热应生成: generated=%v err=%v", generated, err)
	}
	generated, err = svc.WarmThumb(ctx, "/ok.jpg")
	if err != nil || generated {
		t.Fatalf("已缓存应返回 false,nil: generated=%v err=%v", generated, err)
	}
	if _, err := svc.WarmThumb(ctx, "/nope.jpg"); err == nil {
		t.Fatal("不存在路径应返回错误")
	}
}

// newTestSettingsRepo 基于 svc 的 dataDir 打开独立 DB 连接构造 app_settings 仓储。
func newTestSettingsRepo(t *testing.T, svc *ThumbnailService) *repository.AppSettingsRepository {
	t.Helper()
	db, err := repository.Open(svc.dataDir)
	if err != nil {
		t.Fatalf("打开设置数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close(db) })
	return repository.NewAppSettingsRepository(db)
}

// 预热目录顺序：按目录修改时间降序（新目录优先）；目录内文件仍按文件名字典序。
func TestCacheWarmerCollectOrderNewestFirst(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeTestJPEG(t, imageRoot, "old/1.jpg", 100, 100)
	writeTestJPEG(t, imageRoot, "new/1.jpg", 100, 100)
	writeTestJPEG(t, imageRoot, "new/2.jpg", 100, 100)

	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(imageRoot, "old"), past, past); err != nil {
		t.Fatalf("回拨 old 目录 mtime 失败: %v", err)
	}

	w := NewCacheWarmer(svc.fs, svc, newTestSettingsRepo(t, svc), time.Hour, model.WarmLevelMinimal, true)
	dirs := w.collect()
	if len(dirs) != 2 || dirs[0].rel != "/new" || dirs[1].rel != "/old" {
		t.Fatalf("应按目录修改时间降序（new 先于 old）: %+v", dirs)
	}
	if len(dirs[0].files) != 2 || dirs[0].files[0] != "/new/1.jpg" {
		t.Fatalf("目录内文件应按文件名字典序: %+v", dirs[0].files)
	}
}

// 预热器级别范围：minimal（thumb 20 / preview 5）下 6 张图目录 →
// 6 张 thumb 全部预热、preview 仅文件名序前 5 张；进度字段与状态语义正确。
func TestCacheWarmerPassRespectsLevel(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	for i := 1; i <= 6; i++ {
		writeTestJPEG(t, imageRoot, fmt.Sprintf("a/%02d.jpg", i), 800, 600)
	}
	writeTestJPEG(t, imageRoot, ".hidden/3.jpg", 800, 600)

	w := NewCacheWarmer(svc.fs, svc, newTestSettingsRepo(t, svc), time.Hour, model.WarmLevelMinimal, true)
	w.pass(ctx)
	w.pass(ctx) // 二轮幂等（全命中缓存）

	for i := 1; i <= 6; i++ {
		rel := fmt.Sprintf("/a/%02d.jpg", i)
		if !thumbnail.IsFresh(thumbCachePath(t, svc, rel)) {
			t.Fatalf("%s 的 thumb 应被预热", rel)
		}
		if i <= 5 {
			if !thumbnail.IsFresh(previewCachePath(t, svc, rel)) {
				t.Fatalf("%s 的 preview 应被预热（minimal 前 5 张）", rel)
			}
		} else if thumbnail.IsFresh(previewCachePath(t, svc, "/a/06.jpg")) {
			t.Fatal("第 6 张 preview 不应被预热（minimal 仅前 5 张）")
		}
	}
	if thumbnail.IsFresh(thumbCachePath(t, svc, ".hidden/3.jpg")) {
		t.Fatal("隐藏目录不应被预热")
	}

	st := w.Status()
	if st.DirsTotal != 1 || st.DirsDone != 1 {
		t.Fatalf("目录进度应 1/1: %+v", st)
	}
	if st.Phase != "idle" || st.LastPassEnd == 0 {
		t.Fatalf("完成后应 idle 且记录完成时间: %+v", st)
	}
	if st.ImagesPlanned != 6+5 || st.ImagesDone != 11 {
		t.Fatalf("图片计划/完成数应 11/11: %+v", st)
	}
}

// 级别持久化：SetLevel 写入 app_settings，LoadWarmLevel 读回；非法值回退默认档。
func TestCacheWarmerSetLevelPersists(t *testing.T) {
	svc, _ := newTestThumbnailEnv(t)
	settings := newTestSettingsRepo(t, svc)

	if got := LoadWarmLevel(ctx, settings); got != model.WarmLevelDefault {
		t.Fatalf("未设置时应回退默认档 %s: %s", model.WarmLevelDefault, got)
	}

	w := NewCacheWarmer(svc.fs, svc, settings, time.Hour, model.WarmLevelDefault, true)
	if err := w.SetLevel(ctx, model.WarmLevel1); err != nil {
		t.Fatalf("SetLevel 失败: %v", err)
	}
	if w.Level() != model.WarmLevel1 {
		t.Fatalf("级别应即时生效: %s", w.Level())
	}
	if got := LoadWarmLevel(ctx, settings); got != model.WarmLevel1 {
		t.Fatalf("级别应持久化: %s", got)
	}

	// 非法持久化值（外部写库）→ LoadWarmLevel 回退默认
	if err := settings.Set(ctx, warmLevelKey, "bogus"); err != nil {
		t.Fatalf("写入非法级别失败: %v", err)
	}
	if got := LoadWarmLevel(ctx, settings); got != model.WarmLevelDefault {
		t.Fatalf("非法值应回退默认档: %s", got)
	}
}

// WarmPreview：缺失时生成返回 true；已缓存返回 false；off 级别预热器空转不生成。
func TestWarmPreviewAndOffLevel(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeTestJPEG(t, imageRoot, "ok.jpg", 800, 600)

	generated, err := svc.WarmPreview(ctx, "/ok.jpg")
	if err != nil || !generated {
		t.Fatalf("首次预热 preview 应生成: generated=%v err=%v", generated, err)
	}
	generated, err = svc.WarmPreview(ctx, "/ok.jpg")
	if err != nil || generated {
		t.Fatalf("已缓存应返回 false,nil: generated=%v err=%v", generated, err)
	}

	w := NewCacheWarmer(svc.fs, svc, newTestSettingsRepo(t, svc), time.Hour, model.WarmLevelOff, true)
	w.pass(ctx) // off：空转
	if st := w.Status(); st.Phase != "idle" {
		t.Fatalf("off 级别应 idle: %+v", st)
	}
}
