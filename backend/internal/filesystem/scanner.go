package filesystem

import (
	"os"
	"path/filepath"
	"strings"

	"albumshelf/backend/internal/model"
)

// imageExts 支持的图片扩展名（统一小写比较）。
var imageExts = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".webp": {},
	".gif":  {},
}

// scanDir 单层遍历目录：子目录收集为 Folder，图片扩展名文件收集为 Image。
// 跳过以 . 开头的隐藏项；不排序、不碰数据库。
func scanDir(absDir, root string) ([]model.Folder, []model.Image, error) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, nil, err
	}

	folders := make([]model.Folder, 0, len(entries))
	images := make([]model.Image, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		if e.IsDir() {
			folders = append(folders, model.Folder{
				Name: name,
				Path: relPathStyle(root, filepath.Join(absDir, name)),
			})
			continue
		}

		if !isImageExt(filepath.Ext(name)) {
			continue
		}
		img, err := readImageMetadata(filepath.Join(absDir, name), root)
		if err != nil {
			return nil, nil, err
		}
		images = append(images, img)
	}
	return folders, images, nil
}

// isImageExt 判断扩展名是否为支持的图片格式（大小写不敏感）。
func isImageExt(ext string) bool {
	_, ok := imageExts[strings.ToLower(ext)]
	return ok
}
