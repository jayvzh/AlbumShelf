package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/repository"
)

// newTestFavoriteEnv 构造「临时图片目录 + 临时数据库」的 FavoriteService。
// 假 .jpg 内容非图片：GetMetadata 仅文件头解析失败回退 nil 尺寸，不影响收藏语义。
func newTestFavoriteEnv(t *testing.T) (*FavoriteService, *filesystem.LocalFilesystem) {
	t.Helper()

	root := t.TempDir()
	for _, name := range []string{"001.jpg", "002.jpg"} {
		if err := os.WriteFile(filepath.Join(root, name), make([]byte, 16), 0o644); err != nil {
			t.Fatalf("写入 %s 失败: %v", name, err)
		}
	}

	db, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("打开临时数据库失败: %v", err)
	}
	t.Cleanup(func() { repository.Close(db) })

	fs := filesystem.NewLocalFilesystem(root)
	return NewFavoriteService(fs, repository.NewFavoriteRepository(db)), fs
}

// Toggle 新增 → 再 Toggle 取消。
func TestFavoriteToggleRoundTrip(t *testing.T) {
	svc, _ := newTestFavoriteEnv(t)

	favorited, err := svc.Toggle("/001.jpg")
	if err != nil || !favorited {
		t.Fatalf("首次 Toggle 期望 (true, nil)，实际 (%v, %v)", favorited, err)
	}

	favorited, err = svc.Toggle("/001.jpg")
	if err != nil || favorited {
		t.Fatalf("二次 Toggle 期望 (false, nil)，实际 (%v, %v)", favorited, err)
	}
}

// Toggle 不存在的文件 → 404 语义（filesystem.ErrNotFound 透传）。
func TestFavoriteToggleMissingFile(t *testing.T) {
	svc, _ := newTestFavoriteEnv(t)

	if _, err := svc.Toggle("/Missing.jpg"); !errors.Is(err, filesystem.ErrNotFound) {
		t.Fatalf("期望 ErrNotFound 透传，实际 %v", err)
	}
}

// Toggle 非法路径（目录穿越）→ filesystem.ErrInvalidPath。
func TestFavoriteToggleInvalidPath(t *testing.T) {
	svc, _ := newTestFavoriteEnv(t)

	if _, err := svc.Toggle("/../../etc/passwd"); !errors.Is(err, filesystem.ErrInvalidPath) {
		t.Fatalf("期望 ErrInvalidPath，实际 %v", err)
	}
}

// List 惰性清理：文件删除后 List 自动过滤失效条目并从库中移除。
func TestFavoriteListLazilyCleansStale(t *testing.T) {
	svc, fs := newTestFavoriteEnv(t)

	for _, p := range []string{"/001.jpg", "/002.jpg"} {
		if _, err := svc.Toggle(p); err != nil {
			t.Fatalf("Toggle %s 失败: %v", p, err)
		}
	}

	// 删除 002.jpg 对应的真实文件（root/002.jpg）
	if err := os.Remove(fs.Root() + "/002.jpg"); err != nil {
		t.Fatalf("删除测试文件失败: %v", err)
	}

	images, err := svc.List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(images) != 1 || images[0].Path != "/001.jpg" {
		t.Fatalf("失效条目应被过滤，实际 %+v", images)
	}

	// 重建文件后再次 Toggle：若惰性清理已落库，此次应视为"新增收藏"
	if err := os.WriteFile(fs.Root()+"/002.jpg", make([]byte, 16), 0o644); err != nil {
		t.Fatalf("重建测试文件失败: %v", err)
	}
	favorited, err := svc.Toggle("/002.jpg")
	if err != nil || !favorited {
		t.Fatalf("清理后 Toggle 应为新增 (true, nil)，实际 (%v, %v)", favorited, err)
	}
}

// List 倒序：后收藏的在前。
func TestFavoriteListOrderNewestFirst(t *testing.T) {
	svc, _ := newTestFavoriteEnv(t)

	if _, err := svc.Toggle("/001.jpg"); err != nil {
		t.Fatalf("Toggle /001.jpg 失败: %v", err)
	}
	if _, err := svc.Toggle("/002.jpg"); err != nil {
		t.Fatalf("Toggle /002.jpg 失败: %v", err)
	}

	images, err := svc.List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(images) != 2 || images[0].Path != "/002.jpg" || images[1].Path != "/001.jpg" {
		t.Fatalf("期望后收藏在前: %+v", images)
	}
}
