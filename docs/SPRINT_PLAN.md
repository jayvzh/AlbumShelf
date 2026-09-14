# AlbumShelf・NAS图集馆 Sprint 开发计划

> 版本：v1.1 ｜ 核心原则：**AI IDE 一次只开发一个 Sprint，禁止一次生成整个项目。**
> v1.1 增补：新增 Sprint 7（用户登录 + 设置中心），原「Docker + NAS 部署收尾」顺延为 Sprint 8。
> 每个 Sprint 开始前，将 SPRINT0_TASK.md 同风格的"任务规格"交给 AI IDE（任务上下文 / 目标 / 允许创建的文件 / 禁止项 / 验收标准 / 自检清单）。

---

## Sprint 总览

| Sprint | 主题 | 新增模块 |
| --- | --- | --- |
| 0 | Project Foundation | backend 骨架 + frontend 骨架 + Docker + Health API |
| 1 | 文件浏览 | filesystem、folder（含 Path Security） |
| 2 | 图片 Viewer | viewer 组件族、/image、/image/info |
| 3 | Filmstrip + Thumbnail | filmstrip 组件、thumbnail 模块 + 缓存 |
| 4 | 基础排序 + 文件夹设置 | sorting（filename/natural/time/size）、settings |
| 5 | Regex / 多规则排序 | regex_sorter、rule_sorter、RegexEditor + Preview |
| 6 | 双页 Spread（自动拼页） | spread 引擎（utils/spread.ts）、SpreadFrame、阅读方向设置 |
| 7 | 用户登录 + 设置中心 | auth、middleware/access、sessions/protected_folders 表、cache 管理、config 导入导出、LoginPage + SettingsPage + vue-router |
| 8 | Docker + NAS 部署收尾 | 部署优化、文档、性能验证 |
| 9 | 图片收藏（Favorites） | favorites 访问层 + /favorites API + FavoritesStore + 收藏 UI |

> 依赖关系：1 → 2 → 3 → 4 → 5 → 6 → 7 严格递进；8 收尾；9 为 Sprint 8 后追加的增量功能（依赖 1/2/7）。
> 验证节奏：Sprint 0 只保留最小 Docker 骨架；各 Sprint 的日常开发/验收一律走**开发模式** `./scripts/dev.sh start`（前端 :5160 / 后端 :8080，/api 代理到后端），**不反复构建 Docker 镜像**；Docker 镜像优化、编排完善与最终部署冒烟统一放在 **Sprint 8（收尾）**。

---

## Sprint 0：Project Foundation

**目标**：初始化项目骨架。

**任务**：

- Go Backend 最小骨架（cmd/server/main.go + internal/app + internal/config + internal/api）
- Vue3 Frontend 最小骨架（App.vue + main.ts，能显示 "AlbumShelf・NAS图集馆"）
- Docker 构建文件 + docker-compose.yml
- Config 加载（IMAGE_ROOT / DATA_DIR / PORT）
- `GET /api/v1/health` 健康检查

**完成标准**：

- `./scripts/dev.sh start` 后访问 http://localhost:5160 显示 AlbumShelf・NAS图集馆 页面，`/api/v1/health` 返回 `{"data":{"status":"ok"}}`
- （可选，仅验证 Docker 骨架存在）`make docker-up` 能构建并启动容器；镜像优化与部署完善在 Sprint 8 完成

---

## Sprint 1：文件浏览

**目标**：读取 NAS 文件夹并在前端展示目录树与图片列表。

**任务**：

- `internal/filesystem/`：filesystem.go（接口）、scanner.go、**path.go（路径安全，本 Sprint 必须完成）**、metadata.go
- **图片尺寸读取**：扫描时用 `image.DecodeConfig` 读取文件头获得 width/height（双页 Spread 的依赖，见 SPREAD_ENGINE.md §4），随 folders 接口返回
- `GET /api/v1/folders`（本 Sprint 只支持 filename 排序）
- 前端：AppLayout、FolderTree、图片列表展示（此时可以先简单网格）
- 前端：folder store + folder.service.ts + api.ts

**禁止**：创建 Spread / Favorite / Tag / Regex Editor；实现 RegexSorter。

**完成标准**：

