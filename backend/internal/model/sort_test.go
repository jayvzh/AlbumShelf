package model

import (
	"testing"
)

// ParseSortRules 契约（API.md §3.2）：空串视为未提供；非法 JSON / 缺 rules 返回错误。
func TestParseSortRules(t *testing.T) {
	// 空串 → nil, nil（未提供）
	rules, err := ParseSortRules("")
	if rules != nil || err != nil {
		t.Fatalf("空串应返回 nil,nil，got %v, %v", rules, err)
	}

	// 正常解析信封结构
	rules, err = ParseSortRules(`{"rules":[{"group":1,"type":"number","direction":"asc"},{"group":2,"type":"string","direction":"desc"}]}`)
	if err != nil {
		t.Fatalf("合法输入不应报错: %v", err)
	}
	if len(rules) != 2 || rules[0].Group != 1 || rules[0].Type != SortRuleTypeNumber ||
		rules[1].Group != 2 || rules[1].Type != SortRuleTypeString || rules[1].Direction != SortDirectionDesc {
		t.Fatalf("解析结果错误: %+v", rules)
	}

	// 非法 JSON
	if _, err := ParseSortRules(`not json`); err == nil {
		t.Fatal("非法 JSON 应返回错误")
	}
	// 缺 rules 数组
	if _, err := ParseSortRules(`{"foo":1}`); err == nil {
		t.Fatal("缺 rules 数组应返回错误")
	}
	// 空 rules 数组合法（视为显式提供空规则，sorting 层回退单规则简写）
	if _, err := ParseSortRules(`{"rules":[]}`); err != nil {
		t.Fatalf("空 rules 数组不应报错: %v", err)
	}
}
