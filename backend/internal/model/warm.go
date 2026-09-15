package model

// WarmLevel 后台缓存预热级别：限定预热器在每个含图目录内的处理范围。
// 档位按写真/COS 图库典型分布设计（单目录大多 ≤100 张）：
// 一级即覆盖绝大多数目录的全量缩略图，二级覆盖 100~300 张的大目录。
type WarmLevel string

const (
	// WarmLevelOff 不预热。
	WarmLevelOff WarmLevel = "off"
	// WarmLevelMinimal 预热级：进目录首屏网格秒出 + 点开头几张秒开。
	WarmLevelMinimal WarmLevel = "minimal"
	// WarmLevel1 一级：对 ≤100 张的目录等于全量缩略图 + 前半预览。
	WarmLevel1 WarmLevel = "level1"
	// WarmLevel2 二级：覆盖 100~300 张的大目录（长篇扫图/合辑）。
	WarmLevel2 WarmLevel = "level2"
	// WarmLevelFull 完整：全库缩略图 + 预览图全量。
	WarmLevelFull WarmLevel = "full"
)

// WarmLevelDefault 新部署未配置时的默认级别。
const WarmLevelDefault = WarmLevelMinimal

// WarmLimits 各级别每个目录按文件名序处理的前缀数量上限；0 = 不限量。
type WarmLimits struct {
	Thumb   int
	Preview int
}

// warmLimits 各级别上限表。Preview ≤ Thumb 保证"先 preview 后派生 thumb"链路成立。
var warmLimits = map[WarmLevel]WarmLimits{
	WarmLevelOff:     {0, 0},
	WarmLevelMinimal: {20, 5},
	WarmLevel1:       {100, 50},
	WarmLevel2:       {300, 150},
	WarmLevelFull:    {0, 0},
}

// ParseWarmLevel 解析级别字符串；非法值返回 false。
func ParseWarmLevel(s string) (WarmLevel, bool) {
	switch l := WarmLevel(s); l {
	case WarmLevelOff, WarmLevelMinimal, WarmLevel1, WarmLevel2, WarmLevelFull:
		return l, true
	default:
		return "", false
	}
}

// Limits 返回该级别的每目录前缀上限（off 返回零值，调用方先判 off 决定是否运行）。
func (l WarmLevel) Limits() WarmLimits {
	return warmLimits[l]
}
