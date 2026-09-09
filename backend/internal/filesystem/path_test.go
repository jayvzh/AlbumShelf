package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestResolve 表驱动验证路径解析：合法路径、根、穿越攻击与不存在路径。
func TestResolve(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Comics"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantRel string
		wantErr error
	}{
		{name: "合法子路径", path: "/Comics", wantRel: "/Comics"},
		{name: "根-空串", path: "", wantRel: "/"},
		{name: "根-斜杠", path: "/", wantRel: "/"},
		{name: "无前导斜杠视为相对root", path: "Comics", wantRel: "/Comics"},
		{name: "穿越攻击-多级上跳", path: "../../etc/passwd", wantErr: ErrInvalidPath},
		{name: "穿越攻击-混合下钻上跳", path: "/Comics/../../etc", wantErr: ErrInvalidPath},
		{name: "不存在子目录", path: "/Nope", wantErr: ErrNotFound},
		{name: "绝对路径被拼到root下不存在", path: "/etc/passwd", wantErr: ErrNotFound},
		{name: "root外已存在目录拼到root下", path: "/tmp", wantErr: ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs, rel, err := Resolve(root, tt.path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Resolve(%q) err = %v, want %v", tt.path, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve(%q) unexpected err: %v", tt.path, err)
			}
			if rel != tt.wantRel {
				t.Fatalf("Resolve(%q) rel = %q, want %q", tt.path, rel, tt.wantRel)
			}
			if !isWithin(root, abs) {
				t.Fatalf("Resolve(%q) abs = %q 越出 root %q", tt.path, abs, root)
			}
		})
	}
}

// TestResolveSymlink 验证符号链接：指向 root 内允许，逃逸到 root 外拒绝。
func TestResolveSymlink(t *testing.T) {
	root := t.TempDir()
	comics := filepath.Join(root, "Comics")
	if err := os.Mkdir(comics, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()

	// alias 指向 root 内目录，escape 指向 root 外目录
	if err := os.Symlink(comics, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}

	t.Run("指向root内合法目标", func(t *testing.T) {
		abs, rel, err := Resolve(root, "/alias")
		if err != nil {
			t.Fatalf("Resolve(/alias) unexpected err: %v", err)
		}
		if abs != comics || rel != "/Comics" {
			t.Fatalf("Resolve(/alias) = (%q, %q), want (%q, /Comics)", abs, rel, comics)
		}
	})

	t.Run("逃逸到root外", func(t *testing.T) {
		if _, _, err := Resolve(root, "/escape"); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("Resolve(/escape) err = %v, want ErrInvalidPath", err)
		}
	})
}