- 点击目录树进入子目录，图片列表正确显示
- `path=../../etc/passwd` 返回 `INVALID_PATH`
- 含路径安全单元测试

---

## Sprint 2：图片 Viewer

**目标**：单图大图浏览。

**任务**：

- `GET /api/v1/image`（variant=original|preview，Sprint 2 未实现 preview 时回退返回原图；URL 版本参数 v + `Cache-Control: immutable` + ETag 缓存头）、`GET /api/v1/image/info`
- 前端 viewer 组件族：ImageViewer、**ViewerCanvas（底层复用组件，架构关键）**、ViewerToolbar、ImageInfo、LoadingOverlay
- composables：useViewer / useZoom / usePan / useKeyboard
- Zoom、Pan、Fit、100%、Fullscreen、快捷键（←/→/Space/F/0/1）
- viewer store（currentIndex / currentImage / mode）
- 本 Sprint Viewer 可暂用原图显示，图片 URL 统一走 `image.service.ts` 构造（拼 v 参数），Sprint 3 无缝切换 preview

**完成标准**：

- 点击列表图片进入大图浏览，全部快捷键生效
- 翻页流畅，LoadingOverlay 无白屏闪烁
- 图片响应带 immutable 缓存头，刷新页面命中浏览器缓存（DevTools 显示 disk/memory cache）

---

## Sprint 3：Filmstrip + Thumbnail

**目标**：底部缩略图条 + 后端缩略图服务。

**任务**：

- `internal/thumbnail/`：generator.go（govips，同时产出 thumb/preview 两种变体）、cache.go（hash 命名 + `/data/cache/{thumb,preview}/` 持久化）、validator.go（mtime/size 失效判断）
- `GET /api/v1/thumbnail?width=300`、`GET /api/v1/image?variant=preview`（长边 1920px JPEG q80），全部带版本参数 + immutable 缓存头
- image_cache 表迁移（variant 区分 thumb/preview）
- 前端：Filmstrip、ThumbnailItem、useFilmstrip
- 动态布局 + 虚拟渲染 + 懒加载 + 当前图自动居中高亮：
  - 缩略图等高不等宽（宽度按宽高比缩放 + clamp），固定间距
  - 可见数量按视口宽度动态计算（ResizeObserver + 前缀和偏移 + 二分查找，见 ARCHITECTURE.md §4.6），**禁止硬编码固定数量**
- **Viewer 切换为默认加载 preview**；ViewerToolbar 增加"加载原图"按钮（点击后加载原图并显示"原图"标识；同一张图重复点击直接命中浏览器缓存，无重新下载）

**禁止**：一次性渲染全部缩略图节点；一次性加载原图列表；硬编码可见缩略图数量或"固定项宽 × 索引"的虚拟滚动定位；Viewer 默认加载原图。

**完成标准**：

- 10000 张图片目录 Filmstrip 流畅滚动（虚拟渲染）
- 拖动改变窗口宽度，可见缩略图数量随之增减（布局动态重算），横图/竖图混排时项宽各不相同且间距一致
- 重访同一目录：缩略图/预览图全部命中浏览器缓存（DevTools Network 无请求，显示 disk cache）
- 同一张原图第二次点击"加载原图"：命中浏览器缓存，无重新下载（DevTools Network 无请求或 304/磁盘缓存）
- 服务器缓存持久化：重启容器后缩略图/预览图仍命中，不重新生成
- 修改原图后 mtime 变化 → URL 版本参数更新 → 缓存自动失效重建

---

## Sprint 4：基础排序 + 文件夹设置

**目标**：非 Regex 的全部排序 + 每目录设置记忆。

**任务**：

- `internal/sorting/`：sorter.go 接口、filename_sorter、natural_sorter、time_sorter、size_sorter（regex_sorter 仅留接口文件不实现）
- folders 接口接入 sort/direction 参数（后端排序）
- folder_settings 表 + SettingsRepository + `GET/PUT /api/v1/folder/settings`
- 前端：SortMenu、settings store、settings.service.ts
- 打开目录时自动应用该目录已保存设置

**完成标准**：

- Natural 排序：`1,10,100,11,2,20 → 1,2,10,11,20,100`
- 每目录排序设置保存后刷新仍生效
- sorting 模块单元测试通过

---

## Sprint 5：Regex / 多规则排序

