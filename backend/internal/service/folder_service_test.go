package service

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"albumshelf/backend/internal/filesystem"
	"albumshelf/backend/internal/model"
	"albumshelf/backend/internal/repository"
)

// newTestFolderEnv 构造「临时图片目录 + 临时数据库」的 FolderService 与 SettingsService。
// 假 .jpg 内容非图片，文件头解析失败安全回退 Width/Height=nil，不影响排序断言。
// 三个文件的文件名与大小设计为三种排序结果互不相同：
//
//	filename asc   → 1.jpg, 10.jpg, 2.jpg
//	natural desc   → 10.jpg, 2.jpg, 1.jpg
//	file_size desc → 1.jpg(200B), 2.jpg(20B), 10.jpg(2B)
func newTestFolderEnv(t *testing.T) (*FolderService, *SettingsService) {
	t.Helper()

	root := t.TempDir()
	writeFakeImage := func(name string, size int) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), make([]byte, size), 0o644); err != nil {
			t.Fatalf("写入 %s 失败: %v", name, err)
		}
	}
	writeFakeImage("1.jpg", 200)
	writeFakeImage("2.jpg", 20)
	writeFakeImage("10.jpg", 2)

	db, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("打开临时数据库失败: %v", err)
	}
	t.Cleanup(func() { repository.Close(db) })

	fs := filesystem.NewLocalFilesystem(root)
	settingsRepo := repository.NewSettingsRepository(db)
	folderSvc := NewFolderService(fs, settingsRepo)
	settingsSvc := NewSettingsService(fs, repository.NewFolderRepository(db), settingsRepo)
	return folderSvc, settingsSvc
}

// imageNames 提取排序后的文件名序列。
func imageNames(result *FolderListResult) []string {
	names := make([]string, len(result.Images))
	for i, img := range result.Images {
		names[i] = img.Name
	}
	return names
}

func assertImageOrder(t *testing.T, result *FolderListResult, want []string) {
	t.Helper()
	if got := imageNames(result); !reflect.DeepEqual(got, want) {
		t.Fatalf("排序结果 got=%v, want=%v", got, want)
	}
}

// 优先级 3（默认）：无请求参数且无已保存设置 → filename asc。
func TestListDefaultsToFilenameAsc(t *testing.T) {
	folderSvc, _ := newTestFolderEnv(t)

	result, err := folderSvc.List("/", model.SortOptions{})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	assertImageOrder(t, result, []string{"1.jpg", "10.jpg", "2.jpg"})
}

// 优先级 2（已保存设置）：请求未带参数时使用已保存的 natural desc。
func TestListUsesSavedSettings(t *testing.T) {
	folderSvc, settingsSvc := newTestFolderEnv(t)

	if _, err := settingsSvc.Save("/", model.FolderSettings{
		SortMode: model.SortModeNatural, SortDirection: model.SortDirectionDesc,
	}); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	result, err := folderSvc.List("/", model.SortOptions{})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	assertImageOrder(t, result, []string{"10.jpg", "2.jpg", "1.jpg"})
}

// 优先级 1（请求参数）：带参数时覆盖已保存设置，按 file_size desc 排序。
func TestListRequestParamOverridesSaved(t *testing.T) {
	folderSvc, settingsSvc := newTestFolderEnv(t)

	if _, err := settingsSvc.Save("/", model.FolderSettings{
		SortMode: model.SortModeNatural, SortDirection: model.SortDirectionDesc,
	}); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	result, err := folderSvc.List("/", model.SortOptions{
		Mode: model.SortModeFileSize, Direction: model.SortDirectionDesc,
	})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	assertImageOrder(t, result, []string{"1.jpg", "2.jpg", "10.jpg"})
}
