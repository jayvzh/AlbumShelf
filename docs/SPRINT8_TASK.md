# Sprint 8 开发任务书（AI IDE Task Specification）

> 用法：将本文档整体复制给 AI IDE（Cursor / Claude Code / Windsurf 等）作为 Sprint 8 的任务输入。
> Sprint 8 定位：**部署收尾 Sprint**——容器化闭环：修复 Dockerfile 构建缺陷、生产镜像健康检查、首次部署初始化引导页、docker-compose 全面转向环境变量配置（账号/密码/端口）；发布镜像为 `jayvzh/nasimageshelf:latest`（README image 版部署文档已随本任务书前置交付，见 §5.7）。

---

## 1. 任务上下文

在开始编码前，AI 必须先完整阅读以下文档：

```
docs/PRD.md（§7 成功标准——本 Sprint 验收须逐场景对照）
docs/ARCHITECTURE.md（部署形态与配置项表——本 Sprint 需同步修订）
docs/PROJECT_STRUCTURE.md（§2 页面与 service 清单——本 Sprint 需同步修订）
docs/API.md（§1 通用约定、§3 端点——新增 setup/status 端点需同步）
docs/DEVELOPMENT_RULES.md（分层红线：Handler → Service → Domain/Repository，禁止跨层）
docs/SPRINT_PLAN.md（v1.1，Sprint 8 部分）
docs/SPRINT7_TASK.md（仅作上下文：认证契约、env 变量名——其交付为本文档基线）
开发规则.md（根目录精简版：命令与自测约定）
docker/backend/Dockerfile、docker-compose.yml、Makefile、.env.example、README.md（待改文件现状）
```

### 已确认决策（不要重新设计）

| 决策点 | 结论 |
| --- | --- |
| 发布镜像 | `jayvzh/nasimageshelf:latest`（用户自行 `docker build` + `docker push` 到 Docker Hub；项目不建 CI/CD） |
| 部署配置源 | **环境变量是唯一配置源**：账号/密码/端口/路径全部经 docker-compose `environment` + `.env` 管理，不引入任何配置文件/数据库配置项 |
| 初始化流程 | **无状态检测 + 引导页**：后端只读检测 `IMAGE_ROOT` 可用性并暴露公开端点；前端守卫检测未就绪时跳 `/setup` 展示修复指引。SetupPage **不写任何配置**（否则 env 与 UI 双源冲突） |
| 容器内路径 | `IMAGE_ROOT=/images`、`DATA_DIR=/data`、`PORT=8080` 容器内固定；用户仅在 compose 的 volumes 映射与 `${PORT}` 宿主端口上做选择 |
| 运行用户 | 保持 root 运行（卷权限最简）；README 加安全说明；PUID/PGID 留 Phase 2 |

### 现状基线（Sprint 7 交付后；动手前必须先读对应文件再改）

| 现状 | 说明 |
| --- | --- |
| **Dockerfile 构建缺陷（本 Sprint 必修）** | `docker/backend/Dockerfile` L5 builder 为 `golang:1.25`，L14 仍 `CGO_ENABLED=0 go build`——govips 是 cgo 包，**当前镜像构建必然失败**（Sprint 3 引入 govips 时遗留未修，文件内注释已自我标注）；builder 未安装 `libvips-dev`/`pkg-config` |
| Dockerfile runtime | `debian:bookworm-slim` + 清华源 + `ca-certificates libvips42`；无 curl/wget，无法做 HEALTHCHECK；无 HEALTHCHECK 指令 |
| docker-compose.yml | **build 形态**（`build: docker/backend/Dockerfile`）；端口映射 `"${PORT:-8080}:8080"` 已就绪；Sprint 7 已透传 `AUTH_USERNAME`/`AUTH_PASSWORD`/`SESSION_MAX_AGE` 三变量 |
| .env.example | Sprint 7 后共 6 变量（IMAGE_ROOT/DATA_DIR/PORT/AUTH_USERNAME/AUTH_PASSWORD/SESSION_MAX_AGE），带注释；本 Sprint 不再新增变量 |
| Makefile | help/tidy/dev-*/build-*/test/lint/docker-up/docker-down/clean；`docker-up` 为 `docker compose up --build -d`；**无 docker-build/docker-push/docker-run** |
| README.md | **已前置完成 image 版部署文档重写**（compose 示例 / docker run / 6 变量环境变量表 / 初始化引导 / 登录系统 / 自行构建镜像 / 文档表 Sprint 0-8 均已就位）；本 Sprint 仅做校对同步 |
| 前端路由 | Sprint 7 后：`/`、`/settings`、`/login` 三路由 + auth 守卫 + `meta: { public: true }` 机制已就绪（`router/index.ts`）；auth store 已存在 |
| api.ts | `request<T>()` + `ApiError(code, message)`；401 统一跳 `/login` 已存在（setup/status 为公开端点，不涉及） |
| health | `GET /api/v1/health` 返回 `{"data":{"status":"ok"}}`——HEALTHCHECK 探测目标，保持不变 |
| 后端 | 无 setup 相关 handler/service；`internal/config/config.go` 6 字段（含 Sprint 7 的 AUTH 三字段），本 Sprint **不新增**配置字段 |

