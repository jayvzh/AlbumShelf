package repository

import (
	"testing"

	"albumshelf/backend/internal/model"
)

// Upsert 插入后按 path JOIN folders 可读回已保存设置。
func TestSettingsUpsertThenGetByPath(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	folders := NewFolderRepository(db)
	settings := NewSettingsRepository(db)

	folderID, err := folders.GetOrCreateID("/Comics")
	if err != nil {
		t.Fatalf("GetOrCreateID 失败: %v", err)
	}
	if err := settings.Upsert(folderID, model.FolderSettings{
		SortMode: model.SortModeNatural, SortDirection: model.SortDirectionDesc,
	}); err != nil {
		t.Fatalf("Upsert 失败: %v", err)
	}

	got, err := settings.GetByPath("/Comics")
	if err != nil {
		t.Fatalf("GetByPath 失败: %v", err)
	}
	if got == nil {
		t.Fatal("期望查到设置，实际为 nil")
	}
	if got.Path != "/Comics" || got.SortMode != model.SortModeNatural || got.SortDirection != model.SortDirectionDesc {
		t.Fatalf("读回不一致: %+v", got)
	}
}

// 同一目录二次 Upsert 覆盖更新且不新增行（folder_id 唯一约束）。
func TestSettingsUpsertTwiceKeepsSingleRow(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	folders := NewFolderRepository(db)
	settings := NewSettingsRepository(db)

	folderID, err := folders.GetOrCreateID("/Comics")
	if err != nil {
		t.Fatalf("GetOrCreateID 失败: %v", err)
	}
	first := model.FolderSettings{SortMode: model.SortModeNatural, SortDirection: model.SortDirectionDesc}
	if err := settings.Upsert(folderID, first); err != nil {
		t.Fatalf("第一次 Upsert 失败: %v", err)
	}
	second := model.FolderSettings{SortMode: model.SortModeFileSize, SortDirection: model.SortDirectionAsc}
	if err := settings.Upsert(folderID, second); err != nil {
		t.Fatalf("第二次 Upsert 失败: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM folder_settings`).Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("期望 1 行，实际 %d 行", count)
	}

	got, err := settings.GetByPath("/Comics")
	if err != nil {
		t.Fatalf("GetByPath 失败: %v", err)
	}
	if got == nil || got.SortMode != model.SortModeFileSize || got.SortDirection != model.SortDirectionAsc {
		t.Fatalf("覆盖更新未生效: %+v", got)
	}
}

// 无记录目录返回 (nil, nil)，不报错。
func TestSettingsGetByPathNoRow(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	settings := NewSettingsRepository(db)

	got, err := settings.GetByPath("/Missing")
	if err != nil {
		t.Fatalf("无记录应返回 nil error，实际 %v", err)
	}
	if got != nil {
		t.Fatalf("无记录应返回 nil 设置，实际 %+v", got)
	}
}

// 列为 NULL 时扫描为空串/nil 指针。
func TestSettingsNullScanAsEmpty(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	folders := NewFolderRepository(db)
	settings := NewSettingsRepository(db)

	folderID, err := folders.GetOrCreateID("/Comics")
	if err != nil {
		t.Fatalf("GetOrCreateID 失败: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO folder_settings (folder_id, updated_at) VALUES (?, CURRENT_TIMESTAMP)`, folderID,
	); err != nil {
		t.Fatalf("插入 NULL 行失败: %v", err)
	}

	got, err := settings.GetByPath("/Comics")
	if err != nil {
		t.Fatalf("GetByPath 失败: %v", err)
	}
	if got == nil {
		t.Fatal("期望查到设置，实际为 nil")
	}
	if got.SortMode != "" || got.SortDirection != "" || got.PageMode != "" || got.ReadOrder != "" {
		t.Fatalf("NULL 应扫描为空串: %+v", got)
	}
	if got.WideRatio != nil || got.SingleFirstPage != nil || got.SingleLastPage != nil {
		t.Fatalf("NULL 应扫描为 nil 指针: %+v", got)
	}
}

// spread 五字段 Upsert 后读回一致；再以空值覆盖后读回空串/nil（NULL 写入路径）。
func TestSettingsSpreadFieldsRoundTrip(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	folders := NewFolderRepository(db)
	settings := NewSettingsRepository(db)

	folderID, err := folders.GetOrCreateID("/Comics")
	if err != nil {
		t.Fatalf("GetOrCreateID 失败: %v", err)
	}
	ratio := 1.6
	if err := settings.Upsert(folderID, model.FolderSettings{
		PageMode:        model.PageModeSpread,
		ReadOrder:       model.ReadOrderRightToLeft,
		WideRatio:       &ratio,
		SingleFirstPage: boolPtr(false),
		SingleLastPage:  boolPtr(true),
	}); err != nil {
		t.Fatalf("Upsert 失败: %v", err)
	}

	got, err := settings.GetByPath("/Comics")
	if err != nil {
		t.Fatalf("GetByPath 失败: %v", err)
	}
	if got == nil || got.PageMode != model.PageModeSpread || got.ReadOrder != model.ReadOrderRightToLeft {
		t.Fatalf("spread 字符串字段读回不一致: %+v", got)
	}
	if got.WideRatio == nil || *got.WideRatio != 1.6 {
		t.Fatalf("wide_ratio 读回不一致: %v", got.WideRatio)
	}
	if got.SingleFirstPage == nil || *got.SingleFirstPage {
		t.Fatalf("single_first_page 读回不一致: %v", got.SingleFirstPage)
	}
	if got.SingleLastPage == nil || !*got.SingleLastPage {
		t.Fatalf("single_last_page 读回不一致: %v", got.SingleLastPage)
	}

	// 空值覆盖：空串/nil 指针写入 NULL，读回空串/nil。
	if err := settings.Upsert(folderID, model.FolderSettings{}); err != nil {
		t.Fatalf("空值 Upsert 失败: %v", err)
	}
	got, err = settings.GetByPath("/Comics")
	if err != nil {
		t.Fatalf("空值 GetByPath 失败: %v", err)
	}
	if got == nil || got.PageMode != "" || got.ReadOrder != "" ||
		got.WideRatio != nil || got.SingleFirstPage != nil || got.SingleLastPage != nil {
		t.Fatalf("空值覆盖后应读回空串/nil: %+v", got)
	}
}

// boolPtr 测试辅助：生成 bool 指针。
func boolPtr(v bool) *bool { return &v }

// GetOrCreateID 幂等：同 path 返回同一 id 且 folders 只有一行。
func TestFolderGetOrCreateIDIdempotent(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	folders := NewFolderRepository(db)

	id1, err := folders.GetOrCreateID("/Comics")
	if err != nil {
		t.Fatalf("第一次 GetOrCreateID 失败: %v", err)
	}
	id2, err := folders.GetOrCreateID("/Comics")
	if err != nil {
		t.Fatalf("第二次 GetOrCreateID 失败: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("同 path 两次 id 不一致: %d != %d", id1, id2)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM folders`).Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("期望 1 行，实际 %d 行", count)
	}
}
