package thumbnail

import "os"

// IsFresh 判断缓存文件是否可用（DATA_MODEL §3 判定规则的文件系统侧等价实现）：
// 缓存键已编码 path|mtime|size，原图变化 → 键变 → 请求落到新路径，
// 因此"该路径下文件存在"即意味着它由当前原图生成过，无需再比对记录，
// 文件存在（且不是目录）即新鲜；不存在（键首次出现或文件丢失）即失效待重建。
func IsFresh(cachePath string) bool {
	info, err := os.Stat(cachePath)
	return err == nil && !info.IsDir()
}