---

## 2. 本 Sprint 目标

**Docker 部署收尾：镜像可构建、部署可配置、首次使用有引导**

1. 修复 Dockerfile：`CGO_ENABLED=1` + builder 安装 `libvips-dev`/`pkg-config`，镜像可成功构建且缩略图功能正常
2. 镜像健康检查：runtime 加 HEALTHCHECK（探测 `/api/v1/health`），`docker ps` 可见 healthy
3. docker-compose 转向 image 形态（`jayvzh/nasimageshelf:latest`），环境变量一站式配置：端口 `${PORT}`、管理员账号 `AUTH_USERNAME`/`AUTH_PASSWORD`、会话 `SESSION_MAX_AGE`；本地构建方式以注释保留
4. 初始化引导：`GET /api/v1/setup/status`（公开、只读检测）+ 前端 `/setup` 引导页（状态清单 + 修复指引 + 重新检测），router 守卫未就绪自动跳转
5. Makefile 补 `docker-build` / `docker-push` / `docker-run`（镜像名默认 `jayvzh/nasimageshelf:latest`）
6. README 部署文档就位：image 版 compose 快速开始 / docker run 单命令 / 环境变量表 / 初始化引导 / 开发者构建说明——**已于本任务书制定时前置交付，Sprint 8 仅校对同步**（见 §5.7）

---

## 3. 允许创建 / 填充 / 修改的文件（精确清单）

> **超出此清单的任何创建/修改一律禁止。**

### A. 新建（Backend 4，含 1 个测试）

```
backend/internal/model/setup.go                 # SetupStatus{ImageRootConfigured, ImageRootExists, ImageRootReadable, AuthEnabled, Initialized}
backend/internal/service/setup_service.go       # SetupService.Status(ctx)：IMAGE_ROOT 只读检测（存在/目录/可读），不写任何状态
backend/internal/service/setup_service_test.go  # 存在目录 / 不存在 / 指向文件 / 空配置 → 各字段与 initialized 判定
backend/internal/api/handler/setup_handler.go   # GET /api/v1/setup/status（公开端点，无需登录）
```

### A'. 新建（Frontend 3）

```
frontend/src/pages/SetupPage.vue       # 初始化引导页：状态清单 + 分项修复指引（代码块）+ 重新检测 + 日志排查提示
frontend/src/services/setup.service.ts # fetchSetupStatus()（普通 request 封装）
frontend/src/types/setup.ts            # SetupStatus 接口（与后端 model 字段一一对应）
```

### B. 填充占位

无（Sprint 7 已将全部 TODO 占位转正，本 Sprint 无占位文件）。

### C. 允许修改的既有文件（附理由，改动保持最小）

