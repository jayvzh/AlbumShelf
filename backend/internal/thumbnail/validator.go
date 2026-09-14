package thumbnail

import (
	"io"
	"os"
)

// IsFresh 判断缓存文件是否可用（DATA_MODEL §3 判定规则的文件系统侧等价实现）：
// 缓存键已编码 path|mtime|size，原图变化 → 键变 → 请求落到新路径，
// 因此"该路径下文件存在"即意味着它由当前原图生成过，无需再比对记录。
// 在存在性之上追加 JPEG EOI 尾标记校验：外因损坏/截断的文件判 miss，
// 走重新生成 → WriteAtomic 原子覆盖，自愈，不再持续返回坏图。
func IsFresh(cachePath string) bool {
	info, err := os.Stat(cachePath)
	if err != nil || info.IsDir() {
		return false
	}
	return hasJPEGEOI(cachePath)
}

// hasJPEGEOI 校验文件末尾两字节是否为 JPEG 码流结束标记（FF D9）。
// 文件不足 2 字节、打不开或读不出尾字节均视为不新鲜。
func hasJPEGEOI(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	if _, err := f.Seek(-2, io.SeekEnd); err != nil {
		return false
	}
	tail := make([]byte, 2)
	if _, err := io.ReadFull(f, tail); err != nil {
		return false
	}
	return tail[0] == 0xFF && tail[1] == 0xD9
}
