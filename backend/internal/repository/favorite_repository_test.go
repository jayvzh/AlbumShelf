package repository

import (
	"testing"
)

// Add 后 Exists 为真；重复 Add 幂等（唯一约束 DO NOTHING，仅一行）。
func TestFavoriteAddIdempotent(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	repo := NewFavoriteRepository(db)

	if err := repo.Add("/Comics/001.jpg"); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if err := repo.Add("/Comics/001.jpg"); err != nil {
		t.Fatalf("重复 Add 应幂等不报错: %v", err)
	}

	ok, err := repo.Exists("/Comics/001.jpg")
	if err != nil || !ok {
		t.Fatalf("Exists 应为 true，实际 (%v, %v)", ok, err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM favorites`).Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("重复 Add 后期望 1 行，实际 %d 行", count)
	}
}

// 不存在的路径 Exists 为 false。
func TestFavoriteExistsMissing(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	repo := NewFavoriteRepository(db)

	ok, err := repo.Exists("/Missing.jpg")
	if err != nil || ok {
		t.Fatalf("未收藏路径 Exists 应为 false，实际 (%v, %v)", ok, err)
	}
}

// List 按收藏时间倒序返回；Delete 后不再出现。
func TestFavoriteListOrderAndDelete(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	repo := NewFavoriteRepository(db)

	// 同一秒内插入时 created_at 相同，以 id DESC 保证后插在前
	for _, p := range []string{"/Comics/001.jpg", "/Comics/002.jpg", "/Manga/001.jpg"} {
		if err := repo.Add(p); err != nil {
			t.Fatalf("Add %s 失败: %v", p, err)
		}
	}

	items, err := repo.List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("期望 3 条收藏，实际 %d 条", len(items))
	}
	if items[0].Path != "/Manga/001.jpg" || items[2].Path != "/Comics/001.jpg" {
		t.Fatalf("倒序不符合预期: %v", items)
	}

	if err := repo.Delete("/Comics/001.jpg"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	// 幂等：删除不存在的记录不报错
	if err := repo.Delete("/Comics/001.jpg"); err != nil {
		t.Fatalf("重复 Delete 应幂等不报错: %v", err)
	}

	items, err = repo.List()
	if err != nil {
		t.Fatalf("二次 List 失败: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("删除后期望 2 条收藏，实际 %d 条", len(items))
	}
}

// DeleteMany 清理指定条目并返回删除行数；空切片直接返回 0。
func TestFavoriteDeleteMany(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer Close(db)

	repo := NewFavoriteRepository(db)

	for _, p := range []string{"/A.jpg", "/B.jpg", "/C.jpg"} {
		if err := repo.Add(p); err != nil {
			t.Fatalf("Add %s 失败: %v", p, err)
		}
	}

	n, err := repo.DeleteMany(nil)
	if err != nil || n != 0 {
		t.Fatalf("空切片应返回 (0, nil)，实际 (%d, %v)", n, err)
	}

	n, err = repo.DeleteMany([]string{"/A.jpg", "/Missing.jpg", "/C.jpg"})
	if err != nil {
		t.Fatalf("DeleteMany 失败: %v", err)
	}
	if n != 2 {
		t.Fatalf("期望删除 2 行，实际 %d 行", n)
	}

	ok, err := repo.Exists("/B.jpg")
	if err != nil || !ok {
		t.Fatalf("未列入清理的 /B.jpg 应保留，实际 (%v, %v)", ok, err)
	}
}