```
backend/
├── internal/api/router.go               # v1 组注册 GET /setup/status（公开：不挂 RequireAuth / ProtectedAccess）
└── internal/app/app.go                  # 组装 SetupService → SetupHandler → 注入 Router（沿用现有装配模式）
部署
├── docker/backend/Dockerfile            # builder：+libvips-dev +pkg-config、CGO_ENABLED=1；runtime：+curl、HEALTHCHECK
├── docker-compose.yml                   # build 改 image: jayvzh/nasimageshelf:latest（build 块注释保留）+ environment 完整透传 + 注释完善
├── Makefile                             # +docker-build / docker-push / docker-run（IMAGE ?= jayvzh/nasimageshelf:latest）；.PHONY 同步
├── scripts/dev.sh                       # 注释更正（Sprint 7→8）+ auth 三变量导出（与 IMAGE_ROOT 同风格）+ 帮助示例补登录模式
├── README.md                            # 校对同步：与 compose 最终形态逐项核对（镜像名/6 变量/端口），有出入才修正；无出入则不动
frontend/
└── src/router/index.ts                  # +/setup 路由（meta public）+ 全局守卫前置 setup 检测（模块级 memo，仅首次导航请求）
配置 / 文档
├── docs/API.md                          # +GET /api/v1/setup/status（含响应示例、公开端点说明）
├── docs/PROJECT_STRUCTURE.md            # §2 转正 SetupPage.vue / setup.service.ts / types/setup.ts 与后端 setup 模块
└── docs/ARCHITECTURE.md                 # 部署形态补镜像/HEALTHCHECK/初始化检测流程简述（最小改动）
```

---

## 4. 禁止开发

本 Sprint **禁止**创建或实现：

```
SetupPage 内任何配置写入 / 表单提交 / "保存配置"按钮      （env 是唯一配置源，引导页仅诊断）
app_config.json / SQLite 配置表等第二配置源               （与 env 双源优先级无法自洽）
初始化向导多步骤表单（选目录/建账号等）                   （选型已定：单页诊断引导，非配置向导）
自动修复 / 自动挂载 / 自动重试循环                        （仅展示修复指引，用户手动修 compose）
PUID/PGID 非 root 运行                                   （卷权限复杂度高，Phase 2）
多容器编排（反向代理 / Redis / 数据库容器）               （单容器自足，HTTPS 由 NAS 侧处理）
CI/CD 自动构建发布流水线                                  （用户手动 make docker-build / docker-push）
多架构自动构建矩阵                                       （如需 arm64：构建命令追加 --platform linux/arm64 即可，不建矩阵）
除 Go 标准库外任何新后端依赖 / 除现有外任何新前端依赖      （检测用 os.Stat/os.Open，无第三方库）
修改 Sprint 7 已定契约（端点/错误码/env 变量名/表结构）    （本 Sprint 只读消费，不改动）
```

架构红线（分层、数据流、禁止 Component 直接 fetch 等）以 docs/DEVELOPMENT_RULES.md 与 PROJECT_STRUCTURE.md §19 为准，本文档不重复。

---

## 5. 技术要求

### 5.1 Dockerfile 修复（docker/backend/Dockerfile）

- builder 阶段：
  - 安装编译依赖：`apt-get install -y --no-install-recommends libvips-dev pkg-config`（沿用清华源 sed 写法，与 runtime 一致）
  - `RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server`（替换 L14 的 `CGO_ENABLED=0`；文件内 Sprint 0 注释一并更新）
  - builder 基础镜像保持 `golang:1.25`（debian 系，可 apt）
- runtime 阶段：
  - 追加安装 `curl`（与 ca-certificates、libvips42 同层合并，控制层数）
  - 追加 HEALTHCHECK：
    ```dockerfile
    HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
      CMD curl -fsS http://localhost:${PORT:-8080}/api/v1/health || exit 1
    ```
- 其余（三阶段结构、frontend-builder、ENV 默认值、EXPOSE 8080、ENTRYPOINT、清华源、Go 模块代理）全部保持不变
- 验证口径：`docker build` 一次成功；容器内生成缩略图正常（govips 真正生效）

### 5.2 docker-compose.yml（image 形态 + 环境变量一站式配置）

