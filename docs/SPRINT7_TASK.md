# Sprint 7 开发任务书（AI IDE Task Specification）

> 用法：将本文档整体复制给 AI IDE（Cursor / Claude Code / Windsurf 等）作为 Sprint 7 的任务输入。
> Sprint 7 定位：**功能增强 Sprint**——在已完成的 Sprint 0-6 之上新增「用户登录 + 设置中心」：单管理员认证、私有目录、缓存管理、配置导入导出。原「Docker + NAS 部署收尾」已顺延为 Sprint 8。

---

## 1. 任务上下文

在开始编码前，AI 必须先完整阅读以下文档：

```
docs/PRD.md（§7 成功标准；登录系统为计划增补，PRD 无对应条目，以本文档为准）
docs/ARCHITECTURE.md（§5 模块划分、§7 配置项表——本 Sprint 需同步修订）
docs/PROJECT_STRUCTURE.md（§2 预留 router/index.ts 与 SettingsPage.vue、§13 Store 设计、§19 依赖规则——本 Sprint 需同步修订）
docs/API.md（§1 通用约定、§3 端点风格、§5 错误码——本 Sprint 需同步修订）
docs/DATA_MODEL.md（§2 表结构——本 Sprint 需同步修订）
docs/DEVELOPMENT_RULES.md（分层红线：Handler → Service → Domain/Repository，禁止跨层）
docs/SPRINT_PLAN.md（v1.1，Sprint 7 部分）
开发规则.md（根目录精简版：命令与自测约定）
```

### 已确认决策（不要重新设计）

| 决策点 | 结论 |
| --- | --- |
| 账户模型 | 单管理员 + 游客可浏览公开目录；私有目录对游客**完全隐藏**（列表过滤 + 直接访问 401），登录后可见 |
| 凭证存储 | 环境变量 `AUTH_USERNAME` / `AUTH_PASSWORD`；`AUTH_PASSWORD` 未设置 = 登录系统整体关闭（向后兼容） |
| 认证载体 | **Cookie 会话**（图片经 `<img>` 标签加载，无法携带 JWT header）；token = crypto/rand 32B hex；HttpOnly + SameSite=Lax；sessions 表 SQLite 持久化（重启不丢） |
| 导入策略 | 配置导入为按路径 **upsert 合并**（非整库替换），降低误操作风险 |

### 现状基线（动手前必须先读对应代码再改）

| 现状 | 说明 |
| --- | --- |
| 前端路由 | `frontend/src/router/index.ts` 仅 `// TODO(Sprint 2)` 占位；`main.ts` 仅注册 Pinia；`App.vue` 直接渲染 `BrowserPage`；**vue-router 尚未安装** |
| SettingsPage | `frontend/src/pages/SettingsPage.vue` 仅 `<!-- TODO(Sprint 4) -->` 占位；PROJECT_STRUCTURE.md §2 已预留该文件与 AppHeader Settings 入口 |
| 后端认证 | 无 `internal/auth` 包；`middleware/` 仅 error.go / logger.go；无 sessions、protected_folders 表 |
| Config | `backend/internal/config/config.go` 仅 IMAGE_ROOT / DATA_DIR / PORT 三字段，`getEnv(key, fallback)` 模式 |
| image_cache 表 | 含 `source_path` / `cache_path` / `variant` 等列；Repository 已有 `Upsert(ctx, record)` 与 `DeleteBySource(ctx, sourcePath, variant)`——孤儿判定与清理的实现基础 |
| 缓存键 | sha1(sourcePath\|mtime\|size\|variant\|sizeBucket)（`internal/thumbnail/cache.go` 的 `CachePath()`）；原图变更后索引行被删除但旧磁盘文件遗留 → 形成"磁盘有但索引无"孤儿 |
| AppHeader | 现有主题切换 / ViewModeSwitch / SortMenu / SpreadControls / HelpModal，无 Settings 入口、无登录态 |
| Store | 仅 4 个（folder / settings / viewer / spread）；PROJECT_STRUCTURE.md §13 规定 4 个——本 Sprint 增加第 5 个 auth store，需同步修订 §13 |
| api.ts | 统一 `request<T>()` + `ApiError(code, message)`；401 统一处理挂在此层 |
| 主题 | `useTheme.ts` 经 localStorage（键 `imageshelf:theme`）持久化，属用户本地偏好，**不入全局配置导入导出** |
| health | `GET /api/v1/health` 返回 `{"data":{"status":"ok"}}`，保持不变 |

