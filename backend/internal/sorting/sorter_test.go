package sorting

import (
	"reflect"
	"testing"
	"time"

	"albumshelf/backend/internal/model"
)

func mkImages(names []string) []model.Image {
	images := make([]model.Image, len(names))
	for i, n := range names {
		images[i] = model.Image{File: model.File{Name: n}}
	}
	return images
}

func nameList(images []model.Image) []string {
	names := make([]string, len(images))
	for i, img := range images {
		names[i] = img.Name
	}
	return names
}

// sortNames 每次用全新切片排序，避免用例间相互影响。
func sortNames(t *testing.T, names []string, options model.SortOptions) []string {
	t.Helper()
	images := mkImages(names)
	sorted, err := NewEngine().Sort(images, options)
	if err != nil {
		t.Fatalf("Sort 意外失败: %v", err)
	}
	return nameList(sorted)
}

func assertNames(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("排序结果 got=%v, want=%v", got, want)
	}
}

func reverse(s []string) []string {
	out := make([]string, len(s))
	for i := range s {
		out[i] = s[len(s)-1-i]
	}
	return out
}

// SORT_ENGINE §8.1：natural asc 数字段按数值比较。
func TestEngineNaturalAsc(t *testing.T) {
	got := sortNames(t,
		[]string{"1.jpg", "10.jpg", "100.jpg", "11.jpg", "2.jpg", "20.jpg"},
		model.SortOptions{Mode: model.SortModeNatural, Direction: model.SortDirectionAsc})
	assertNames(t, got, []string{"1.jpg", "2.jpg", "10.jpg", "11.jpg", "20.jpg", "100.jpg"})
}

// SORT_ENGINE §4：大小写不敏感（sensitivity: base）。
func TestEngineNaturalCaseInsensitive(t *testing.T) {
	got := sortNames(t, []string{"B2.jpg", "a10.jpg"}, model.SortOptions{Mode: model.SortModeNatural})
	assertNames(t, got, []string{"a10.jpg", "B2.jpg"})

	got = sortNames(t, []string{"File10.txt", "file2.txt"}, model.SortOptions{Mode: model.SortModeNatural})
	assertNames(t, got, []string{"file2.txt", "File10.txt"})
}

// SORT_ENGINE §4：数字段等值回退后整体等值返回 0，稳定排序保持输入顺序。
func TestEngineNaturalEqualKeepsInputOrder(t *testing.T) {
	got := sortNames(t, []string{"a01.jpg", "a1.jpg"}, model.SortOptions{Mode: model.SortModeNatural})
	assertNames(t, got, []string{"a01.jpg", "a1.jpg"})
}

// NaturalCompare 导出函数契约：Chapter2 < Chapter10。
func TestNaturalCompareChunked(t *testing.T) {
	if c := NaturalCompare("Chapter2", "Chapter10"); c >= 0 {
		t.Fatalf("NaturalCompare(Chapter2, Chapter10) = %d, want < 0", c)
	}
	if c := NaturalCompare("abc", "abc"); c != 0 {
		t.Fatalf("NaturalCompare(abc, abc) = %d, want 0", c)
	}
}

func TestFilenameSortAscDesc(t *testing.T) {
	input := []string{"10.jpg", "2.jpg", "b.jpg", "A.jpg"}
	got := sortNames(t, input, model.SortOptions{Mode: model.SortModeFilename, Direction: model.SortDirectionAsc})
	assertNames(t, got, []string{"10.jpg", "2.jpg", "A.jpg", "b.jpg"})

	got = sortNames(t, input, model.SortOptions{Mode: model.SortModeFilename, Direction: model.SortDirectionDesc})
	assertNames(t, got, []string{"b.jpg", "A.jpg", "2.jpg", "10.jpg"})
}

// SORT_ENGINE §8.5：DESC 与 ASC 完全反转（distinct key）。
func TestNaturalDescExactReverseOfAsc(t *testing.T) {
	input := []string{"1.jpg", "10.jpg", "100.jpg", "11.jpg", "2.jpg", "20.jpg"}
	asc := sortNames(t, input, model.SortOptions{Mode: model.SortModeNatural, Direction: model.SortDirectionAsc})
	desc := sortNames(t, input, model.SortOptions{Mode: model.SortModeNatural, Direction: model.SortDirectionDesc})
	assertNames(t, desc, reverse(asc))
}