```yaml
services:
  imageshelf:
    image: jayvzh/nasimageshelf:latest
    # 自行构建镜像（开发者）：注释上行，启用下行
    # build:
    #   context: .
    #   dockerfile: docker/backend/Dockerfile
    container_name: imageshelf
    ports:
      - "${PORT:-8080}:8080"                 # 左侧宿主端口可改（如 9090）；容器内固定 8080 勿改
    environment:
      - IMAGE_ROOT=/images                   # 容器内挂载点，勿改（宿主路径在 volumes 左侧改）
      - DATA_DIR=/data                       # 容器内固定，勿改
      - PORT=8080                            # 容器内监听端口，勿改
      - AUTH_USERNAME=${AUTH_USERNAME:-}     # 管理员用户名（可空）
      - AUTH_PASSWORD=${AUTH_PASSWORD:-}     # 管理员密码：设置后启用登录系统；留空 = 关闭登录（游客模式）
      - SESSION_MAX_AGE=${SESSION_MAX_AGE:-168h}   # 会话有效期（Go duration 格式）
    volumes:
      - ${IMAGE_ROOT:-./images}:/images:ro   # 左侧改为宿主机照片目录（NAS 挂载路径）
      - ${DATA_DIR:-./data}:/data            # 缩略图缓存与 SQLite 数据（建议持久目录）
    restart: unless-stopped
```

- `AUTH_PASSWORD` 为空串与未设置等价（后端 `getEnv` fallback 语义），登录系统关闭，向后兼容成立
- 端口方案：容器内固定 8080，宿主端口经 `${PORT}` 可配——**不**把 PORT 透传进容器（避免 EXPOSE/HEALTHCHECK/反代连锁改动）
- 现有 build 形态配置以注释保留，保证开发者单文件切换

### 5.3 setup 检测服务与端点（后端）

- `SetupService.Status(ctx)` 返回 5 字段，**全部只读检测、不落盘、不缓存**：
  - `image_root_configured`：`cfg.ImageRoot != ""`
  - `image_root_exists`：`os.Stat(ImageRoot)` 成功且 `IsDir()`
  - `image_root_readable`：`os.Open` + `Readdirnames(1)` 无错误（真实读权限探测，覆盖 NAS 卷权限错误场景）
  - `auth_enabled`：`cfg.AuthPassword != ""`（仅布尔，不泄露路径）
  - `initialized`：前三者逻辑与（auth_enabled 不参与——游客模式同样可用）
- 端点：`GET /api/v1/setup/status`，**公开**（不挂 RequireAuth / ProtectedAccess）——未登录 + 配置有误时必须仍可访问，否则引导页死锁
- setup_service_test.go 必须覆盖：存在的可读目录（全 true）、不存在的路径（exists/readable false）、指向普通文件的路径（exists false）、空 IMAGE_ROOT（configured false）、initialized 仅在前三项全真时为 true

### 5.4 前端守卫与路由（router/index.ts）

- 新增 `/setup` → SetupPage，`meta: { public: true }`
- 全局守卫中、auth 检测**之前**加入 setup 检测（模块级 memo，进程内仅首次导航请求一次）：
  - `!initialized && to.path !== '/setup'` → `redirect: '/setup'`
  - `initialized && to.path === '/setup'` → `redirect: '/'`
- 检测请求走 `setup.service.ts`（不经 auth store）；请求失败（网络错误等）**不拦截导航**（放行进入应用，避免后端临时不可达时全站锁死在引导页）
- 与 Sprint 7 守卫共存：setup 前置、auth 后置，互不改写对方逻辑

### 5.5 SetupPage.vue（单页诊断引导，非配置向导）

- 标题区：「欢迎使用 AlbumShelf・NAS图集馆」+ 一句说明（服务已启动，但图片目录尚未就绪）
- 状态清单（✓/✗ 图标逐项展示，字段来自 setup/status）：
  1. 环境变量 IMAGE_ROOT 已配置
  2. 图片目录已挂载且存在
  3. 图片目录可读
  4. 信息行（非阻塞、不参与 initialized）：登录系统 已启用（用户名 AUTH_USERNAME）/ 未启用（游客模式）
- 修复指引（仅渲染未通过项对应代码块）：
  - exists ✗ → docker-compose.yml `volumes` 片段示例（宿主照片目录 → `/images:ro`）+ `docker compose up -d` 重建提示
  - readable ✗ → `chmod`/`chown` 提示 + NAS 共享文件夹权限设置说明（群晖/威联通通用表述）
  - configured ✗ → `environment: IMAGE_ROOT=/images` 片段（正常部署不会出现，防御性展示）
