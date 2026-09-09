package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/repository"
)

// newTestAppSettingsEnv 构造「含 /Private、/Private/sub、/PrivateX 的临时图片目录 + 临时数据库」。
func newTestAppSettingsEnv(t *testing.T) (*AppSettingsService, string) {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{"Private", "Private/sub", "PrivateX", "Public"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("创建目录 %s 失败: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "Private", "a.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "Private", "sub", "deep"), 0o755); err != nil {
		t.Fatalf("创建深层目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "Private", "sub", "deep", "x.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatalf("写入深层测试文件失败: %v", err)
	}

	db, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("打开临时数据库失败: %v", err)
	}
	t.Cleanup(func() { repository.Close(db) })

	fs := filesystem.NewLocalFilesystem(root)
	svc := NewAppSettingsService(fs, repository.NewProtectedFolderRepository(db))
	return svc, root
}

// 替换为私有目录：/Private 与 /Private/a.jpg 命中，/PrivateX 与 /Public 不命中。
func TestIsProtectedSubtreeMatching(t *testing.T) {
	svc, _ := newTestAppSettingsEnv(t)

	if _, err := svc.ReplacePaths([]string{"/Private"}); err != nil {
		t.Fatalf("设置私有目录失败: %v", err)
	}

	cases := []struct {
		path string
		want bool
	}{
		{"/Private", true},
		{"/Private/a.jpg", true},
		{"/Private/sub", true},
		{"/Private/sub/deep/x.jpg", true},
		{"/PrivateX", false}, // 前缀相同但非子树，不误伤
		{"/Public", false},
	}
	for _, c := range cases {
		if got := svc.IsProtectedRelPath(c.path); got != c.want {
			t.Fatalf("IsProtectedRelPath(%s) = %v, want %v", c.path, got, c.want)
		}
		if got := svc.IsProtectedPath(c.path); got != c.want {
			t.Fatalf("IsProtectedPath(%s) = %v, want %v", c.path, got, c.want)
		}
	}
}

// 根内子路径前缀边界：保护 /Private/sub 时父级 /Private 不命中。
func TestIsProtectedChildBoundary(t *testing.T) {
	svc, _ := newTestAppSettingsEnv(t)

	if _, err := svc.ReplacePaths([]string{"/Private/sub"}); err != nil {
		t.Fatalf("设置私有目录失败: %v", err)
	}
	if !svc.IsProtectedRelPath("/Private/sub") {
		t.Fatal("/Private/sub 应命中")
	}
	if !svc.IsProtectedRelPath("/Private/sub/inner.jpg") {
		t.Fatal("/Private/sub 下级应命中")
	}
	if svc.IsProtectedRelPath("/Private") {
		t.Fatal("/Private 自身不应被子路径保护命中")
	}
	if svc.IsProtectedRelPath("/Private/a.jpg") {
		t.Fatal("/Private/a.jpg 不应被子路径保护命中")
	}
}

// 空路径与根目录拒绝（INVALID_PATH）。
func TestReplacePathsRejectsEmptyAndRoot(t *testing.T) {
	svc, _ := newTestAppSettingsEnv(t)

	for _, bad := range [][]string{{""}, {"/"}, {"/Private", ""}} {
		if _, err := svc.ReplacePaths(bad); !errors.Is(err, filesystem.ErrInvalidPath) {
			t.Fatalf("ReplacePaths(%v) 应返回 ErrInvalidPath，got %v", bad, err)
		}
	}
}

// 不存在目录拒绝（ErrNotFound）；去重与规范化（末尾斜杠、大小写不处理）验证。
func TestReplacePathsValidatesAndNormalizes(t *testing.T) {
	svc, _ := newTestAppSettingsEnv(t)

	if _, err := svc.ReplacePaths([]string{"/NoSuch"}); !errors.Is(err, filesystem.ErrNotFound) {
		t.Fatalf("不存在目录应返回 ErrNotFound，got %v", err)
	}

	paths, err := svc.ReplacePaths([]string{"/Public/", "/Private", "/Private"})
	if err != nil {
		t.Fatalf("合法替换失败: %v", err)
	}
	want := []string{"/Private", "/Public"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("规范化结果 got=%v, want=%v", paths, want)
	}

	list, err := svc.ListPaths()
	if err != nil {
		t.Fatalf("ListPaths 失败: %v", err)
	}
	if !reflect.DeepEqual(list, want) {
		t.Fatalf("ListPaths got=%v, want=%v", list, want)
	}
}
