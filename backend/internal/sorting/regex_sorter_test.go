package sorting

import (
	"errors"
	"testing"

	"albumshelf/backend/internal/model"
)

// regexOptions 构造 regex 排序选项的测试助手。
func regexOptions(regex string, rules ...model.SortRule) model.SortOptions {
	return model.SortOptions{Mode: model.SortModeRegex, Regex: regex, Rules: rules}
}

// SORT_ENGINE §8.2：多规则排序，chapter1_page1 / chapter1_page10 / chapter2_page1 / chapter10_page1
// （group1 数字升序 → group2 数字升序）。
func TestRegexSortMultiRules(t *testing.T) {
	rules := []model.SortRule{
		{Group: 1, Type: model.SortRuleTypeNumber, Direction: model.SortDirectionAsc},
		{Group: 2, Type: model.SortRuleTypeNumber, Direction: model.SortDirectionAsc},
	}
	got := sortNames(t,
		[]string{"chapter10_page1.jpg", "chapter1_page10.jpg", "chapter2_page1.jpg", "chapter1_page1.jpg"},
		regexOptions(`chapter(\d+)_page(\d+)`, rules...))
	assertNames(t, got, []string{"chapter1_page1.jpg", "chapter1_page10.jpg", "chapter2_page1.jpg", "chapter10_page1.jpg"})
}

// SORT_ENGINE §5.3：无 rules 时回退单规则简写 {group:1, number, 顶层 direction}，
// 数字按数值比较（ch1 < ch2 < ch10）。
func TestRegexSortSingleRuleShorthand(t *testing.T) {
	got := sortNames(t,
		[]string{"ch10.jpg", "ch2.jpg", "ch1.jpg"},
		model.SortOptions{Mode: model.SortModeRegex, Regex: `ch(\d+)`, Direction: model.SortDirectionAsc})
	assertNames(t, got, []string{"ch1.jpg", "ch2.jpg", "ch10.jpg"})
}

// SORT_ENGINE §8.3：未匹配文件固定排最后且保持输入原序。
func TestRegexSortUnmatchedLastKeepsInputOrder(t *testing.T) {
	rules := []model.SortRule{{Group: 1, Type: model.SortRuleTypeNumber, Direction: model.SortDirectionAsc}}
	got := sortNames(t,
		[]string{"misc_b.jpg", "ch2.jpg", "misc_a.jpg", "ch1.jpg"},
		regexOptions(`ch(\d+)`, rules...))
	assertNames(t, got, []string{"ch1.jpg", "ch2.jpg", "misc_b.jpg", "misc_a.jpg"})
}

// SORT_ENGINE §8.4：非法/空正则返回 ErrInvalidRegex，images 保持输入原序。
func TestRegexSortInvalidRegexReturnsError(t *testing.T) {
	images := mkImages([]string{"a.jpg", "b.jpg"})
	sorted, err := NewEngine().Sort(images, model.SortOptions{Mode: model.SortModeRegex, Regex: "ch("})
	if !errors.Is(err, ErrInvalidRegex) {
		t.Fatalf("err = %v, want ErrInvalidRegex", err)
	}
	assertNames(t, nameList(sorted), []string{"a.jpg", "b.jpg"})

	if _, err := NewEngine().Sort(images, model.SortOptions{Mode: model.SortModeRegex}); !errors.Is(err, ErrInvalidRegex) {
		t.Fatalf("空正则 err = %v, want ErrInvalidRegex", err)
	}
}

// SORT_ENGINE §8.5：DESC 与 ASC 完全反转（distinct key，regex 模式）。
func TestRegexSortDescReversesAsc(t *testing.T) {
	input := []string{"ch1.jpg", "ch10.jpg", "ch2.jpg"}
	asc := sortNames(t, input, regexOptions(`ch(\d+)`,
		model.SortRule{Group: 1, Type: model.SortRuleTypeNumber, Direction: model.SortDirectionAsc}))
	desc := sortNames(t, input, regexOptions(`ch(\d+)`,
		model.SortRule{Group: 1, Type: model.SortRuleTypeNumber, Direction: model.SortDirectionDesc}))
	assertNames(t, desc, reverse(asc))
}

