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

	_, file, err := svc.GetThumb(ctx, "/broken.jpg", 300)
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

	_, file, err := svc.GetThumb(ctx, "/ok.jpg", 300)
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

	_, file, err := svc.GetThumb(ctx, "/ok.jpg", 300)
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

	_, _, err1 := svc.GetThumb(ctx, "/broken.jpg", 300)
	if err1 == nil {
		t.Fatal("损坏图应生成失败")
	}

	// TTL 内：命中负缓存，返回记录的同一错误实例（而非重新生成的新实例）
	_, _, err2 := svc.GetThumb(ctx, "/broken.jpg", 300)
	if err2 == nil {
		t.Fatal("TTL 内应命中负缓存快速失败")
	}
	if err1 != err2 {
		t.Fatalf("TTL 内应返回负缓存记录的同一错误实例")
	}

	// TTL 过期：负缓存失效，重新生成并得到新错误实例
	time.Sleep(60 * time.Millisecond)
	_, _, err3 := svc.GetThumb(ctx, "/broken.jpg", 300)
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

	_, file, err := svc.GetThumb(ctx, "/ok.jpg", 300)
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

	_, file, err := svc.GetThumb(ctx, "/ok.jpg", 300)
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

// 预热器 pass：递归补齐可见目录 thumbs，跳过隐藏目录；重复执行幂等。
func TestThumbWarmerPass(t *testing.T) {
	svc, imageRoot := newTestThumbnailEnv(t)
	writeTestJPEG(t, imageRoot, "a/1.jpg", 800, 600)
	writeTestJPEG(t, imageRoot, "b/c/2.jpg", 800, 600)
	writeTestJPEG(t, imageRoot, ".hidden/3.jpg", 800, 600)

	w := NewThumbWarmer(svc.fs, svc, time.Hour)
	w.pass(ctx)
	w.pass(ctx) // 二轮幂等（全命中缓存）

	for _, rel := range []string{"/a/1.jpg", "/b/c/2.jpg"} {
		if !thumbnail.IsFresh(thumbCachePath(t, svc, rel)) {
			t.Fatalf("%s 的 thumb 应被预热", rel)
		}
	}
	if thumbnail.IsFresh(thumbCachePath(t, svc, ".hidden/3.jpg")) {
		t.Fatal("隐藏目录不应被预热")
	}
}