---

## 2. 本 Sprint 目标

**用户登录 + 设置中心**

1. 认证：单管理员（env 凭证）+ Cookie 会话（SQLite 持久化，后端重启会话不丢）
2. 私有目录：游客完全隐藏（目录列表过滤 + 直接访问 401），登录后可见
3. 设置页四区块：私有目录 / 缓存管理（体积统计 + 孤儿清理 + 全部清理）/ 配置导入导出 / 账户与系统信息
4. vue-router 占位转正（`/` / `/settings` / `/login` + 路由守卫）+ auth store（第 5 个 Store）
5. 兼容性：`AUTH_PASSWORD` 未设置时全部行为与 Sprint 6 一致（零配置向后兼容）
6. 文档同步：API / DATA_MODEL / PROJECT_STRUCTURE / ARCHITECTURE / .env.example / docker-compose（env 透传）/ README

---

## 3. 允许创建 / 填充 / 修改的文件（精确清单）

> **超出此清单的任何创建/修改一律禁止。**

### A. 新建（Backend，19 个，含 4 个测试）

```
backend/internal/auth/auth.go                        # AuthService：Login / Logout / Validate / Enabled，constant-time 比对
backend/internal/auth/auth_test.go                   # 登录成功 / 错误密码 / 未知用户 / 过期 token / 未启用拒绝
backend/internal/middleware/access.go                # 会话解析 + RequireAuth（管理端点）+ ProtectedAccess（内容端点）
backend/internal/repository/session_repository.go    # sessions 表 CRUD + 过期清理
backend/internal/repository/protected_folder_repository.go   # protected_folders 表 CRUD（含全量替换）
backend/internal/model/auth.go                       # Session、AuthStatus
backend/internal/model/appsettings.go                # ProtectedFolder、CacheStats、CleanupResult、ConfigPayload
backend/internal/api/request/auth_request.go         # LoginRequest
backend/internal/api/request/config_request.go       # ConfigImportRequest
backend/internal/api/handler/auth_handler.go         # POST login / POST logout / GET status
backend/internal/api/handler/appsettings_handler.go  # GET/PUT /protected-folders
backend/internal/api/handler/cache_handler.go        # GET /cache/stats、POST /cache/cleanup/orphan、POST /cache/cleanup/all
backend/internal/api/handler/config_handler.go       # GET /config/export（附件下载）、POST /config/import
backend/internal/service/appsettings_service.go      # 私有目录列表 / 全量替换 + IsProtected(relPath) 子树匹配
backend/internal/service/appsettings_service_test.go # 前缀边界：/Private 不误伤 /PrivateX、子树命中、空路径拒绝
backend/internal/service/cache_service.go            # 统计（thumb/preview count+bytes）/ 两类孤儿清理 / 全部清理
backend/internal/service/cache_service_test.go       # 孤儿两类判定 + 统计准确（临时目录构造）
backend/internal/service/config_service.go           # 导出聚合 / 导入 upsert 合并
backend/internal/service/config_service_test.go      # 导出 → 导入 round-trip
```

### B. 填充占位（2 个，现为 TODO 注释）

```
frontend/src/router/index.ts             # createRouter + 三条路由 + 全局守卫
frontend/src/pages/SettingsPage.vue      # 组合四区块（纵向分节 + 标题）
```

### A'. 新建（Frontend，10 个）

