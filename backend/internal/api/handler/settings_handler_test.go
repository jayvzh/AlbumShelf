package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"imageshelf/backend/internal/api/response"
	"imageshelf/backend/internal/filesystem"
	"imageshelf/backend/internal/repository"
	"imageshelf/backend/internal/service"
)

// newTestSettingsHandler 构造「临时图片目录 + 临时数据库」的 SettingsHandler（参考 settings_service_test.go）。
func newTestSettingsHandler(t *testing.T) *SettingsHandler {
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
	svc := service.NewSettingsService(fs, repository.NewFolderRepository(db), repository.NewSettingsRepository(db))
	return NewSettingsHandler(svc)
}

// newTestGin 构造 TestMode 的 gin 引擎并注册 settings 路由。
func newTestGin(h *SettingsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/folder/settings", h.Get)
	r.PUT("/api/v1/folder/settings", h.Save)
	return r
}

// decodeErrorCode 从错误响应体解析 error.code。
func decodeErrorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body response.ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析错误响应失败: %v", err)
	}
	return body.Error.Code
}

// GET settings：路径合法但不存在 → 404 FOLDER_NOT_FOUND。
func TestGetSettingsFolderNotFound(t *testing.T) {
	h := newTestSettingsHandler(t)
	r := newTestGin(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/folder/settings?path=/NoSuchDir", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际 %d", w.Code)
	}
	if code := decodeErrorCode(t, w); code != response.CodeFolderNotFound {
		t.Fatalf("期望 code=%s，实际 %s", response.CodeFolderNotFound, code)
	}
}

