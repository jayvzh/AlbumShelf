package sorting

import (
	"sort"
	"strings"

	"albumshelf/backend/internal/model"
)

// filenameSorter 按文件名字典序排序（Go string 逐字节比较，大小写敏感）。
type filenameSorter struct{}

func (filenameSorter) Sort(images []model.Image, options model.SortOptions) []model.Image {
	sort.SliceStable(images, func(i, j int) bool {
		return lessByDir(strings.Compare(images[i].Name, images[j].Name), options.Direction)
	})
	return images
}