```
frontend/src/pages/LoginPage.vue                       # 登录表单（用户名/密码/错误提示），登录成功跳 redirect 参数或 /
frontend/src/components/settings/ProtectedFoldersSection.vue   # 私有路径列表 + 添加/删除 + 整体保存
frontend/src/components/settings/CacheSection.vue      # 体积统计展示 + 孤儿清理 / 全部清理（均二次确认）
frontend/src/components/settings/ConfigSection.vue     # 导出下载 + 导入文件选择（结果提示）
frontend/src/components/settings/AccountSection.vue    # 登录态/退出 + 系统信息（IMAGE_ROOT/DATA_DIR，仅已登录可见，只读）
frontend/src/services/auth.service.ts                  # login / logout / fetchStatus
frontend/src/services/appsettings.service.ts           # 私有目录 / 缓存统计与清理 / 导入导出
frontend/src/stores/auth.ts                            # 第 5 个 Pinia Store（enabled/authenticated/username/loaded + fetchStatus/login/logout）
frontend/src/types/auth.ts                             # AuthStatus、LoginRequest
frontend/src/types/appsettings.ts                      # CacheStats、CleanupResult、ConfigPayload
```

### C. 允许修改的既有文件（附理由，改动保持最小）

```
backend/
├── internal/config/config.go              # +AUTH_USERNAME / AUTH_PASSWORD / SESSION_MAX_AGE（getEnv 模式）
├── internal/repository/database.go        # ensureSchema 追加 CREATE TABLE IF NOT EXISTS sessions、protected_folders
├── internal/app/app.go                    # 组装 AuthService / 新 Repository / 新 Service / 新 Handler / 中间件注入
├── internal/api/router.go                 # 注册 10 个新端点 + 会话解析与访问控制中间件挂载
└── internal/service/folder_service.go     # List()：auth enabled && 游客时，folders 数组过滤私有条目
frontend/
├── src/main.ts                            # app.use(router)
├── src/App.vue                            # <router-view /> 替换直渲 BrowserPage（BrowserPage 行为不变，挂 / 路由）
├── package.json                           # dependencies 增加 vue-router@4
├── src/services/api.ts                    # 响应 401 统一处理：跳 /login?redirect=<当前路径>（登录页内请求除外，防循环）
└── src/components/layout/AppHeader.vue    # 齿轮按钮 → /settings；auth enabled 时显示登录入口或用户名 + 退出
配置 / 文档
├── .env.example                           # +AUTH_USERNAME / AUTH_PASSWORD / SESSION_MAX_AGE（带注释示例）
├── docker-compose.yml                     # environment 仅透传上述三变量（镜像/Dockerfile 迭代归 Sprint 8）
├── docs/API.md                            # 新增认证/私有目录/缓存/导入导出端点 + 错误码 UNAUTHORIZED/INVALID_CREDENTIALS/CONFIG_INVALID + Cookie 会话说明
├── docs/DATA_MODEL.md                     # 新增 sessions、protected_folders 两表 DDL 与说明
├── docs/PROJECT_STRUCTURE.md              # §2 路由/设置页/登录页转正、后端 auth 模块；§13 增补第 5 个 Store（理由：认证为全局横切状态）
├── docs/ARCHITECTURE.md                   # 模块划分 + 配置项表 + 认证/访问控制流程简述
└── README.md                              # 环境变量表补三变量 + 「登录系统」一节（简述：默认关闭、如何开启、HTTPS 由 NAS 反代负责）
```

---

## 4. 禁止开发

本 Sprint **禁止**创建或实现：

```
多用户 / 多角色 / 注册 / 找回密码                     （单管理员，env 凭证）
修改密码 UI 与任何密码持久化                         （凭证每次启动从 env 读取）
JWT / localStorage 令牌                              （<img> 无法携带 header，必须 Cookie）
缓存自动清理策略 / 定时任务 / 容量阈值               （仅手动清理）
HTTPS / 反向代理 / Basic Auth                        （由 NAS 侧处理，README 提示即可）
Docker 镜像 / Dockerfile 迭代                        （Sprint 8；compose env 透传除外）
favorites 功能化（repository / UI / API）            （MVP 只建表，PRD F010）
Dummy Page 补位 / 静态奇偶配对 / DividePage          （Phase 2）
主题等本地偏好纳入配置导入导出                        （localStorage 偏好非服务端配置）
除 vue-router@4 外任何新前端依赖                      （UI 组件复用现有 Headless UI / 自有组件）
除 Go 标准库外任何新后端依赖                          （密码比对 crypto/subtle、token crypto/rand、JSON 标准库）
```

