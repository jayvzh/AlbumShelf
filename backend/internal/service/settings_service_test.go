package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
)

// newTestSettingsService 构造「临时图片目录（含 /Comics 子目录）+ 临时数据库」的 SettingsService。
// /Comics 子目录必须真实存在：Resolve 依赖 EvalSymlinks，目标不存在返回 ErrNotFound。
func newTestSettingsService(t *testing.T) *SettingsService {
	t.Helper()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Comics"), 0o755); err != nil {
		t.Fatalf("创建 Comics 目录失败: %v", err)
	}

	db, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("打开临时数据库失败: %v", err)
	}
	t.Cleanup(func() { repository.Close(db) })

	fs := filesystem.NewLocalFilesystem(root)
	return NewSettingsService(fs, repository.NewFolderRepository(db), repository.NewSettingsRepository(db))
}

// 非法/缺省 sort_mode/sort_direction 容错规范化为 filename/asc；
// spread 五字段缺省规范化为 single/left_to_right/1.0/true/false，持久化与返回值一致。
func TestSaveNormalizesInvalidValues(t *testing.T) {
	svc := newTestSettingsService(t)

	saved, err := svc.Save("/Comics", model.FolderSettings{
		SortMode: "bogus", SortDirection: "sideways",
		PageMode: "bogus", ReadOrder: "bogus",
	})
	if err != nil {
		t.Fatalf("Save 失败: %v", err)
	}
	if saved.SortMode != model.SortModeFilename || saved.SortDirection != model.SortDirectionAsc {
		t.Fatalf("非法值未规范化: %+v", saved)
	}
	assertSpreadDefaults(t, saved)

	got, err := svc.Get("/Comics")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.SortMode != model.SortModeFilename || got.SortDirection != model.SortDirectionAsc {
		t.Fatalf("持久化结果与规范化不一致: %+v", got)
	}
	assertSpreadDefaults(t, got)
}

// assertSpreadDefaults 校验 spread 五字段为缺省规范化值（single/left_to_right/1.0/true/false）。
func assertSpreadDefaults(t *testing.T, s *model.FolderSettings) {
	t.Helper()
	if s.PageMode != model.PageModeSingle || s.ReadOrder != model.ReadOrderLeftToRight {
		t.Fatalf("spread 字符串字段应规范化为默认值: %+v", s)
	}
	if s.WideRatio == nil || *s.WideRatio != 1.0 {
		t.Fatalf("wide_ratio 应规范化为 1.0: %+v", s.WideRatio)
	}
	if s.SingleFirstPage == nil || !*s.SingleFirstPage {
		t.Fatalf("single_first_page 应规范化为 true: %+v", s.SingleFirstPage)
	}
	if s.SingleLastPage == nil || *s.SingleLastPage {
		t.Fatalf("single_last_page 应规范化为 false: %+v", s.SingleLastPage)
	}
}

// Get/Save 越界路径透传 filesystem.ErrInvalidPath。
func TestSettingsInvalidPath(t *testing.T) {
	svc := newTestSettingsService(t)

	if _, err := svc.Get("../../x"); !errors.Is(err, filesystem.ErrInvalidPath) {
		t.Fatalf("Get 期望 ErrInvalidPath，实际 %v", err)
	}
	if _, err := svc.Save("../../x", model.FolderSettings{
		SortMode: model.SortModeNatural, SortDirection: model.SortDirectionAsc,
	}); !errors.Is(err, filesystem.ErrInvalidPath) {
		t.Fatalf("Save 期望 ErrInvalidPath，实际 %v", err)
	}
}

// Save 后 Get 返回一致（含 Path 规范化与 spread 五字段指针值）。
func TestSaveThenGetRoundTrip(t *testing.T) {
	svc := newTestSettingsService(t)

	ratio := 1.6
	saved, err := svc.Save("/Comics", model.FolderSettings{
		SortMode: model.SortModeNatural, SortDirection: model.SortDirectionDesc,
		PageMode:        model.PageModeSpread,
		ReadOrder:       model.ReadOrderRightToLeft,
		WideRatio:       &ratio,
		SingleFirstPage: boolPtr(false),
		SingleLastPage:  boolPtr(true),
	})
	if err != nil {
		t.Fatalf("Save 失败: %v", err)
	}
	if saved.Path != "/Comics" || saved.SortMode != model.SortModeNatural || saved.SortDirection != model.SortDirectionDesc {
		t.Fatalf("Save 返回不一致: %+v", saved)
	}
	assertSpreadSaved(t, saved, ratio)

	got, err := svc.Get("/Comics")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.Path != saved.Path || got.SortMode != saved.SortMode || got.SortDirection != saved.SortDirection {
		t.Fatalf("Get 与 Save 不一致: got=%+v saved=%+v", got, saved)
	}
	assertSpreadSaved(t, got, ratio)
}

// assertSpreadSaved 校验 spread 五字段与保存值一致（spread/right_to_left/1.6/false/true）。
func assertSpreadSaved(t *testing.T, s *model.FolderSettings, ratio float64) {
	t.Helper()
	if s.PageMode != model.PageModeSpread || s.ReadOrder != model.ReadOrderRightToLeft {
		t.Fatalf("spread 字符串字段与保存值不一致: %+v", s)
	}
	if s.WideRatio == nil || *s.WideRatio != ratio {
		t.Fatalf("wide_ratio 与保存值不一致: %+v", s.WideRatio)
	}
	if s.SingleFirstPage == nil || *s.SingleFirstPage {
		t.Fatalf("single_first_page 与保存值不一致: %+v", s.SingleFirstPage)
	}
	if s.SingleLastPage == nil || !*s.SingleLastPage {
		t.Fatalf("single_last_page 与保存值不一致: %+v", s.SingleLastPage)
	}
}

// boolPtr 测试辅助：生成 bool 指针。
func boolPtr(v bool) *bool { return &v }
