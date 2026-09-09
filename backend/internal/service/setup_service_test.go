package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// mustChmod 调整权限，失败直接终止测试（仅 POSIX 测试环境使用）。
func mustChmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("修改权限 %s 失败: %v", path, err)
	}
}

// TestSetupStatusReadableDir 存在的可读目录：全 true，initialized 为 true（含空目录）。
func TestSetupStatusReadableDir(t *testing.T) {
	root := t.TempDir() // 空目录同样必须判定可读（Readdirnames 返回 io.EOF）
	svc := NewSetupService(root, "secret")

	s := svc.Status(context.Background())
	if !s.ImageRootConfigured || !s.ImageRootExists || !s.ImageRootReadable {
		t.Fatalf("可读目录三项检测应全为 true: %+v", s)
	}
	if !s.AuthEnabled {
		t.Fatalf("authPassword 非空时 AuthEnabled 应为 true: %+v", s)
	}
	if !s.Initialized {
		t.Fatalf("前三项全真时 Initialized 应为 true: %+v", s)
	}
}

// TestSetupStatusMissingDir 不存在的路径：exists/readable false，initialized false。
func TestSetupStatusMissingDir(t *testing.T) {
	svc := NewSetupService(filepath.Join(t.TempDir(), "not-exist"), "")

	s := svc.Status(context.Background())
	if !s.ImageRootConfigured {
		t.Fatalf("路径非空时 Configured 应为 true: %+v", s)
	}
	if s.ImageRootExists || s.ImageRootReadable {
		t.Fatalf("不存在路径 Exists/Readable 应为 false: %+v", s)
	}
	if s.AuthEnabled {
		t.Fatalf("authPassword 为空时 AuthEnabled 应为 false: %+v", s)
	}
	if s.Initialized {
		t.Fatalf("Initialized 应为 false: %+v", s)
	}
}

// TestSetupStatusFileAsRoot 指向普通文件：Exists（要求目录）为 false。
func TestSetupStatusFileAsRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "plain.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	svc := NewSetupService(file, "")

	s := svc.Status(context.Background())
	if s.ImageRootExists {
		t.Fatalf("指向普通文件时 Exists 应为 false: %+v", s)
	}
	if s.ImageRootReadable {
		t.Fatalf("指向普通文件时 Readable 应为 false: %+v", s)
	}
	if s.Initialized {
		t.Fatalf("Initialized 应为 false: %+v", s)
	}
}

// TestSetupStatusEmptyImageRoot 空 IMAGE_ROOT：configured false，其余联动 false。
func TestSetupStatusEmptyImageRoot(t *testing.T) {
	svc := NewSetupService("", "secret")

	s := svc.Status(context.Background())
	if s.ImageRootConfigured {
		t.Fatalf("空 IMAGE_ROOT 时 Configured 应为 false: %+v", s)
	}
	if s.ImageRootExists || s.ImageRootReadable {
		t.Fatalf("未配置时 Exists/Readable 应为 false: %+v", s)
	}
	if !s.AuthEnabled {
		t.Fatalf("authPassword 非空时 AuthEnabled 应为 true: %+v", s)
	}
	if s.Initialized {
		t.Fatalf("Initialized 应为 false: %+v", s)
	}
}

// TestSetupStatusInitializedCombination initialized 仅在前三项全真时为 true。
func TestSetupStatusInitializedCombination(t *testing.T) {
	root := t.TempDir()

	// 目录存在但不可读（去掉读权限）：前三项缺 readable → false。
	// 以非 root 运行时有效；root 用户绕过权限位，跳过该分支。
	if os.Getuid() != 0 {
		mustChmod(t, root, 0o300) // r-- 位全撤，保留遍历能力便于 Stat
		svc := NewSetupService(root, "")
		if s := svc.Status(context.Background()); s.Initialized {
			mustChmod(t, root, 0o755)
			t.Fatalf("目录不可读时 Initialized 应为 false: %+v", s)
		}
		mustChmod(t, root, 0o755)
	}

	// 恢复可读但 auth 关闭：游客模式同样判定初始化就绪。
	svc := NewSetupService(root, "")
	if s := svc.Status(context.Background()); !s.Initialized || s.AuthEnabled {
		t.Fatalf("游客模式下可读目录应 initialized=true / auth_enabled=false: %+v", s)
	}
}
