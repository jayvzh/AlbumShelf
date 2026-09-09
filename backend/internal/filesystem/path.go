package filesystem

import (
	"errors"
	"path/filepath"
	"strings"
)

// 路径安全 sentinel errors，由 handler 映射为 INVALID_PATH / FOLDER_NOT_FOUND / FILE_NOT_FOUND。
var (
	// ErrInvalidPath 路径非法或越出 IMAGE_ROOT（含符号链接逃逸）。
	ErrInvalidPath = errors.New("invalid path")
	// ErrNotFound 目标目录不存在。
	ErrNotFound = errors.New("not found")
	// ErrUnsupported 文件存在但不是支持的图片格式。
	ErrUnsupported = errors.New("unsupported image format")
)

// Resolve 将用户输入 path 解析为受限在 root 内的绝对路径。
//
// userPath 为相对 IMAGE_ROOT 的绝对风格路径（如 /Comics），不以 / 开头视为相对 root，
// 空串或 / 表示根目录。处理流程：拼到 root 下 Clean → 越界检查 → EvalSymlinks →
// 再次越界检查（防符号链接逃逸）。返回 absPath 为解析后的绝对路径，
// relPath 为 /Comics 风格（根为 /）。
func Resolve(root, userPath string) (absPath string, relPath string, err error) {
	root = canonicalRoot(root)

	// 归一：拼到 root 下再 Clean，保留 .. 语义以便越界检测
	joined := filepath.Clean(filepath.Join(root, userPath))
	if !isWithin(root, joined) {
		return "", "", ErrInvalidPath
	}

	resolved, err := filepath.EvalSymlinks(joined)
	if err != nil {
		return "", "", ErrNotFound
	}
	if !isWithin(root, resolved) {
		return "", "", ErrInvalidPath
	}
	return resolved, relPathStyle(root, resolved), nil
}

// ValidatePath 仅做路径安全校验（Clean + 越界检查），不要求目标存在。
// 用于收藏配置导入等文件可能暂缺的场景；运行期访问仍走 Resolve 全量校验。
// 返回 /Comics 风格相对路径。
func ValidatePath(root, userPath string) (string, error) {
	root = canonicalRoot(root)
	joined := filepath.Clean(filepath.Join(root, userPath))
	if !isWithin(root, joined) {
		return "", ErrInvalidPath
	}
	return relPathStyle(root, joined), nil
}

// canonicalRoot 将 root 解析为绝对路径并尽量消除符号链接，失败时退化为 Clean 后的绝对路径。
func canonicalRoot(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return filepath.Clean(abs)
}

// isWithin 判断 target 是否等于 root 或位于 root 之下；
// 用 Rel 判断，避免 /images2 误判为 /images 的子路径。
func isWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// relPathStyle 把 root 内的绝对路径转为 /Comics 风格；根目录为 /。
func relPathStyle(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == "." {
		return "/"
	}
	return "/" + rel
}
