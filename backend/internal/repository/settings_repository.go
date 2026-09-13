package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"albumshelf/backend/internal/model"
)

// SettingsRepository folder_settings 表访问（DATA_MODEL §2.2）。
// 读写排序、regex 与 spread（view/page）字段；view_mode 列保留暂不读写。
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository 构造 SettingsRepository。
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// GetByPath 返回 path 对应目录的设置；无记录返回 (nil, nil)。
// 字符串列为 NULL 时扫描为空串，spread 指针字段为 NULL 时扫描为 nil。
func (r *SettingsRepository) GetByPath(path string) (*model.FolderSettings, error) {
	var mode, direction, regexPattern, regexConfig, pageMode, readOrder sql.NullString
	var wideRatio sql.NullFloat64
	var singleFirstPage, singleLastPage sql.NullBool
	err := r.db.QueryRow(`
SELECT fs.sort_mode, fs.sort_direction, fs.regex_pattern, fs.regex_config,
       fs.page_mode, fs.read_order, fs.wide_ratio, fs.single_first_page, fs.single_last_page
FROM folder_settings fs
JOIN folders f ON f.id = fs.folder_id
WHERE f.path = ?`, path).Scan(&mode, &direction, &regexPattern, &regexConfig,
		&pageMode, &readOrder, &wideRatio, &singleFirstPage, &singleLastPage)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 folder_settings 失败: %w", err)
	}
	return &model.FolderSettings{
		Path:            path,
		SortMode:        nullToEmpty(mode),
		SortDirection:   nullToEmpty(direction),
		RegexPattern:    nullToEmpty(regexPattern),
		RegexConfig:     nullToEmpty(regexConfig),
		PageMode:        nullToEmpty(pageMode),
		ReadOrder:       nullToEmpty(readOrder),
		WideRatio:       nullToFloat64Ptr(wideRatio),
		SingleFirstPage: nullToBoolPtr(singleFirstPage),
		SingleLastPage:  nullToBoolPtr(singleLastPage),
	}, nil
}

// Upsert 按 folder_id 覆盖保存设置（folder_id 唯一约束）。
func (r *SettingsRepository) Upsert(folderID int64, s model.FolderSettings) error {
	if _, err := r.db.Exec(`
INSERT INTO folder_settings (folder_id, sort_mode, sort_direction, regex_pattern, regex_config,
                             page_mode, read_order, wide_ratio, single_first_page, single_last_page, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(folder_id) DO UPDATE SET
    sort_mode = excluded.sort_mode,
    sort_direction = excluded.sort_direction,
    regex_pattern = excluded.regex_pattern,
    regex_config = excluded.regex_config,
    page_mode = excluded.page_mode,
    read_order = excluded.read_order,
    wide_ratio = excluded.wide_ratio,
    single_first_page = excluded.single_first_page,
    single_last_page = excluded.single_last_page,
    updated_at = CURRENT_TIMESTAMP`,
		folderID, s.SortMode, s.SortDirection, s.RegexPattern, s.RegexConfig,
		nullString(s.PageMode), nullString(s.ReadOrder),
		nullFloat64(s.WideRatio), nullBool(s.SingleFirstPage), nullBool(s.SingleLastPage)); err != nil {
		return fmt.Errorf("保存 folder_settings 失败: %w", err)
	}
	return nil
}

// ListAll 返回全部目录设置（含 path，配置导出用，SPRINT7 §5.7）。
func (r *SettingsRepository) ListAll() ([]model.FolderSettings, error) {
	rows, err := r.db.Query(`
SELECT f.path, fs.sort_mode, fs.sort_direction, fs.regex_pattern, fs.regex_config,
       fs.page_mode, fs.read_order, fs.wide_ratio, fs.single_first_page, fs.single_last_page
FROM folder_settings fs
JOIN folders f ON f.id = fs.folder_id
ORDER BY f.path`)
	if err != nil {
		return nil, fmt.Errorf("查询 folder_settings 全量失败: %w", err)
	}
	defer rows.Close()

	var items []model.FolderSettings
	for rows.Next() {
		var mode, direction, regexPattern, regexConfig, pageMode, readOrder sql.NullString
		var wideRatio sql.NullFloat64
		var singleFirstPage, singleLastPage sql.NullBool
		var path string
		if err := rows.Scan(&path, &mode, &direction, &regexPattern, &regexConfig,
			&pageMode, &readOrder, &wideRatio, &singleFirstPage, &singleLastPage); err != nil {
			return nil, fmt.Errorf("扫描 folder_settings 全量失败: %w", err)
		}
		items = append(items, model.FolderSettings{
			Path:            path,
			SortMode:        nullToEmpty(mode),
			SortDirection:   nullToEmpty(direction),
			RegexPattern:    nullToEmpty(regexPattern),
			RegexConfig:     nullToEmpty(regexConfig),
			PageMode:        nullToEmpty(pageMode),
			ReadOrder:       nullToEmpty(readOrder),
			WideRatio:       nullToFloat64Ptr(wideRatio),
			SingleFirstPage: nullToBoolPtr(singleFirstPage),
			SingleLastPage:  nullToBoolPtr(singleLastPage),
		})
	}
	return items, rows.Err()
}

// nullToEmpty NULL 扫描为空串，其余透传。
func nullToEmpty(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

// nullString 空串写入 NULL（未保存语义），其余透传。
func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// nullToFloat64Ptr NULL 扫描为 nil，其余取值。
func nullToFloat64Ptr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

// nullToBoolPtr NULL 扫描为 nil，其余取值。
func nullToBoolPtr(v sql.NullBool) *bool {
	if !v.Valid {
		return nil
	}
	return &v.Bool
}

// nullFloat64 nil 指针写入 NULL，其余取值。
func nullFloat64(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

// nullBool nil 指针写入 NULL，其余取值。
func nullBool(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}
