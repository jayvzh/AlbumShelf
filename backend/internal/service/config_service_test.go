package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
)

// newConfigServicesForFS 在既有图片根上以独立临时数据库组装 config 相关服务。
func newConfigServicesForFS(t *testing.T, fs *filesystem.LocalFilesystem) (*SettingsService, *AppSettingsService, *ConfigService, *repository.FavoriteRepository, *repository.QuickAccessRepository) {
	t.Helper()

	db, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("打开临时数据库失败: %v", err)
	}
	t.Cleanup(func() { repository.Close(db) })

	folders := repository.NewFolderRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	quickAccessRepo := repository.NewQuickAccessRepository(db)
	protected := NewAppSettingsService(fs, repository.NewProtectedFolderRepository(db))
	cfg := NewConfigService(fs, folders, settingsRepo, protected, favoriteRepo, quickAccessRepo)
	settings := NewSettingsService(fs, folders, settingsRepo)
	return settings, protected, cfg, favoriteRepo, quickAccessRepo
}

// newTestConfigEnv 构造含 /Manga、/Novel 目录的临时图片根 + 配套服务。
func newTestConfigEnv(t *testing.T) (*filesystem.LocalFilesystem, *SettingsService, *AppSettingsService, *ConfigService, *repository.FavoriteRepository, *repository.QuickAccessRepository) {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{"Manga", "Novel"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("创建目录 %s 失败: %v", dir, err)
		}
	}
	fs := filesystem.NewLocalFilesystem(root)
	settings, protected, cfg, favRepo, qaRepo := newConfigServicesForFS(t, fs)
	return fs, settings, protected, cfg, favRepo, qaRepo
}

// round-trip：构造设置 + 保护目录 → 导出 → 全新空库导入 → 再导出状态一致。
func TestConfigRoundTrip(t *testing.T) {
	fs, settings, protected, cfg, _, _ := newTestConfigEnv(t)

	// /Manga：spread 全字段；/Novel：regex 模式含正则字段
	wide := 2.5
	singleFirst, singleLast := true, true
	if _, err := settings.Save("/Manga", model.FolderSettings{
		SortMode:        model.SortModeNatural,
		SortDirection:   model.SortDirectionDesc,
		PageMode:        model.PageModeSpread,
		ReadOrder:       model.ReadOrderRightToLeft,
		WideRatio:       &wide,
		SingleFirstPage: &singleFirst,
		SingleLastPage:  &singleLast,
	}); err != nil {
		t.Fatalf("保存 /Manga 设置失败: %v", err)
	}
	if _, err := settings.Save("/Novel", model.FolderSettings{
		SortMode:     model.SortModeRegex,
		RegexPattern: `^(\d+)`,
		RegexConfig:  `{"rules":[{"type":"number"}]}`,
	}); err != nil {
		t.Fatalf("保存 /Novel 设置失败: %v", err)
	}
	if _, err := protected.ReplacePaths([]string{"/Manga"}); err != nil {
		t.Fatalf("设置私有目录失败: %v", err)
	}

	payload1, err := cfg.Export()
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	if payload1.Version != model.ConfigVersion || len(payload1.FolderSettings) != 2 {
		t.Fatalf("导出内容不符: version=%d entries=%d", payload1.Version, len(payload1.FolderSettings))
	}

	// 全新空库模拟清库后导入
	_, _, cfg2, _, _ := newConfigServicesForFS(t, fs)
	result, err := cfg2.Import(model.ConfigPayload{
		Version:          payload1.Version,
		FolderSettings:   payload1.FolderSettings,
		ProtectedFolders: payload1.ProtectedFolders,
	})
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if result.FolderSettings != 2 || result.ProtectedFolders != 1 {
		t.Fatalf("导入条数不符: %+v", result)
	}

	payload2, err := cfg2.Export()
	if err != nil {
		t.Fatalf("再导出失败: %v", err)
	}
	if !reflect.DeepEqual(payload1.FolderSettings, payload2.FolderSettings) {
		t.Fatalf("目录设置 round-trip 不一致:\n got=%+v\nwant=%+v", payload2.FolderSettings, payload1.FolderSettings)
	}
	if !reflect.DeepEqual(payload1.ProtectedFolders, payload2.ProtectedFolders) {
		t.Fatalf("私有目录 round-trip 不一致: got=%v want=%v", payload2.ProtectedFolders, payload1.ProtectedFolders)
	}
}

// 错误 version 拒绝导入（CONFIG_INVALID）；坏 JSON 属 handler 绑定层，不在 service 覆盖范围。
func TestConfigImportRejectsBadVersion(t *testing.T) {
	_, _, _, cfg, _, _ := newTestConfigEnv(t)

	if _, err := cfg.Import(model.ConfigPayload{Version: model.ConfigVersion + 1}); !errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("错误 version 应返回 ErrConfigInvalid，got %v", err)
	}
}

