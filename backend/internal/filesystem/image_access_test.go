package filesystem

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeTestPNG 在 dir 下生成一张 2x1 的真实 PNG，返回文件路径。
func writeTestPNG(t *testing.T, dir, name string) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// newTestFilesystem 构造指向 root 的 LocalFilesystem 并在其中创建测试图片。
func newTestFilesystem(t *testing.T) (*LocalFilesystem, string) {
	t.Helper()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Comics"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestPNG(t, filepath.Join(root, "Comics"), "001.png")
	return NewLocalFilesystem(root), root
}

// TestOpenImageSuccess 验证合法图片 OpenImage 返回正确元信息与可读文件句柄。
func TestOpenImageSuccess(t *testing.T) {
	fs, root := newTestFilesystem(t)

	img, file, err := fs.OpenImage("/Comics/001.png")
	if err != nil {
		t.Fatalf("OpenImage unexpected err: %v", err)
	}
	defer file.Close()

	if img.Name != "001.png" || img.Path != "/Comics/001.png" {
		t.Fatalf("元信息 Name/Path = (%q, %q)", img.Name, img.Path)
	}
	if img.Extension != ".png" {
		t.Fatalf("Extension = %q, want .png", img.Extension)
	}
	if img.Width == nil || *img.Width != 2 || img.Height == nil || *img.Height != 1 {
		t.Fatalf("宽高 = (%v, %v), want (2, 1)", img.Width, img.Height)
	}

	info, err := os.Stat(filepath.Join(root, "Comics", "001.png"))
	if err != nil {
		t.Fatal(err)
	}
	if img.Size != info.Size() || !img.ModifiedAt.Equal(info.ModTime()) {
		t.Fatalf("Size/ModifiedAt 与磁盘不一致: (%d, %v)", img.Size, img.ModifiedAt)
	}

	// 句柄应可读：能读出非空内容
	buf := make([]byte, 4)
	n, err := file.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("文件句柄读取失败: n=%d err=%v", n, err)
	}
}

// TestOpenImageErrors 验证 OpenImage 的错误分支：越界 / 不存在 / 目录 / 非图片格式。
func TestOpenImageErrors(t *testing.T) {
	fs, root := newTestFilesystem(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{name: "路径越界", path: "../../etc/passwd", wantErr: ErrInvalidPath},
		{name: "不存在", path: "/Comics/nope.png", wantErr: ErrNotFound},
		{name: "目标是目录", path: "/Comics", wantErr: ErrNotFound},
		{name: "非图片扩展名", path: "/a.txt", wantErr: ErrUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, file, err := fs.OpenImage(tt.path)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("OpenImage(%q) err = %v, want %v", tt.path, err, tt.wantErr)
			}
			if file != nil {
				file.Close()
				t.Fatalf("OpenImage(%q) 失败时不应返回文件句柄", tt.path)
			}
			if img.Name != "" {
				t.Fatalf("OpenImage(%q) 失败时不应返回元信息", tt.path)
			}
		})
	}
}

// TestGetMetadataSuccess 验证 GetMetadata 不打开文件也能读到宽高。
func TestGetMetadataSuccess(t *testing.T) {
	fs, _ := newTestFilesystem(t)

	img, err := fs.GetMetadata("/Comics/001.png")
	if err != nil {
		t.Fatalf("GetMetadata unexpected err: %v", err)
	}
	if img.Path != "/Comics/001.png" {
		t.Fatalf("Path = %q, want /Comics/001.png", img.Path)
	}
	if img.Width == nil || *img.Width != 2 || img.Height == nil || *img.Height != 1 {
		t.Fatalf("宽高 = (%v, %v), want (2, 1)", img.Width, img.Height)
	}
}

// TestGetMetadataErrors 验证 GetMetadata 与 OpenImage 共用同一套校验错误。
func TestGetMetadataErrors(t *testing.T) {
	fs, root := newTestFilesystem(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{name: "路径越界", path: "../secret.png", wantErr: ErrInvalidPath},
		{name: "不存在", path: "/Comics/nope.png", wantErr: ErrNotFound},
		{name: "非图片扩展名", path: "/a.txt", wantErr: ErrUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fs.GetMetadata(tt.path)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetMetadata(%q) err = %v, want %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
