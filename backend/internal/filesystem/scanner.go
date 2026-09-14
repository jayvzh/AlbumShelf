package filesystem

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

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
	var imgPaths []string
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
		imgPaths = append(imgPaths, filepath.Join(absDir, name))
	}

	return folders, readImagesConcurrently(imgPaths, root), nil
}

// readImagesConcurrently 有界 worker 池并发读取图片元信息（Stat + DecodeConfig 是扫描主耗时），
// 按 index 回填保持原顺序输出；单文件失败仅跳过并记日志，不拖垮整个目录列表。
func readImagesConcurrently(paths []string, root string) []model.Image {
	if len(paths) == 0 {
		return nil
	}

	images := make([]model.Image, len(paths))
	failed := make([]bool, len(paths))
	workers := min(runtime.NumCPU(), 8, len(paths))

	next := make(chan int)
	var wg sync.WaitGroup
	var failedCount atomic.Int64
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				img, err := readImageMetadata(paths[i], root)
				if err != nil {
					failed[i] = true
					failedCount.Add(1)
					log.Printf("[filesystem] 读取图片元信息失败，已跳过 %s: %v", paths[i], err)
					continue
				}
				images[i] = img
			}
		}()
	}
	for i := range paths {
		next <- i
	}
	close(next)
	wg.Wait()

	if skipped := failedCount.Load(); skipped > 0 {
		log.Printf("[filesystem] 目录 %s 共 %d 个图片元信息读取失败已跳过", root, skipped)
	}

	out := make([]model.Image, 0, len(images)-int(failedCount.Load()))
	for i, img := range images {
		if failed[i] {
			continue
		}
		out = append(out, img)
	}
	return out
}

// isImageExt 判断扩展名是否为支持的图片格式（大小写不敏感）。
func isImageExt(ext string) bool {
	_, ok := imageExts[strings.ToLower(ext)]
	return ok
}