- 「重新检测」按钮：重调 `fetchSetupStatus()`；`initialized === true` → `router.push('/')`
- 底部排查提示：`docker compose logs imageshelf` 查看详细日志；修改 `.env`/compose 后需 `docker compose up -d` 重建生效
- 组件规范：直接 fetch 禁止（走 service 层）；样式复用现有 Tailwind 风格与 AppModal/图标用法，不引入新依赖

### 5.6 Makefile

```makefile
IMAGE ?= jayvzh/nasimageshelf:latest

docker-build: ## 构建生产镜像（$(IMAGE)；arm64 追加 --platform linux/arm64）
	docker build -f docker/backend/Dockerfile -t $(IMAGE) .

docker-push: ## 推送镜像到 Docker Hub（需先 docker login）
	docker push $(IMAGE)

docker-run: ## 本地冒烟运行生产镜像
	docker run --rm -p 8080:8080 -v $(CURDIR)/images:/images:ro -v $(CURDIR)/data:/data $(IMAGE)
```

- `.PHONY` 追加三个新目标；`IMAGE` 用 `?=` 允许命令行覆盖（如 `make docker-build IMAGE=jayvzh/nasimageshelf:1.0`）
- 既有 `docker-up`（`--build`）保留不动，注释更新为「本地 build 形态验证用」

### 5.7 README.md 校对同步（image 版部署文档已前置交付）

README 已包含以下内容（本任务书制定时完成，**非 Sprint 8 待写项**）：

- 「Docker Compose 部署（推荐）」：完整 compose 示例（`jayvzh/nasimageshelf:latest` + 6 变量透传）+ `.env` 变量表 + 启动访问步骤 + 初始化引导/健康检查提示
- 「Docker Run（单命令替代）」「登录系统」「自行构建镜像（开发者）」小节；文档表「Sprint 0-8」

Sprint 8 执行时仅做两件事：

1. 逐项核对 README 的 compose 示例与最终 `docker-compose.yml` 一致（镜像名 / 6 变量名与默认值 / `${PORT:-8080}:8080` / volumes 只读标记）
2. 核对「初始化引导」描述与 SetupPage 实际行为一致（触发条件 / 重新检测交互）；有出入才修正对应行，无出入则该文件零改动

### 5.8 文档同步

- `docs/API.md`：新增 `GET /api/v1/setup/status`（响应示例 + 「公开端点」标注 + 5 字段语义表）
- `docs/PROJECT_STRUCTURE.md`：§2 补 SetupPage.vue、setup.service.ts、types/setup.ts、后端 setup 三件套
- `docs/ARCHITECTURE.md`：部署形态补镜像与健康检查一段 + 初始化检测流程（守卫 → setup/status → /setup 引导）简述；配置项表无新增变量（勿添加）

---

## 6. 验收标准

**开发模式验证（`./scripts/dev.sh start`，前端 :5160 / 后端 :8080）**

- `GET /api/v1/setup/status` 返回 5 字段，与本地 IMAGE_ROOT 实际状态一致；`go test ./...` 全绿（含 setup_service_test.go）
- 本地 IMAGE_ROOT 有效：打开任意页面不被拦截，无 `/setup` 闪跳
- 将后端 IMAGE_ROOT 临时指向不存在路径重启：首次打开自动跳 `/setup`，修复后点「重新检测」回 `/`；`pnpm build` / `pnpm test` 通过
- 登录模式快速验证：`AUTH_USERNAME=admin AUTH_PASSWORD=secret ./scripts/dev.sh restart` → `/api/v1/setup/status` 中 `auth_enabled:true`；`AUTH_PASSWORD` 留空重启 → `auth_enabled:false`（Sprint 7 兼容回归）

**容器验证（本 Sprint 核心；镜像一次构建成型，禁止反复 `--build`）**