架构红线（分层、数据流、禁止 Component 直接 fetch 等）以 docs/DEVELOPMENT_RULES.md 与 PROJECT_STRUCTURE.md §19 为准，本文档不重复。

---

## 5. 技术要求

### 5.1 配置与会话开关（config.go）

- 新增字段：`AuthUsername`（默认空）、`AuthPassword`（默认空）、`SessionMaxAge`（`time.ParseDuration` 解析，默认 `168h`，非法值回退默认）
- `AuthEnabled() bool`：`AUTH_PASSWORD != ""` 即启用；**未启用时本 Sprint 全部新语义关闭**——所有接口放行、私有目录不生效、`GET /auth/status` 返回 `{enabled: false}`、`POST /auth/login` 返回 400（错误码沿用 `CONFIG_INVALID` 不合适，用 `UNAUTHORIZED`）

### 5.2 认证服务（internal/auth/auth.go）

- `Login(username, password)`：
  - 用户名与密码均用 `crypto/subtle.ConstantTimeCompare` 比较（未知用户也走一次伪比较，避免时序侧信道区分"用户不存在/密码错误"，但对外统一返回 `INVALID_CREDENTIALS`）
  - 成功：token = `hex.EncodeToString(32B crypto/rand)`；写入 sessions 表（`created_at` / `expires_at = now + SessionMaxAge`）
  - 设置 Cookie：名 `session_token`；`HttpOnly` + `SameSite=Lax` + `Path=/` + `MaxAge`（秒）
- `Logout(token)`：删除 sessions 行 + 清 Cookie（`MaxAge = -1`）
- `Validate(token)`：查表；未命中或已过期 → 无效；命中的过期记录顺带删除（惰性清理）；**不做滑动续期**（固定过期，保持简单）
- `Enabled() bool`
- `auth_test.go` 必须覆盖：正确凭证成功、错误密码 `INVALID_CREDENTIALS`、未知用户同样 `INVALID_CREDENTIALS`、过期 token 无效、未启用时 Login 拒绝

### 5.3 数据表（database.go，CREATE TABLE IF NOT EXISTS 幂等风格）

```sql
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
```

- `path` 存 IMAGE_ROOT 相对路径，统一规范化为 `/` 前缀（如 `/Private`）

### 5.4 访问控制（middleware/access.go）

- 会话解析中间件：读 `session_token` cookie → `Validate` → `c.Set("authenticated", bool)`；每请求执行
- `RequireAuth`：未登录 → 401 `UNAUTHORIZED`。挂载于管理端点：`PUT /protected-folders`、`POST /cache/cleanup/*`、`GET /config/export`、`POST /config/import`（auth enabled 时才拦截）
- `ProtectedAccess`：对携带 `path` 参数的内容端点 `/folders`、`/folder/settings`、`/image`、`/image/info`、`/thumbnail` 判断：auth enabled && 游客 && `IsProtected(path)` → 401 `UNAUTHORIZED`
- auth disabled：两个中间件均直通（放行）
- 中间件只能依赖 auth service 与 appsettings service 接口，不得直接访问 Repository（分层红线）

### 5.5 私有目录（appsettings_service）

- `IsProtected(relPath)`：子树前缀匹配——保护项 `p` 命中条件：`relPath == p || strings.HasPrefix(relPath, p + "/")`；即 `/Private` 保护自身与全部子级，**不误伤 `/PrivateX`**
- `GET /api/v1/protected-folders` → `{paths: ["...", ...]}`
- `PUT /api/v1/protected-folders`：请求 `{paths: [...]}`，**全量替换语义**（事务内 DELETE 全部 + INSERT）；每项先经 `filesystem.Resolve` 校验（非法/不存在 → `INVALID_PATH`，沿用现有错误码）；空字符串与 `/` 拒绝
- 游客目录列表过滤：`folder_service.List()` 在 auth enabled && 游客时剔除 folders 数组中命中 `IsProtected` 的条目；直接路径访问由 5.4 的 `ProtectedAccess` 拦截
- `appsettings_service_test.go` 必须覆盖：`/Private` 与 `/Private/a.jpg` 命中、`/PrivateX` 不命中、空路径拒绝、根内子路径前缀边界

