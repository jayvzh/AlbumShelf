package sorting

import (
	"sort"
	"strings"

	"albumshelf/backend/internal/model"
)

// NaturalCompare 自然排序比较：把名字切为文本段/数字段序列逐段比较，
// 数字段按数值比较（去前导零后先比位数再逐位），使 Chapter2 < Chapter10。
// 大小写不敏感（SORT_ENGINE §4 sensitivity: base）；整体等值返回 0，
// 由稳定排序保证同序名相对顺序确定。
func NaturalCompare(a, b string) int {
	a, b = strings.ToLower(a), strings.ToLower(b)
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isDigit(a[i]) && isDigit(b[j]) {
			// 数值比较：去前导零后先比位数（长度），再逐位比较
			ea, eb := i, j
			for ea < len(a) && isDigit(a[ea]) {
				ea++
			}
			for eb < len(b) && isDigit(b[eb]) {
				eb++
			}
			na := strings.TrimLeft(a[i:ea], "0")
			nb := strings.TrimLeft(b[j:eb], "0")
			if len(na) != len(nb) {
				if len(na) < len(nb) {
					return -1
				}
				return 1
			}
			if c := strings.Compare(na, nb); c != 0 {
				return c
			}
			i, j = ea, eb
			continue
		}
		if a[i] != b[j] {
			if a[i] < b[j] {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	switch {
	case i < len(a):
		return 1 // a 更长（剩余部分）
	case j < len(b):
		return -1
	default:
		return 0
	}
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// naturalSorter 自然排序：名字中数字段按数值比较。
type naturalSorter struct{}

func (naturalSorter) Sort(images []model.Image, options model.SortOptions) []model.Image {
	sort.SliceStable(images, func(i, j int) bool {
		return lessByDir(NaturalCompare(images[i].Name, images[j].Name), options.Direction)
	})
	return images
}
