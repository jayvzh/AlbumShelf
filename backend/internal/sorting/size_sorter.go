package sorting

import (
	"sort"

	"imageshelf/backend/internal/model"
)

// sizeSorter 按文件大小排序，小→大为 asc。
type sizeSorter struct{}

func (sizeSorter) Sort(images []model.Image, options model.SortOptions) []model.Image {
	sort.SliceStable(images, func(i, j int) bool {
		return lessByDir(compareInt64(images[i].Size, images[j].Size), options.Direction)
	})
	return images
}

// compareInt64 三态比较两个 int64。
func compareInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
