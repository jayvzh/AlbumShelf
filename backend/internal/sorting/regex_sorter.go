package sorting

import (
	"fmt"
	"regexp"
	"sort"

	"imageshelf/backend/internal/model"
)

// PreviewMatch 预览中的单个匹配结果（API.md §3.7）
type PreviewMatch struct {
	Filename string   `json:"filename"`
	Groups   []string `json:"groups"`
}

// SortPreview POST /sort/preview 响应体（API.md §3.7）：
// matches/unmatched 保持输入原序，sorted 为完整最终顺序（匹配项排前、未匹配项原序殿后）。
type SortPreview struct {
	Sorted    []string       `json:"sorted"`
	Matches   []PreviewMatch `json:"matches"`
	Unmatched []string       `json:"unmatched"`
}

// regexCompiled 编译后的正则排序方案
type regexCompiled struct {
	re    *regexp.Regexp
	rules []model.SortRule
}

// compileRegexSort 编译正则并裁剪规则（SORT_ENGINE §5.3/§5.5）：
//   - Rules 为空时回退单规则简写 {group:1, type:number, direction:顶层Direction}
//   - group 越界、非法 type/direction 的规则直接跳过（不报错）
//
// 编译失败返回包装 ErrInvalidRegex 的错误。
func compileRegexSort(options model.SortOptions) (*regexCompiled, error) {
	if options.Regex == "" {
		return nil, fmt.Errorf("%w: 正则表达式为空", ErrInvalidRegex)
	}
	re, err := regexp.Compile(options.Regex)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRegex, err)
	}
	// 顶层方向规范化：仅 regex 分支使用，与 normalizeOptions 保持同一回退语义
	if options.Direction != model.SortDirectionDesc {
		options.Direction = model.SortDirectionAsc
	}
	numGroups := re.NumSubexp()
	var rules []model.SortRule
	if len(options.Rules) == 0 {
		rules = []model.SortRule{{Group: 1, Type: model.SortRuleTypeNumber, Direction: options.Direction}}
	} else {
		rules = make([]model.SortRule, 0, len(options.Rules))
		for _, r := range options.Rules {
			if r.Group < 1 || r.Group > numGroups {
				continue
			}
			if r.Type != model.SortRuleTypeNumber && r.Type != model.SortRuleTypeString {
				continue
			}
			if r.Direction != model.SortDirectionAsc && r.Direction != model.SortDirectionDesc {
				continue
			}
			rules = append(rules, r)
		}
	}
	return &regexCompiled{re: re, rules: rules}, nil
}

// matchGroups 提取文件名的捕获组切片（m[1:]，未参与匹配的组为空串）；
// 不匹配返回 nil。切片长度恒等于捕获组数，保证 compareByRules 索引安全。
func (c *regexCompiled) matchGroups(name string) []string {
	m := c.re.FindStringSubmatch(name)
	if m == nil {
		return nil
	}
	return m[1:]
}

// sortImages 对 images 排序：匹配项按规则稳定排序，未匹配项保持输入原序固定排最后
// （SORT_ENGINE §5.4）。非法正则不会到达此处（编译已在 Engine.Sort 前置）。
func (c *regexCompiled) sortImages(images []model.Image) []model.Image {
	type matchedEntry struct {
		idx    int
		groups []string
	}
	matched := make([]matchedEntry, 0, len(images))
	unmatchedIdx := make([]int, 0)
	for i, img := range images {
		if gs := c.matchGroups(img.Name); gs != nil {
			matched = append(matched, matchedEntry{idx: i, groups: gs})
		} else {
			unmatchedIdx = append(unmatchedIdx, i)
		}
	}
	// idx 与 groups 绑定在同一元素上，SliceStable 交换时不会错位。
	sort.SliceStable(matched, func(a, b int) bool {
		return compareByRules(matched[a].groups, matched[b].groups, c.rules) < 0
	})
	result := make([]model.Image, 0, len(images))
	for _, m := range matched {
		result = append(result, images[m.idx])
	}
	for _, i := range unmatchedIdx {
		result = append(result, images[i])
	}
	return result
}

// RegexPreview 预览正则排序结果（不修改输入），供 RegexEditor 三列预览使用。
// 非法正则返回包装 ErrInvalidRegex 的错误；空输入返回空预览。
func RegexPreview(filenames []string, options model.SortOptions) (*SortPreview, error) {
	c, err := compileRegexSort(options)
	if err != nil {
		return nil, err
	}
	preview := &SortPreview{
		Sorted:    make([]string, 0, len(filenames)),
		Matches:   make([]PreviewMatch, 0, len(filenames)),
		Unmatched: make([]string, 0),
	}
	for _, name := range filenames {
		if gs := c.matchGroups(name); gs != nil {
			preview.Matches = append(preview.Matches, PreviewMatch{Filename: name, Groups: gs})
		} else {
			preview.Unmatched = append(preview.Unmatched, name)
		}
	}
	// Matches 下标按规则排序，matchGroups 内已对齐，直接取 Groups 比较
	matchedIdx := make([]int, len(preview.Matches))
	for i := range matchedIdx {
		matchedIdx[i] = i
	}
	sort.SliceStable(matchedIdx, func(a, b int) bool {
		return compareByRules(preview.Matches[matchedIdx[a]].Groups, preview.Matches[matchedIdx[b]].Groups, c.rules) < 0
	})
	for _, mi := range matchedIdx {
		preview.Sorted = append(preview.Sorted, preview.Matches[mi].Filename)
	}
	preview.Sorted = append(preview.Sorted, preview.Unmatched...)
	return preview, nil
}