**目标**：项目核心特色功能。

**任务**：

- regex_sorter.go、rule_sorter.go（多规则链式比较）
- `POST /api/v1/sort/preview`
- folder_settings 支持 regex_pattern + regex_config JSON
- 前端：RegexEditor（输入校验 + **实时 Preview 三列展示**）、SortRuleEditor
- 未匹配文件排最后策略

**完成标准**：

- `chapter(\d+)_page(\d+)` 多规则排序场景 D 验证通过
- 非法 Regex 前端即时提示 `INVALID_REGEX`
- Preview 展示每个文件的 Group 提取值与排序结果

---

## Sprint 6：双页 Spread（自动拼页）✅ 已完成

**目标**：NeeView 式双页自动拼凑的连续阅读体验。

**任务**：

- `utils/spread.ts`：SpreadEngine 纯函数（横图判定 `width > height × wideRatio`、竖图两两拼页、横图独占、封面封底单页、imageIndex↔frameIndex 映射）+ 单元测试（用例清单见 SPREAD_ENGINE.md §6）
- spread store：pageMode / readOrder / wideRatio / singleFirstPage / singleLastPage / frames
- 前端：SpreadFrame.vue（单画布渲染 1~2 张图，等高对齐、垂直居中）、SpreadControls.vue（模式切换、阅读方向、WideRatio）
- ViewerStore 翻页改为按帧推进（单页模式退化为按图推进）
- Filmstrip 联动：当前帧所有图片同时高亮；点击缩略图定位到所在帧
- 切换 Single/Spread 时当前图片不跳变
- folder_settings 接入 page_mode / read_order / wide_ratio / single_first_page / single_last_page

**禁止**：实现 Dummy Page 补位、静态奇偶配对、DividePage 拆屏（Phase 2）；做任何对比工具功能。

**完成标准**：

- 场景 E 验证通过：竖图两两拼页、横图自动独占整屏、按帧混合翻页
- right_to_left 模式下双页左右顺序正确（日漫右开本）
- 封面单页开启时第一张竖图不拼页
- spread.ts 单元测试全部通过

> **交付记录（2026-09）**：SpreadEngine（utils/spread.ts，12 Vitest 用例）+ spread/viewer store 帧化 + SpreadFrame/SpreadControls + 旋转（工具栏按钮 / KeyR）+ Filmstrip 帧级联动 + AppHeader 阅读模式入口；后端 folder_settings 五字段全链路（model/request/service/repository/handler）+ favorites 表预留；go test 与 pnpm build/test 全绿，dev.sh + settings API 冒烟通过。view_mode 按决策保留为数据库列不接入 API。

---

## Sprint 7：用户登录 + 设置中心

**目标**：单管理员登录 + 游客浏览公开目录；集中设置页管理私有目录、缓存与配置导入导出。

**任务**：

- 认证：`internal/auth/` AuthService（凭证来自环境变量 `AUTH_USERNAME` / `AUTH_PASSWORD`，constant-time 比对）、Cookie 会话（token = crypto/rand 32B hex，HttpOnly + SameSite=Lax）、sessions 表 SQLite 持久化（重启不丢）、`SESSION_MAX_AGE` 默认 168h
- 认证接口：`POST /api/v1/auth/login`、`POST /api/v1/auth/logout`、`GET /api/v1/auth/status`
- 私有目录：protected_folders 表、`GET/PUT /api/v1/protected-folders`；子树前缀匹配（`/Private` 保护自身与全部子级，不误伤 `/PrivateX`）；游客请求时目录列表过滤 + 直接访问 401
- 缓存管理：`GET /api/v1/cache/stats`（thumb/preview 数量与体积）、`POST /api/v1/cache/cleanup/orphan`（覆盖"索引中源不存在"与"磁盘有但索引无"两类孤儿）、`POST /api/v1/cache/cleanup/all`
- 配置导入导出：`GET /api/v1/config/export`（JSON 附件下载）、`POST /api/v1/config/import`（按路径 upsert 合并，非法 JSON → `CONFIG_INVALID`）
- 前端：vue-router 占位转正（`/` / `/settings` / `/login` + 路由守卫）、LoginPage、SettingsPage 四区块（私有目录 / 缓存管理 / 配置导入导出 / 账户与系统信息）、auth store（第 5 个 Store）、AppHeader 齿轮入口 + 登录态
- 兼容性：`AUTH_PASSWORD` 未设置 = 登录系统整体关闭，行为与 Sprint 6 完全一致（零配置向后兼容）
- 新错误码：UNAUTHORIZED(401)、INVALID_CREDENTIALS(401)、CONFIG_INVALID(400)

