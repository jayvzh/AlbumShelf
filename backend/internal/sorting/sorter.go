// Package sorting 提供纯函数排序：不依赖数据库、不依赖 HTTP（SORT_ENGINE §1）。
//
// 与 docs/SORT_ENGINE.md 的偏差说明：设计文档写输入为 []model.File，
// 但实际浏览列表的排序对象是 model.Image（内嵌 File，附加像素尺寸）。
// 为避免 Image↔File 切片来回转换，本模块直接操作 []model.Image，
// 字段访问均通过内嵌的 File。
package sorting

import (
	"errors"

	"albumshelf/backend/internal/model"
)

// ErrInvalidRegex 正则编译失败（响应码 INVALID_REGEX 的领域来源，SORT_ENGINE §5.5）。
var ErrInvalidRegex = errors.New("invalid regex")

// Sorter 排序器：对 images 原地稳定排序后返回。
// regex 模式因编译期可失败，不经过本接口（见 Engine.Sort 的特殊分发）。
type Sorter interface {
	Sort(images []model.Image, options model.SortOptions) []model.Image
}

// Engine 排序引擎：规范化 SortOptions 后按 Mode 分发到对应 Sorter。
type Engine struct{}

// NewEngine 构造排序引擎。
func NewEngine() *Engine { return &Engine{} }

// Sort 排序入口：未知/空 Mode 回退 filename，非 asc/desc Direction 回退 asc；
// 所有 sorter 均为稳定排序（SORT_ENGINE §6）。
// regex 模式返回编译错误（errors.Is err ErrInvalidRegex），images 保持输入原序。
func (e *Engine) Sort(images []model.Image, options model.SortOptions) ([]model.Image, error) {
	if options.Mode == model.SortModeRegex {
		c, err := compileRegexSort(options)
		if err != nil {
			return images, err
		}
		return c.sortImages(images), nil
	}
	options = normalizeOptions(options)
	switch options.Mode {
	case model.SortModeNatural:
		return naturalSorter{}.Sort(images, options), nil
	case model.SortModeModifiedTime, model.SortModeCreatedTime:
		return timeSorter{}.Sort(images, options), nil
	case model.SortModeFileSize:
		return sizeSorter{}.Sort(images, options), nil
	default:
		return filenameSorter{}.Sort(images, options), nil
	}
}

// normalizeOptions 规范化排序参数：未知或空 Mode → filename；
// 非 desc 的 Direction（含空值与非法值）→ asc。
func normalizeOptions(options model.SortOptions) model.SortOptions {
	switch options.Mode {
	case model.SortModeFilename, model.SortModeNatural, model.SortModeModifiedTime,
		model.SortModeCreatedTime, model.SortModeFileSize:
	default:
		options.Mode = model.SortModeFilename
	}
	if options.Direction != model.SortDirectionDesc {
		options.Direction = model.SortDirectionAsc
	}
	return options
}

// lessByDir 把三态比较结果 cmp（<0 / 0 / >0）转为"前者应排在前"的布尔值，
// 统一 asc/desc 处理，避免各 sorter 重复反转逻辑。
func lessByDir(cmp int, direction string) bool {
	if direction == model.SortDirectionDesc {
		return cmp > 0
	}
	return cmp < 0
}
