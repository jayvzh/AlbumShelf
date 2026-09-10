// Package repository 提供 SQLite 持久化访问（DATA_DIR/imageshelf.db）。
// repository 层只返回包装后的普通 error（fmt.Errorf + %w），
// API 层错误码语义映射（如 DATABASE_ERROR）由后续 service/handler 完成。
package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // 注册纯 Go SQLite 驱动（driver 名 "sqlite"）
)

// dbName 数据库文件名，固定位于 DATA_DIR 下。
const dbName = "imageshelf.db"

// Open 打开（必要时创建）DATA_DIR 下的 SQLite 数据库并确保 schema 存在。
// 开启 WAL 支持读写并发；当前仅建本任务范围的 image_cache 表（DATA_MODEL §2.5），
// 其余表随对应 Sprint 扩展。
func Open(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dataDir, dbName))
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// WAL 模式：写入持久记录在库文件头，重复设置幂等
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 WAL 失败: %w", err)
	}

	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Close 关闭数据库连接。
func Close(db *sql.DB) error {
	return db.Close()
}

// ensureSchema 以 CREATE TABLE IF NOT EXISTS 幂等建表；
// cache_path 唯一约束支撑缩略图索引按 upsert 覆盖；
// folder_settings.folder_id 唯一约束支撑目录设置按 upsert 覆盖（DATA_MODEL §2.1/§2.2）；
// favorites.file_path 唯一约束支撑收藏去重（表结构 Sprint 6 预留，访问层 Sprint 7 实现）；
// sessions / protected_folders 为 Sprint 7 新增（DATA_MODEL §2.6/§2.7）。
func ensureSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS image_cache (
    id           INTEGER PRIMARY KEY,
    source_path  TEXT,
    variant      TEXT,
    cache_path   TEXT UNIQUE,
    source_mtime INTEGER,
    source_size  INTEGER,
    width        INTEGER,
    height       INTEGER,
    created_at   DATETIME
);
CREATE TABLE IF NOT EXISTS folders (
    id         INTEGER PRIMARY KEY,
    path       TEXT UNIQUE NOT NULL,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE TABLE IF NOT EXISTS folder_settings (
    id                 INTEGER PRIMARY KEY,
    folder_id          INTEGER NOT NULL UNIQUE,
    sort_mode          TEXT,
    sort_direction     TEXT,
    regex_pattern      TEXT,
    regex_config       TEXT,
    view_mode          TEXT,
    page_mode          TEXT,
    read_order         TEXT,
    wide_ratio         REAL,
    single_first_page  INTEGER,
    single_last_page   INTEGER,
    updated_at         DATETIME
);
CREATE TABLE IF NOT EXISTS favorites (
    id         INTEGER PRIMARY KEY,
    file_path  TEXT UNIQUE NOT NULL,
    created_at DATETIME
);
CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS protected_folders (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    path       TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS quick_access (
    id         INTEGER PRIMARY KEY,
    path       TEXT UNIQUE NOT NULL,
    created_at DATETIME
)`)
	if err != nil {
		return fmt.Errorf("初始化 schema 失败: %w", err)
	}
	return nil
}