// time 用例数据：ModifiedAt 升序为 a,b,c；CreatedAt 升序为 c,b,a —— 刻意相反，
// 用于验证两种 mode 各自选对时间字段。
func timeImages() []model.Image {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day := func(n int) time.Time { return base.Add(time.Duration(n) * 24 * time.Hour) }
	return []model.Image{
		{File: model.File{Name: "a.jpg", ModifiedAt: day(1), CreatedAt: day(3)}},
		{File: model.File{Name: "b.jpg", ModifiedAt: day(2), CreatedAt: day(2)}},
		{File: model.File{Name: "c.jpg", ModifiedAt: day(3), CreatedAt: day(1)}},
	}
}

func TestTimeSortModifiedTimeUsesModifiedAt(t *testing.T) {
	images := timeImages()
	NewEngine().Sort(images, model.SortOptions{Mode: model.SortModeModifiedTime, Direction: model.SortDirectionAsc})
	assertNames(t, nameList(images), []string{"a.jpg", "b.jpg", "c.jpg"})
}

func TestTimeSortCreatedTimeUsesCreatedAt(t *testing.T) {
	images := timeImages()
	NewEngine().Sort(images, model.SortOptions{Mode: model.SortModeCreatedTime, Direction: model.SortDirectionAsc})
	assertNames(t, nameList(images), []string{"c.jpg", "b.jpg", "a.jpg"})
}

func TestTimeSortDescReversesAsc(t *testing.T) {
	asc := timeImages()
	NewEngine().Sort(asc, model.SortOptions{Mode: model.SortModeCreatedTime, Direction: model.SortDirectionAsc})
	desc := timeImages()
	NewEngine().Sort(desc, model.SortOptions{Mode: model.SortModeCreatedTime, Direction: model.SortDirectionDesc})
	assertNames(t, nameList(desc), reverse(nameList(asc)))
}

func sizeImages() []model.Image {
	return []model.Image{
		{File: model.File{Name: "big.jpg", Size: 300}},
		{File: model.File{Name: "small.jpg", Size: 1}},
		{File: model.File{Name: "mid.jpg", Size: 100}},
	}
}

func TestSizeSortAscDesc(t *testing.T) {
	images := sizeImages()
	NewEngine().Sort(images, model.SortOptions{Mode: model.SortModeFileSize, Direction: model.SortDirectionAsc})
	assertNames(t, nameList(images), []string{"small.jpg", "mid.jpg", "big.jpg"})

	images = sizeImages()
	NewEngine().Sort(images, model.SortOptions{Mode: model.SortModeFileSize, Direction: model.SortDirectionDesc})
	assertNames(t, nameList(images), []string{"big.jpg", "mid.jpg", "small.jpg"})
}

// SORT_ENGINE §8.6：稳定性 —— 同 key 文件保持输入相对顺序。
func TestStabilityKeepsInputOrderForEqualKeys(t *testing.T) {
	images := []model.Image{
		{File: model.File{Name: "c.jpg", Size: 2}},
		{File: model.File{Name: "a.jpg", Size: 1}},
		{File: model.File{Name: "b.jpg", Size: 1}},
	}
	NewEngine().Sort(images, model.SortOptions{Mode: model.SortModeFileSize, Direction: model.SortDirectionAsc})
	assertNames(t, nameList(images), []string{"a.jpg", "b.jpg", "c.jpg"})
}

func TestEngineNormalizeFallbacks(t *testing.T) {
	input := []string{"b.jpg", "a.jpg", "B.jpg"}

	// 未知 mode → filename；非法 direction → asc
	got := sortNames(t, input, model.SortOptions{Mode: "bogus", Direction: "bogus"})
	assertNames(t, got, []string{"B.jpg", "a.jpg", "b.jpg"})

	// 空 mode（零值，未提供）→ filename asc
	got = sortNames(t, input, model.SortOptions{})
	assertNames(t, got, []string{"B.jpg", "a.jpg", "b.jpg"})

	// 已知 mode + 空 direction → asc
	got = sortNames(t, []string{"2.jpg", "10.jpg"}, model.SortOptions{Mode: model.SortModeNatural})
	assertNames(t, got, []string{"2.jpg", "10.jpg"})
}
