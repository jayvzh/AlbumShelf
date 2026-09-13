package thumbnail

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"albumshelf/backend/internal/model"
)

// 缓存键与失效语义（与 DATA_MODEL §3 判定规则一致）：
// 键 = sha1(sourcePath|mtime|size|variant|sizeBucket)，将原图 mtime 与 size 编码进文件名。
// 原图一旦被修改（mtime 或 size 任一变化）→ 键改变 → 请求落到全新路径 → 必然 miss → 重新生成；
// 旧文件成为孤儿，不影响正确性，由后续清理机制处理。

// cacheSubDir variant 到缓存子目录名的映射。
func cacheSubDir(variant string) (string, error) {
	switch variant {
	case model.VariantThumb:
		return "thumb", nil
	case model.VariantPreview:
		return "preview", nil
	default:
		return "", fmt.Errorf("未知缩略图变体: %s", variant)
	}
}

// CachePath 计算缓存文件完整路径 dataDir/cache/{variant}/{hash}.jpg 并确保子目录存在。
// 同输入恒定返回同一路径；原图 mtime/size 变化即返回不同路径（失效语义见文件顶部注释）。
func CachePath(dataDir, sourcePath, variant string, sizeBucket int, mtime int64, size int64) (string, error) {
	sub, err := cacheSubDir(variant)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d|%s|%d", sourcePath, mtime, size, variant, sizeBucket)))
	dir := filepath.Join(dataDir, "cache", sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建缓存目录失败: %w", err)
	}
	return filepath.Join(dir, hex.EncodeToString(sum[:])+".jpg"), nil
}

// WriteAtomic 原子写入缓存文件：先写同目录临时文件再 rename 覆盖，
// 避免并发请求读到半截 JPEG。
func WriteAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("替换缓存文件失败: %w", err)
	}
	return nil
}
