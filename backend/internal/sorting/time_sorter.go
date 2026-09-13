package sorting

import (
	"sort"

	"albumshelf/backend/internal/model"
)

// timeSorter 同一 sorter 处理两种 mode（SORT_ENGINE §3）：
// modified_time 按 ModifiedAt、created_time 按 CreatedAt；旧→新为 asc。
type timeSorter struct{}

func (timeSorter) Sort(images []model.Image, options model.SortOptions) []model.Image {
	sort.SliceStable(images, func(i, j int) bool {
		cmp := images[i].ModifiedAt.Compare(images[j].ModifiedAt)
		if options.Mode == model.SortModeCreatedTime {
			cmp = images[i].CreatedAt.Compare(images[j].CreatedAt)
		}
		return lessByDir(cmp, options.Direction)
	})
	return images
}