### 5.6 缓存管理（cache_service）

- `Stats`：扫描 `/data/cache/thumb/` 与 `/data/cache/preview/`，输出 `{thumb: {count, bytes}, preview: {count, bytes}}`（以磁盘为准）
- Orphan 清理，**覆盖两类**：
  1. 索引记录的 `source_path` 在 IMAGE_ROOT 中已不存在 → 删对应缓存文件 + 删索引行
  2. 磁盘缓存文件（hash 命名）不在索引 `cache_path` 集合 → 删除文件（原图变更后索引行已删但旧文件遗留的形态）
- All 清理：清空两个缓存目录内容 + `DELETE FROM image_cache`
- 返回 `CleanupResult {removed_files, removed_bytes}`；清理与统计操作不加锁，但须在同一请求内顺序执行，避免与缩略图生成并发时出现半删状态可接受（自用场景，文档注明"清理期间浏览会触发重建"）
- `cache_service_test.go`：临时目录构造正常缓存 + 两类孤儿，断言统计准确、两类孤儿均被清理、正常缓存不受影响、all 清理后目录与索引皆空

### 5.7 配置导入导出（config_service）

- 导出 `GET /api/v1/config/export`：
  ```json
  {
    "version": 1,
    "exported_at": "2026-09-09T12:00:00+08:00",
    "folder_settings": [ ...folder_settings 全量... ],
    "protected_folders": ["/Private", "..."]
  }
  ```
  响应头 `Content-Disposition: attachment; filename=albumshelf-config-<YYYYMMDD>.json`
- 导入 `POST /api/v1/config/import`：JSON 解析失败或 `version != 1` → 400 `CONFIG_INVALID`；`folder_settings` 按目录 path **upsert 合并**；`protected_folders` 按 path upsert 合并（每项经 Resolve 校验）；响应返回各类导入条数
- 主题等 localStorage 偏好不参与导入导出（见 §4）
- `config_service_test.go`：构造设置 + 保护目录 → 导出 → 清库 → 导入 → 状态一致（round-trip）；坏 JSON 与错误 version 均返回 `CONFIG_INVALID`

### 5.8 路由与组装（router.go / app.go）

- `NewRouter` 增加 4 个 handler 参数（auth / appsettings / cache / config）；v1 组内注册共 **10 个新端点**：
  ```
  POST /api/v1/auth/login
  POST /api/v1/auth/logout
  GET  /api/v1/auth/status
  GET  /api/v1/protected-folders
  PUT  /api/v1/protected-folders
  GET  /api/v1/cache/stats
  POST /api/v1/cache/cleanup/orphan
  POST /api/v1/cache/cleanup/all
  GET  /api/v1/config/export
  POST /api/v1/config/import
  ```
- 中间件顺序：logger → error/recovery → 会话解析 → 路由组（RequireAuth / ProtectedAccess 按端点挂载）
- `app.go` 组装顺序沿用现有模式：Config → Filesystem/SQLite → Repository → Service → Handler → Router；AuthService 在 Config 之后立即创建
- `GET /api/v1/auth/status` 响应：`{enabled, authenticated, username?, image_root?, data_dir?}`——`image_root` / `data_dir` **仅在已登录时返回**（游客不泄露服务器路径），供 AccountSection 系统信息展示

### 5.9 前端路由与守卫（router/index.ts / main.ts / App.vue）

- 路由：`/` → BrowserPage（name `browser`，现有行为不变）、`/settings` → SettingsPage、`/login` → LoginPage（`meta: { public: true }`）；未匹配兜底重定向 `/`
- `main.ts`：`app.use(router)`；`App.vue`：`<router-view />`
- 全局守卫（`beforeEach`）：
  - 先确保 auth store `fetchStatus()` 已完成（幂等，仅首次请求）
  - `to === '/settings'` 且 auth enabled 且未登录 → `redirect: /login?redirect=/settings`
  - `to === '/login'` 且 auth disabled → 重定向 `/`
  - auth disabled 时 `/settings` 直接放行（单用户场景管理端点开放）
