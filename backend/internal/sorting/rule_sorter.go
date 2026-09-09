package sorting

import (
	"strconv"
	"strings"

	"imageshelf/backend/internal/model"
)

// compareByRules 多规则链式比较（SORT_ENGINE §5.2）：逐规则比较捕获组内容，
// 首个非零结果即返回；全部相等返回 0（由上层稳定排序保持原序）。
// 前提：规则已经 compileRegexSort 裁剪，group 落在 [1, len(groups)] 内。
func compareByRules(a, b []string, rules []model.SortRule) int {
	for _, r := range rules {
		av, bv := a[r.Group-1], b[r.Group-1]
		var cmp int
		if r.Type == model.SortRuleTypeString {
			cmp = strings.Compare(strings.ToLower(av), strings.ToLower(bv))
		} else {
			cmp = compareNumber(av, bv)
		}
		if cmp != 0 {
			return applyRuleDir(cmp, r.Direction)
		}
	}
	return 0
}

// applyRuleDir 按单条规则方向调整比较结果：desc 取反。
func applyRuleDir(cmp int, direction string) int {
	if direction == model.SortDirectionDesc {
		return -cmp
	}
	return cmp
}

// compareNumber 数值比较：优先按整数解析，任一侧解析失败回退自然比较
// （SORT_ENGINE §5.2，容错形如 "03"、"7x" 的混合内容）。
func compareNumber(a, b string) int {
	na, errA := strconv.Atoi(a)
	nb, errB := strconv.Atoi(b)
	if errA == nil && errB == nil {
		switch {
		case na < nb:
			return -1
		case na > nb:
			return 1
		default:
			return 0
		}
	}
	return NaturalCompare(a, b)
}
