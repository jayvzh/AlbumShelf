package filesystem

import (
	"image"
	_ "image/gif"  // 注册 gif 解码器
	_ "image/jpeg" // 注册 jpeg 解码器
	_ "image/png"  // 注册 png 解码器
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "golang.org/x/image/webp" // 注册 webp 解码器（标准库无 webp）

	"albumshelf/backend/internal/model"
)

// readImageMetadata 读取单个图片文件元信息：Stat 得 Size/ModifiedAt/CreatedAt，
// DecodeConfig 只读文件头得像素尺寸；尺寸解析失败时 Width/Height 保持 nil 且不报错。
func readImageMetadata(absPath, root string) (model.Image, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return model.Image{}, err
	}

	img := model.Image{
		File: model.File{
			Name:       info.Name(),
			Path:       relPathStyle(root, absPath),
			Extension:  strings.ToLower(filepath.Ext(absPath)),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			CreatedAt:  fileCreatedAt(info),
		},
	}
	if w, h, err := decodeSize(absPath); err == nil {
		img.Width = &w
		img.Height = &h
	}
	return img, nil
}

// fileCreatedAt 取文件创建时间：Linux 下从 Stat_t.Ctim 读取；
// 类型断言失败时回退修改时间（排序用兜底，见 model.File.CreatedAt 注释）。
func fileCreatedAt(info os.FileInfo) time.Time {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Ctim.Sec, st.Ctim.Nsec)
	}
	return info.ModTime()
}

// decodeSize 只解析文件头获取宽高，禁止解码全图。
func decodeSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