**禁止**：多用户/角色权限；修改密码 UI（凭证来自环境变量）；缓存自动清理策略/阈值；HTTPS/反向代理配置；Docker 镜像迭代（Sprint 8）；除 vue-router 外新前端依赖；除 Go 标准库外新后端依赖。

**完成标准**：

- 未配置 AUTH_PASSWORD：全部行为与现状一致
- 配置后：游客看不到私有目录（列表隐藏、直接 URL 401），登录后可见且图片/缩略图正常加载；重启后会话仍有效；退出后立即失效
- 设置页四区块逐项验证：私有目录保存刷新生效、缓存统计准确、两类清理生效、导出→导入 round-trip 一致
- `go build ./...`、`go test ./...`、`pnpm build`、`pnpm test` 全绿

---

## Sprint 8：Docker + NAS 部署收尾

**目标**：可交付部署。

**任务**：

- Docker 镜像优化（多阶段构建、govips 依赖层）
- docker-compose 完善（volume 挂载 IMAGE_ROOT 只读、DATA_DIR、.env.example，透传 AUTH_USERNAME / AUTH_PASSWORD / SESSION_MAX_AGE）
- Makefile（dev / build / test / docker）
- README 使用文档（含登录系统配置说明）
- PRD 成功标准逐条验证（10000 图目录流畅、缓存命中、路径安全、登录/私有目录）

**完成标准**：

- 全新机器 `docker compose up` 一次跑通全部用户场景 A-E 及登录/私有目录场景

---

## Sprint 9：图片收藏（Favorites）

**目标**：为图片提供跨目录的个人收藏能力（PRD F010 收藏部分提前实现；Rating/Tags 仍留 Phase 2）。

**任务**：

- 后端：favorites 表访问层（List/Add/Delete/Exists/DeleteMany；`created_at` 倒序、`file_path` 唯一去重）
- 后端：`GET /favorites`（收藏时间倒序，已删除/移动条目惰性清理）、`POST /favorites/toggle`（路径安全校验 + 存在性与格式校验后切换）；两路由挂 RequireAuth（auth enabled 时游客完全隐藏）
- 后端：配置导入导出纳入收藏——version 保持 1，导出 `favorites`（omitempty），导入 `INSERT OR IGNORE` 合并（不覆盖不删除现有收藏、不保留原时间）
- 前端：`favorite.service.ts` + `types/favorite.ts` + FavoritesStore（第 6 个 Store：`paths` Set 供 O(1) 判定、`images` 供收藏夹视图、乐观更新失败回滚、`available` 门控）
- 前端：FolderTree 顶部虚拟「收藏夹」节点（心形 + 数量角标）；FolderStore `favoritesView` 虚拟视图（不发目录请求）
- 前端：收藏夹视图内容区——数据源切换、空态引导、**强制单页帧化**（不写 folder_settings）、取消收藏实时移除
- 前端：AppHeader「只看收藏」筛选 toggle；收藏夹视图隐藏 SortMenu/阅读入口（避免向虚拟路径发请求/落库）
- 前端：ImageViewer ✕ 下方竖排心形浮动按钮组（双页帧 L/R 标记）+ S 键快捷键（双页收藏左图）+ HelpModal 说明
- 前端：网格卡片右下角心形角标（hover 显现、已收藏常亮红；避开长文件名）
- 测试：repository/service/config 导入导出 Go 单测 + favorites store vitest 单测

**禁止**：Rating/Tags UI；收藏夹视图写 folder_settings；新增前后端依赖。

**完成标准**：

- 收藏/取消在网格、查看器、收藏夹视图三处实时同步；S 键在单页/双页（含 L/R 标记）下语义正确
- auth enabled 未登录时收藏入口完全隐藏；配置导出→导入 round-trip 收藏不丢失、不重复
- `go test ./...`、`pnpm build`、`pnpm test` 全绿
