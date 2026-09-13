package thumbnail

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"albumshelf/backend/internal/model"
)

// writeTestJPEG / writeTestPNG 用标准库在临时目录生成指定尺寸的纯色源图。
func writeTestJPEG(t *testing.T, dir string, w, h int) string {
	t.Helper()
	return writeTestImage(t, dir, "src.jpg", w, h, imageTypeJPEG)
}

func writeTestPNG(t *testing.T, dir string, w, h int) string {
	t.Helper()
	return writeTestImage(t, dir, "src.png", w, h, imageTypePNG)
}

const (
	imageTypeJPEG = iota
	imageTypePNG
)

func writeTestImage(t *testing.T, dir, name string, w, h int, typ int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// 渐变色填充，避免极端压缩退化
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("创建源图失败: %v", err)
	}
	defer f.Close()
	switch typ {
	case imageTypeJPEG:
		err = jpeg.Encode(f, img, nil)
	case imageTypePNG:
		err = png.Encode(f, img)
	}
	if err != nil {
		t.Fatalf("编码源图失败: %v", err)
	}
	return p
}

// isJPEG 检查 JPEG 魔数 FFD8。
func isJPEG(data []byte) bool {
	return len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8
}

// 800×600 JPEG → thumb(300)：输出 JPEG、宽高 ≤300 且保持 4:3 比例。
func TestGenerateThumbFitBox(t *testing.T) {
	src := writeTestJPEG(t, t.TempDir(), 800, 600)
	data, w, h, err := Generate(src, model.VariantThumb, 300)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if !isJPEG(data) {
		t.Fatal("输出不是 JPEG（缺少 FFD8 魔数）")
	}
	if w > 300 || h > 300 {
		t.Fatalf("输出超出 300×300 盒: %dx%d", w, h)
	}
	if w*3 != h*4 {
		t.Fatalf("比例未保持 4:3: %dx%d", w, h)
	}
}

// 小图 200×100 → thumb(300)：不放大，输出保持 200×100。
func TestGenerateNoUpscale(t *testing.T) {
	src := writeTestJPEG(t, t.TempDir(), 200, 100)
	data, w, h, err := Generate(src, model.VariantThumb, 300)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if !isJPEG(data) {
		t.Fatal("输出不是 JPEG（缺少 FFD8 魔数）")
	}
	if w != 200 || h != 100 {
		t.Fatalf("小图被放大: %dx%d，期望 200x100", w, h)
	}
}

// 3000×2000 PNG → preview(2560)：输出 JPEG 且长边 ≤2560、比例保持 3:2。
func TestGeneratePreviewLongEdge(t *testing.T) {
	src := writeTestPNG(t, t.TempDir(), 3000, 2000)
	data, w, h, err := Generate(src, model.VariantPreview, 0)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if !isJPEG(data) {
		t.Fatal("输出不是 JPEG（缺少 FFD8 魔数）")
	}
	if w > model.PreviewLongEdge || h > model.PreviewLongEdge {
		t.Fatalf("长边超出 %d: %dx%d", model.PreviewLongEdge, w, h)
	}
	if got := float64(w) / float64(h); math.Abs(got-1.5) > 0.02 {
		t.Fatalf("比例未保持 3:2: %dx%d", w, h)
	}
}

// 非法桶与未知变体必须报错。
func TestGenerateInvalidArgs(t *testing.T) {
	if _, _, _, err := Generate("x.jpg", model.VariantThumb, 250); err == nil {
		t.Fatal("非法桶 250 应返回错误")
	}
	if _, _, _, err := Generate("x.jpg", "evil", 300); err == nil {
		t.Fatal("未知变体应返回错误")
	}
}
