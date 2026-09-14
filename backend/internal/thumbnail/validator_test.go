package thumbnail

import (
	"os"
	"path/filepath"
	"testing"
)

// 缓存文件不存在 → 不新鲜。
func TestIsFreshNotExist(t *testing.T) {
	if IsFresh(filepath.Join(t.TempDir(), "missing.jpg")) {
		t.Fatal("不存在的文件应判定为不新鲜")
	}
}

// 缓存文件存在且以 JPEG EOI 标记结尾 → 新鲜。
func TestIsFreshExist(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cached.jpg")
	// SOI + EOI：最小的完整 JPEG 码流形态
	if err := os.WriteFile(p, []byte{0xFF, 0xD8, 0xFF, 0xD9}, 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	if !IsFresh(p) {
		t.Fatal("以 EOI 结尾的文件应判定为新鲜")
	}
}

// 缓存文件被截断（无 EOI 尾标记）→ 不新鲜，触发重新生成自愈。
func TestIsFreshTruncated(t *testing.T) {
	p := filepath.Join(t.TempDir(), "truncated.jpg")
	if err := os.WriteFile(p, []byte{0xFF, 0xD8}, 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	if IsFresh(p) {
		t.Fatal("缺少 EOI 尾标记的文件应判定为不新鲜")
	}
}

// 路径是目录而非文件 → 不新鲜。
func TestIsFreshDirectory(t *testing.T) {
	dir := t.TempDir()
	if IsFresh(dir) {
		t.Fatal("目录应判定为不新鲜")
	}
}