- `api.ts` 401 统一处理：`ApiError.code === 'UNAUTHORIZED'` 时跳 `/login?redirect=<当前路由全路径>`；发起自登录页或 `/auth/status` 探测的请求**除外**（防循环跳转）

### 5.10 auth store 与 AppHeader

- `stores/auth.ts`（第 5 个 Store）：state `{enabled, authenticated, username, loaded}`；actions `fetchStatus() / login(username, password) / logout()`；`enabled=false` 时 `authenticated` 恒 `true` 语义不需要——保持 `authenticated=false`，守卫与 UI 以 `enabled` 为准
- `AppHeader.vue`：
  - 新增齿轮按钮 → `router.push('/settings')`（PROJECT_STRUCTURE.md §2 预留的 Settings 入口）
  - auth enabled：游客显示「登录」链接（→ `/login`）；已登录显示用户名 + 退出按钮（调 logout 后回 `/`）
  - auth disabled：仅齿轮按钮，无登录态元素
- 登录成功后：`auth.fetchStatus()` 刷新 → 按 `redirect` 参数跳回（默认 `/`）

### 5.11 设置页四区块（SettingsPage.vue + 4 子组件，纵向分节）

- `ProtectedFoldersSection`：展示已保护路径（列表 + 逐项删除）；添加方式 = 文本输入相对路径 + 校验提示（复用 `INVALID_PATH` 错误码展示）；「保存」执行 PUT 全量替换；保存成功提示
- `CacheSection`：进入时 `GET /cache/stats` 展示 thumb/preview 数量与体积（人类可读 MB/GB）；「清理孤儿缓存」「清理全部缓存」均弹二次确认（复用现有 AppModal）；清理完成后刷新统计并展示 `removed_files / removed_bytes`
- `ConfigSection`：「导出配置」= 直接触发 `GET /config/export` 下载（`window.open` 或 `<a download>`）；「导入配置」= 文件选择（.json）→ POST → 成功提示导入条数，失败展示 `CONFIG_INVALID`
- `AccountSection`：auth enabled 已登录 → 用户名 + 「退出登录」；系统信息只读展示 `image_root` / `data_dir`（来自 `/auth/status` 已登录字段）；auth disabled → 提示「登录系统未启用（未设置 AUTH_PASSWORD）」+ 系统信息此时不显示
- 四区块均为独立组件，通过 props/emits 与 SettingsPage 通信，直接 fetch 禁止（走 service 层）

### 5.12 文档同步（本 Sprint 交付的一部分）

- `docs/API.md`：新增 §3.x 端点（认证 / 私有目录 / 缓存 / 导入导出，含请求响应示例）；错误码表新增 `UNAUTHORIZED(401)` / `INVALID_CREDENTIALS(401)` / `CONFIG_INVALID(400)`；通用约定补「Cookie 会话（session_token，HttpOnly）」说明
- `docs/DATA_MODEL.md`：新增 sessions、protected_folders 两表（DDL 见 §5.3）+ 列说明
- `docs/PROJECT_STRUCTURE.md`：§2 router/SettingsPage/LoginPage/components/settings/ 转正；§13 增补第 5 个 Store `auth.ts`（注明理由：认证为全局横切状态，不适合塞入 SettingsStore）
- `docs/ARCHITECTURE.md`：模块划分补 auth / appsettings / cache / config service；配置项表补 `AUTH_USERNAME` / `AUTH_PASSWORD` / `SESSION_MAX_AGE`；补一段认证与访问控制流程（游客 vs 已登录）
- `.env.example` / `docker-compose.yml`：三变量带注释示例与 environment 透传
- `README.md`：环境变量表补三变量；新增「登录系统」小节：默认关闭、如何开启、忘记凭证改 env 重启即可、HTTPS 建议 NAS 反代处理

---

## 6. 验收标准

**开发模式验证（默认方式）：`./scripts/dev.sh start`（前端 :5160 / 后端 :8080）**

兼容回归（不设 AUTH_PASSWORD）：