- `make docker-build` 构建成功（CGO 修复的直接证据）；`docker compose up -d` 后 `docker ps` 显示 `(healthy)`
- 场景 A（默认部署）：`.env` 仅配 IMAGE_ROOT 指向测试图库 → `http://localhost:8080` 正常浏览、缩略图正常生成（govips 生效）
- 场景 B（端口）：`PORT=9090` → `docker compose up -d` 重建后 `http://localhost:9090` 可访问，容器内仍 8080
- 场景 C（登录）：设 `AUTH_USERNAME/AUTH_PASSWORD` → 登录/登出/私有目录在容器内复验 Sprint 7 行为一致；`AUTH_PASSWORD` 留空 → 登录系统关闭
- 场景 D（初始化）：IMAGE_ROOT 指向不存在目录 → `up -d` → 首次访问自动落 `/setup`，状态清单 ✗、修复指引正确 → 修正挂载重建后「重新检测」进入应用
- 场景 E（PRD 成功标准）：按 docs/PRD.md §7 抽样验证（万级图库 Filmstrip 流畅、缩略图缓存命中、路径安全）
- `docker compose config` 输出合法（变量展开无告警）；`docker inspect` Health.Log 无失败记录

命令汇总：

```text
[后端] go build ./... 与 go test ./... 通过
[前端] pnpm build / pnpm test 通过
[镜像] make docker-build 成功；docker compose up -d 后容器 healthy
```

---

## 7. AI 自检清单（完成后必须逐项确认）

```text
[ ] go build ./... / go test ./... / pnpm build / pnpm test 全部通过（含 setup_service_test.go）
[ ] Dockerfile：builder 已装 libvips-dev + pkg-config 且 CGO_ENABLED=1；runtime 已装 curl 且 HEALTHCHECK 就位；其余结构未动
[ ] make docker-build 一次成功；容器内缩略图真实生成（govips 验证），docker ps 显示 (healthy)
[ ] docker-compose.yml 为 image 形态（jayvzh/nasimageshelf:latest），build 块以注释保留；environment 含 6 变量透传；端口 ${PORT:-8080}:8080
[ ] AUTH_PASSWORD 留空 = 登录关闭（Sprint 7 语义），设置后容器内登录/私有目录行为与开发模式一致
[ ] PORT 改变仅影响宿主映射，容器内 8080 与 HEALTHCHECK 不受影响
[ ] /api/v1/setup/status 公开可访问（游客 + 未初始化场景不 401）；只读检测、无任何写操作
[ ] setup_service_test.go 覆盖：可读目录 / 不存在 / 指向文件 / 空配置 / initialized 组合判定
[ ] router 守卫：未初始化自动跳 /setup、初始化后访问 /setup 回跳 /、检测请求失败放行不锁死；setup 检测先于 auth 检测且互不干扰
[ ] SetupPage 为纯诊断引导页：无任何配置写入；仅渲染未通过项的修复指引；「重新检测」通过后正确跳转
[ ] Makefile 新增 docker-build/docker-push/docker-run，IMAGE 默认 jayvzh/nasimageshelf:latest，可命令行覆盖；.PHONY 已同步
[ ] dev.sh：注释已更正（Sprint 7→8）；auth 三变量已与 IMAGE_ROOT 同风格导出；帮助示例含登录模式启动命令
[ ] README 与 compose 最终形态一致（镜像名 / 6 变量名与默认值 / ${PORT:-8080}:8080 / volumes 只读标记）；初始化引导描述与 SetupPage 实际行为一致
[ ] API.md / PROJECT_STRUCTURE.md / ARCHITECTURE.md 已按 §5.8 同步；ARCHITECTURE 配置项表未新增变量
[ ] 创建/修改的文件与 §3 清单完全一致；未引入任何新前后端依赖
[ ] 未触碰 §4 禁止清单（配置写入/第二配置源/多步骤向导/自动修复/PUID/多容器/CI/多架构/新依赖/Sprint 7 契约）
[ ] 全部文档与代码中镜像名统一为 jayvzh/nasimageshelf:latest；env 变量名与 Sprint 7 完全一致（AUTH_USERNAME/AUTH_PASSWORD/SESSION_MAX_AGE）
```

---

## 8. 交付物

- 满足 §3 清单的全部代码、Docker/compose/Makefile 与文档修改
- 自检结果汇报（逐项打勾）
- 验证记录：`make docker-build` 构建输出、`docker compose up -d` 后 `docker ps`（healthy）与 `docker compose logs`（无错误）、场景 A-E 逐一操作说明与结果、`go test` 与 `pnpm build/test` 输出