// PUT settings：路径合法但不存在 → 404 FOLDER_NOT_FOUND。
func TestSaveSettingsFolderNotFound(t *testing.T) {
	h := newTestSettingsHandler(t)
	r := newTestGin(h)

	body := `{"path":"/NoSuchDir","sort_mode":"natural","sort_direction":"asc"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/folder/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际 %d", w.Code)
	}
	if code := decodeErrorCode(t, w); code != response.CodeFolderNotFound {
		t.Fatalf("期望 code=%s，实际 %s", response.CodeFolderNotFound, code)
	}
}

// GET settings：越界路径 → 400 INVALID_PATH。
func TestGetSettingsInvalidPath(t *testing.T) {
	h := newTestSettingsHandler(t)
	r := newTestGin(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/folder/settings?path=../../x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d", w.Code)
	}
	if code := decodeErrorCode(t, w); code != response.CodeInvalidPath {
		t.Fatalf("期望 code=%s，实际 %s", response.CodeInvalidPath, code)
	}
}

// PUT settings：regex 模式携带 regex_pattern/regex_config，响应与 GET 回读均保留
// （回归：Save 曾在 request → model 转换时丢失两字段）。
func TestSaveSettingsRegexFieldsRoundTrip(t *testing.T) {
	h := newTestSettingsHandler(t)
	r := newTestGin(h)

	body := `{"path":"/Comics","sort_mode":"regex","sort_direction":"asc",` +
		`"regex_pattern":"chapter(\\d+)_page(\\d+)",` +
		`"regex_config":"{\"rules\":[{\"group\":1,\"type\":\"number\",\"direction\":\"asc\"}]}",` +
		`"view_mode":"single","page_mode":"single","read_order":"left_to_right",` +
		`"wide_ratio":1.0,"single_first_page":false,"single_last_page":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/folder/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	var putResp struct {
		Data struct {
			RegexPattern    *string  `json:"regex_pattern"`
			RegexConfig     *string  `json:"regex_config"`
			PageMode        *string  `json:"page_mode"`
			ReadOrder       *string  `json:"read_order"`
			WideRatio       *float64 `json:"wide_ratio"`
			SingleFirstPage *bool    `json:"single_first_page"`
			SingleLastPage  *bool    `json:"single_last_page"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &putResp); err != nil {
		t.Fatalf("解析 PUT 响应失败: %v", err)
	}
	if putResp.Data.RegexPattern == nil || *putResp.Data.RegexPattern != `chapter(\d+)_page(\d+)` {
		t.Fatalf("PUT 响应 regex_pattern 丢失或不符: %v", putResp.Data.RegexPattern)
	}
	if putResp.Data.RegexConfig == nil || !strings.Contains(*putResp.Data.RegexConfig, `"group":1`) {
		t.Fatalf("PUT 响应 regex_config 丢失或不符: %v", putResp.Data.RegexConfig)
	}
	assertSpreadResponse(t, putResp.Data.PageMode, putResp.Data.ReadOrder, putResp.Data.WideRatio,
		putResp.Data.SingleFirstPage, putResp.Data.SingleLastPage,
		"single", "left_to_right", 1.0, false, false, "PUT")

	// GET 回读校验持久化
	req = httptest.NewRequest(http.MethodGet, "/api/v1/folder/settings?path=/Comics", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 GET 200，实际 %d", w.Code)
	}
	var getResp struct {
		Data struct {
			RegexPattern    *string  `json:"regex_pattern"`
			RegexConfig     *string  `json:"regex_config"`
			PageMode        *string  `json:"page_mode"`
			ReadOrder       *string  `json:"read_order"`
			WideRatio       *float64 `json:"wide_ratio"`
			SingleFirstPage *bool    `json:"single_first_page"`
			SingleLastPage  *bool    `json:"single_last_page"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("解析 GET 响应失败: %v", err)
	}
	if getResp.Data.RegexPattern == nil || *getResp.Data.RegexPattern != `chapter(\d+)_page(\d+)` {
		t.Fatalf("GET 回读 regex_pattern 丢失或不符: %v", getResp.Data.RegexPattern)
	}
	if getResp.Data.RegexConfig == nil || !strings.Contains(*getResp.Data.RegexConfig, `"group":1`) {
		t.Fatalf("GET 回读 regex_config 丢失或不符: %v", getResp.Data.RegexConfig)
	}
	assertSpreadResponse(t, getResp.Data.PageMode, getResp.Data.ReadOrder, getResp.Data.WideRatio,
		getResp.Data.SingleFirstPage, getResp.Data.SingleLastPage,
		"single", "left_to_right", 1.0, false, false, "GET")
}

// assertSpreadResponse 校验响应中 spread 五字段序列化值。
func assertSpreadResponse(t *testing.T, pageMode, readOrder *string, wideRatio *float64,
	first, last *bool, wantMode, wantOrder string, wantRatio float64, wantFirst, wantLast bool, label string) {
	t.Helper()
	if pageMode == nil || *pageMode != wantMode {
		t.Fatalf("%s page_mode 期望 %s，实际 %v", label, wantMode, pageMode)
	}
	if readOrder == nil || *readOrder != wantOrder {
		t.Fatalf("%s read_order 期望 %s，实际 %v", label, wantOrder, readOrder)
	}
	if wideRatio == nil || *wideRatio != wantRatio {
		t.Fatalf("%s wide_ratio 期望 %v，实际 %v", label, wantRatio, wideRatio)
	}
	if first == nil || *first != wantFirst {
		t.Fatalf("%s single_first_page 期望 %v，实际 %v", label, wantFirst, first)
	}
	if last == nil || *last != wantLast {
		t.Fatalf("%s single_last_page 期望 %v，实际 %v", label, wantLast, last)
	}
}

// PUT spread 非默认值 → 响应与 GET 回读均保留（spread 五字段 RoundTrip）。
func TestSaveSettingsSpreadFieldsRoundTrip(t *testing.T) {
	h := newTestSettingsHandler(t)
	r := newTestGin(h)

	body := `{"path":"/Comics","sort_mode":"natural","sort_direction":"asc",` +
		`"page_mode":"spread","read_order":"right_to_left",` +
		`"wide_ratio":1.6,"single_first_page":false,"single_last_page":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/folder/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}

	var putResp struct {
		Data struct {
			PageMode        *string  `json:"page_mode"`
			ReadOrder       *string  `json:"read_order"`
			WideRatio       *float64 `json:"wide_ratio"`
			SingleFirstPage *bool    `json:"single_first_page"`
			SingleLastPage  *bool    `json:"single_last_page"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &putResp); err != nil {
		t.Fatalf("解析 PUT 响应失败: %v", err)
	}
	assertSpreadResponse(t, putResp.Data.PageMode, putResp.Data.ReadOrder, putResp.Data.WideRatio,
		putResp.Data.SingleFirstPage, putResp.Data.SingleLastPage,
		"spread", "right_to_left", 1.6, false, true, "PUT")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/folder/settings?path=/Comics", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 GET 200，实际 %d", w.Code)
	}
	var getResp struct {
		Data struct {
			PageMode        *string  `json:"page_mode"`
			ReadOrder       *string  `json:"read_order"`
			WideRatio       *float64 `json:"wide_ratio"`
			SingleFirstPage *bool    `json:"single_first_page"`
			SingleLastPage  *bool    `json:"single_last_page"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("解析 GET 响应失败: %v", err)
	}
	assertSpreadResponse(t, getResp.Data.PageMode, getResp.Data.ReadOrder, getResp.Data.WideRatio,
		getResp.Data.SingleFirstPage, getResp.Data.SingleLastPage,
		"spread", "right_to_left", 1.6, false, true, "GET")
}