- Sprint 0-6 全部行为不变；`/settings` 可进入且四区块可用；`/login` 重定向 `/`；AppHeader 无登录态元素

登录场景（设 `AUTH_USERNAME=admin AUTH_PASSWORD=secret` 启动）：

- 游客：公开目录浏览正常；先在设置页将某目录（如 `/Private`）设为保护后，该目录从目录树消失；直接访问 `GET /folders?path=/Private`、其下图片与缩略图均 401；AppHeader 显示「登录」
- 登录：错误密码提示 `INVALID_CREDENTIALS`；正确凭证进入后私有目录可见、图片/缩略图正常加载（`<img>` 自动携带 cookie）；AppHeader 显示用户名
- 会话：登录 → 重启后端 → 刷新页面仍为登录态；退出后立即回到游客态（私有目录即刻消失）

设置页四区块：

- 私有目录：增删保存 → 刷新页面仍生效；非法路径提示 `INVALID_PATH`
- 缓存管理：统计与 `du` 对照一致；手动修改某原图 mtime 触发重建后（旧缓存成孤儿）执行孤儿清理 → 仅孤儿被删；全部清理后目录清空，浏览图片缓存自动重建
- 配置导入导出：导出 JSON 可下载且含 folder_settings 与 protected_folders；改动设置后导入原文件 → 设置恢复（round-trip）；导入坏 JSON 提示 `CONFIG_INVALID`

命令与测试：

```text
[后端] go build ./... 与 go test ./... 通过（含新增 4 个测试文件全绿）
[前端] pnpm build 通过
[前端] pnpm test 通过（既有 spread.test.ts 无回归）
```

---

## 7. AI 自检清单（完成后必须逐项确认）

```text
[ ] go build ./... / go test ./... / pnpm build / pnpm test 全部通过
[ ] auth_test.go 覆盖：成功 / 错误密码 / 未知用户（同错误码）/ 过期 token / 未启用拒绝
[ ] appsettings_service_test.go 覆盖：/Private 命中、/Private/a.jpg 命中、/PrivateX 不命中、空路径拒绝
[ ] cache_service_test.go 覆盖：两类孤儿均清理、正常缓存不受影响、统计准确、all 清空目录与索引
[ ] config_service_test.go 覆盖：导出→导入 round-trip 一致、坏 JSON 与 version!=1 返回 CONFIG_INVALID
[ ] 未设 AUTH_PASSWORD：Sprint 0-6 行为零回归；/login 重定向 /；/auth/status 返回 enabled:false
[ ] 设 AUTH_PASSWORD：游客列表过滤 + 直接访问 401；登录后 <img>/缩略图经 cookie 正常加载
[ ] 重启后端会话仍有效（sessions 表持久化）；退出立即失效
[ ] /settings 与管理端点在 auth enabled 时要求登录；auth disabled 时开放
[ ] vue-router 三条路由 + 守卫生效；api.ts 401 统一跳登录且登录页自身不循环跳转
[ ] 缓存两类清理均有二次确认（AppModal）；配置导入有二次确认或明确提示
[ ] /auth/status 未登录不返回 image_root/data_dir
[ ] 新增 10 个端点、2 张表、3 个 env 变量、3 个错误码与 API.md/DATA_MODEL.md/ARCHITECTURE.md 完全一致
[ ] stores/auth.ts 为唯一新增 Store；PROJECT_STRUCTURE.md §13 已同步修订
[ ] 创建/修改的文件与 §3 清单完全一致；未引入 vue-router 之外的新前端依赖、未引入标准库之外的新 Go 依赖
[ ] 未触碰 §4 禁止清单（多用户/改密码 UI/JWT/自动清理/HTTPS/Docker 迭代/favorites/主题入导出）
[ ] docker-compose.yml 仅 environment 透传三变量，未动 Dockerfile
```

---

## 8. 交付物

- 满足 §3 清单的全部代码与文档修改
- 自检结果汇报（逐项打勾）
- 开发模式验证记录：兼容回归（无 AUTH_PASSWORD）+ 登录场景（游客/登录/会话）+ 设置页四区块操作说明与结果、`go test` 与 `pnpm build/test` 输出
