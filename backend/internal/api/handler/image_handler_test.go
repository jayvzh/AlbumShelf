package handler

import (
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
	"albumshelf/backend/internal/service"
	"albumshelf/backend/internal/thumbnail"
)

// newTestImageHandler 构造「临时图片目录 + 临时数据目录 + 临时数据库」的完整图片链路
//（参考 settings_handler_test.go 的组装方式），返回 handler 与图片根目录。
func newTestImageHandler(t *testing.T) (*ImageHandler, string) {
	t.Helper()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Comics"), 0o755); err != nil {
		t.Fatalf("创建 Comics 目录失败: %v", err)
	}
	db, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("打开临时数据库失败: %v", err)
	}
	t.Cleanup(func() { repository.Close(db) })

	fs := filesystem.NewLocalFilesystem(root)
	ts := service.NewThumbnailService(fs, t.TempDir(),
		repository.NewImageCacheRepository(db), thumbnail.NewScheduler(1))
	return NewImageHandler(service.NewImageService(fs, ts)), root
}

// writeValidJPEG 用标准库生成一张真实 JPEG 源图（参考 generator_test.go 的渐变填充），
// 返回虚拟路径（/Comics/…）。
func writeValidJPEG(t *testing.T, root, name string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	p := filepath.Join(root, "Comics", name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("创建源图失败: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatalf("编码源图失败: %v", err)
	}
	return "/Comics/" + name
}

// newImageTestGin 注册 image 路由。
func newImageTestGin(h *ImageHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/image", h.Get)
	return r
}

// GET image variant=preview：生成成功 → X-Image-Variant=preview，
// 响应保持 immutable 强缓存。
func TestGetImagePreviewImmutableCache(t *testing.T) {
	h, root := newTestImageHandler(t)
	path := writeValidJPEG(t, root, "ok.jpg", 800, 600)
	r := newImageTestGin(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/image?path="+path+"&variant=preview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Image-Variant"); got != model.VariantPreview {
		t.Fatalf("X-Image-Variant 期望 %s，实际 %s", model.VariantPreview, got)
	}
	const immutable = "public, max-age=31536000, immutable"
	if got := w.Header().Get("Cache-Control"); got != immutable {
		t.Fatalf("Cache-Control 期望 %s，实际 %s", immutable, got)
	}
	if w.Header().Get("ETag") == "" {
		t.Fatal("preview 响应缺少 ETag")
	}
}

// GET image variant=preview：源图损坏（合法扩展名、非法内容）生成失败 → 回退原图，
// X-Image-Variant=original，响应必须 no-cache + ETag，不得 immutable 长缓存
//（回归：回退响应曾被 immutable 缓存一年，导致该 URL 长期以原图体积加载）。
func TestGetImagePreviewFallbackNoCache(t *testing.T) {
	h, root := newTestImageHandler(t)
	p := filepath.Join(root, "Comics", "broken.jpg")
	if err := os.WriteFile(p, []byte("not a real jpeg"), 0o644); err != nil {
		t.Fatalf("写入坏图失败: %v", err)
	}
	r := newImageTestGin(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/image?path=/Comics/broken.jpg&variant=preview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（回退原图），实际 %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Image-Variant"); got != service.VariantOriginal {
		t.Fatalf("X-Image-Variant 期望 %s，实际 %s", service.VariantOriginal, got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("回退响应 Cache-Control 期望 no-cache，实际 %s", got)
	}
	if w.Header().Get("ETag") == "" {
		t.Fatal("回退响应缺少 ETag（无法再校验）")
	}
}