// SORT_ENGINE §8.6：稳定性 —— 同 key 文件保持输入相对顺序。
func TestRegexSortStabilityEqualKeys(t *testing.T) {
	rules := []model.SortRule{{Group: 1, Type: model.SortRuleTypeString, Direction: model.SortDirectionAsc}}
	got := sortNames(t,
		[]string{"ch5_c.jpg", "ch5_a.jpg", "ch5_b.jpg"},
		regexOptions(`ch(\d+)`, rules...))
	assertNames(t, got, []string{"ch5_c.jpg", "ch5_a.jpg", "ch5_b.jpg"})
}

// SORT_ENGINE §5.5：group 越界 / 非法 type 的规则裁剪后无有效规则 → 保持输入原序。
func TestRegexSortSkipsInvalidRules(t *testing.T) {
	rules := []model.SortRule{
		{Group: 5, Type: model.SortRuleTypeNumber, Direction: model.SortDirectionAsc},
		{Group: 1, Type: "bogus", Direction: model.SortDirectionAsc},
	}
	got := sortNames(t, []string{"ch2.jpg", "ch1.jpg"}, regexOptions(`ch(\d+)`, rules...))
	assertNames(t, got, []string{"ch2.jpg", "ch1.jpg"})
}

// SORT_ENGINE §5.2：string 类型规则大小写不敏感。
func TestRegexSortStringRuleCaseInsensitive(t *testing.T) {
	rules := []model.SortRule{{Group: 1, Type: model.SortRuleTypeString, Direction: model.SortDirectionAsc}}
	got := sortNames(t, []string{"f_B.jpg", "f_a.jpg"}, regexOptions(`f_(\w+)`, rules...))
	assertNames(t, got, []string{"f_a.jpg", "f_B.jpg"})
}

// SORT_ENGINE §5.2：number 规则对非纯数字内容回退自然比较（"v2" < "v10"）。
func TestRegexSortNumberRuleFallsBackToNatural(t *testing.T) {
	rules := []model.SortRule{{Group: 1, Type: model.SortRuleTypeNumber, Direction: model.SortDirectionAsc}}
	got := sortNames(t, []string{"x_v10.jpg", "x_v2.jpg"}, regexOptions(`x_(v\d+)`, rules...))
	assertNames(t, got, []string{"x_v2.jpg", "x_v10.jpg"})
}

// RegexPreview（API.md §3.7）：matches/unmatched 保持输入原序，
// sorted 为完整最终顺序；非法正则返回 ErrInvalidRegex。
func TestRegexPreviewLayout(t *testing.T) {
	rules := []model.SortRule{{Group: 1, Type: model.SortRuleTypeNumber, Direction: model.SortDirectionAsc}}
	preview, err := RegexPreview(
		[]string{"z.jpg", "ch2.jpg", "y.jpg", "ch1.jpg"},
		regexOptions(`ch(\d+)`, rules...))
	if err != nil {
		t.Fatalf("RegexPreview 失败: %v", err)
	}
	assertNames(t, preview.Sorted, []string{"ch1.jpg", "ch2.jpg", "z.jpg", "y.jpg"})
	if len(preview.Matches) != 2 ||
		preview.Matches[0].Filename != "ch2.jpg" || preview.Matches[0].Groups[0] != "2" ||
		preview.Matches[1].Filename != "ch1.jpg" || preview.Matches[1].Groups[0] != "1" {
		t.Fatalf("matches 错误: %+v", preview.Matches)
	}
	assertNames(t, preview.Unmatched, []string{"z.jpg", "y.jpg"})

	if _, err := RegexPreview([]string{"a"}, model.SortOptions{Mode: model.SortModeRegex, Regex: "("}); !errors.Is(err, ErrInvalidRegex) {
		t.Fatalf("err = %v, want ErrInvalidRegex", err)
	}
}