// 收藏纳入导出/导入：round-trip 后收藏路径集合一致；合并语义下已有收藏不覆盖、
// 不删除；目标文件暂缺时导入仅做路径安全校验（created_at 导入不保留，不比较）。
func TestConfigRoundTripWithFavorites(t *testing.T) {
	fs, _, _, cfg, favRepo, _ := newTestConfigEnv(t)

	for _, p := range []string{"/Manga/001.jpg", "/Novel/010.jpg"} {
		if err := favRepo.Add(p); err != nil {
			t.Fatalf("添加收藏 %s 失败: %v", p, err)
		}
	}

	payload1, err := cfg.Export()
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	if len(payload1.Favorites) != 2 {
		t.Fatalf("导出收藏条数不符: %d", len(payload1.Favorites))
	}

	// 全新空库：预置一条已有收藏，导入合并后应保留且不重复
	_, _, cfg2, favRepo2, _ := newConfigServicesForFS(t, fs)
	if err := favRepo2.Add("/Manga/009.jpg"); err != nil {
		t.Fatalf("预置收藏失败: %v", err)
	}
	result, err := cfg2.Import(model.ConfigPayload{
		Version:   payload1.Version,
		Favorites: payload1.Favorites, // 文件暂不存在，导入应仅做路径安全校验
	})
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if result.Favorites != 2 {
		t.Fatalf("导入收藏条数不符: %+v", result)
	}

	want := []string{"/Manga/001.jpg", "/Manga/009.jpg", "/Novel/010.jpg"}
	assertFavoritePaths(t, favRepo2, want)

	payload2, err := cfg2.Export()
	if err != nil {
		t.Fatalf("再导出失败: %v", err)
	}
	got := make([]string, 0, len(payload2.Favorites))
	for _, f := range payload2.Favorites {
		got = append(got, f.Path)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("再导出收藏路径不符: got=%v want=%v", got, want)
	}
}

// 越界路径的收藏条目导入被拒绝。
func TestConfigImportRejectsEscapingFavorite(t *testing.T) {
	_, _, _, cfg, _, _ := newTestConfigEnv(t)

	_, err := cfg.Import(model.ConfigPayload{
		Version:   model.ConfigVersion,
		Favorites: []model.ConfigFavoriteEntry{{Path: "../../etc/passwd"}},
	})
	if !errors.Is(err, filesystem.ErrInvalidPath) {
		t.Fatalf("越界收藏路径应返回 ErrInvalidPath，got %v", err)
	}
}

// 快捷访问纳入导出/导入：round-trip 后固定目录集合一致；合并语义下已有条目
// 不覆盖、不删除；目标目录暂缺时导入仅做路径安全校验。
func TestConfigRoundTripWithQuickAccess(t *testing.T) {
	fs, _, _, cfg, _, qaRepo := newTestConfigEnv(t)

	if err := qaRepo.Add("/Manga"); err != nil {
		t.Fatalf("添加快捷访问 /Manga 失败: %v", err)
	}

	payload1, err := cfg.Export()
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	if !reflect.DeepEqual(payload1.QuickAccess, []string{"/Manga"}) {
		t.Fatalf("导出快捷访问不符: %v", payload1.QuickAccess)
	}

	// 全新空库：预置一条已有条目，导入合并后应保留且不重复
	_, _, cfg2, _, qaRepo2 := newConfigServicesForFS(t, fs)
	if err := qaRepo2.Add("/Novel"); err != nil {
		t.Fatalf("预置快捷访问失败: %v", err)
	}
	result, err := cfg2.Import(model.ConfigPayload{
		Version:     payload1.Version,
		QuickAccess: payload1.QuickAccess,
	})
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if result.QuickAccess != 1 {
		t.Fatalf("导入快捷访问条数不符: %+v", result)
	}

	want := []string{"/Manga", "/Novel"}
	assertQuickAccessPaths(t, qaRepo2, want)

	payload2, err := cfg2.Export()
	if err != nil {
		t.Fatalf("再导出失败: %v", err)
	}
	if !reflect.DeepEqual(payload2.QuickAccess, want) {
		t.Fatalf("再导出快捷访问不符: got=%v want=%v", payload2.QuickAccess, want)
	}
}

// 越界路径的快捷访问条目导入被拒绝。
func TestConfigImportRejectsEscapingQuickAccess(t *testing.T) {
	_, _, _, cfg, _, _ := newTestConfigEnv(t)

	_, err := cfg.Import(model.ConfigPayload{
		Version:     model.ConfigVersion,
		QuickAccess: []string{"../../etc"},
	})
	if !errors.Is(err, filesystem.ErrInvalidPath) {
		t.Fatalf("越界快捷访问路径应返回 ErrInvalidPath，got %v", err)
	}
}

// assertQuickAccessPaths 断言仓库快捷访问路径集合（排序后）与期望一致。
func assertQuickAccessPaths(t *testing.T, repo *repository.QuickAccessRepository, want []string) {
	t.Helper()
	entries, err := repo.List()
	if err != nil {
		t.Fatalf("查询快捷访问失败: %v", err)
	}
	got := make([]string, 0, len(entries))
	for _, q := range entries {
		got = append(got, q.Path)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("快捷访问集合不符: got=%v want=%v", got, want)
	}
}

// assertFavoritePaths 断言仓库收藏路径集合（排序后）与期望一致。
func assertFavoritePaths(t *testing.T, repo *repository.FavoriteRepository, want []string) {
	t.Helper()
	favs, err := repo.List()
	if err != nil {
		t.Fatalf("查询收藏失败: %v", err)
	}
	got := make([]string, 0, len(favs))
	for _, f := range favs {
		got = append(got, f.Path)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("收藏集合不符: got=%v want=%v", got, want)
	}
}
