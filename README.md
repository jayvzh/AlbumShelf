# AlbumShelf・NAS图集馆

> 一个面向 NAS / 私有服务器部署的轻量级、自托管、高性能 Web 沉浸式图集阅览器。
> 专注于图集浏览、底部缩略图 Filmstrip 导航、高级文件排序（自然排序 / Regex 排序）和双页阅读。适合写真集、漫画集、图片集等。

## 核心特性

- 📁 **文件夹浏览**：直接读取本地 / NAS 目录，无需导入
- 🖼️ **大图浏览**：Zoom / Pan / Fit / 100% / Fullscreen，快捷键完整支持
- 🎞️ **Filmstrip**：底部缩略图条，虚拟渲染 + 懒加载，10000+ 图片流畅滚动
- 📖 **双页阅读**：NeeView 式自动拼页——竖图两两拼双页、横图自动独占整屏，支持日漫右开本
- 🔀 **高级排序**：文件名 / 自然排序 / 时间 / 大小 / **Regex 提取排序 / 多规则组合排序**
- 💾 **每目录记忆**：排序与阅读设置按文件夹保存，进目录自动应用
- ⚡ **双端缓存**：预览图（长边 2560px）+ 缩略图服务器持久缓存；URL 版本化 + immutable 缓存头，重访同目录 0 网络请求；原图按需加载，重复查看直接命中浏览器缓存
- 🐳 **Docker 一键部署**：单容器，SQLite 零依赖

## 支持的图片格式

后端按扩展名白名单扫描目录（见 `backend/internal/filesystem/scanner.go`），**大小写不敏感**：

| 格式 | 扩展名 |
| --- | --- |
| JPEG | `.jpg`、`.jpeg` |
| PNG | `.png` |
| WebP | `.webp` |
| GIF | `.gif` |

- 不在上表内的文件（如 `.heic`、`.avif`、`.tiff`、`.bmp`、`.svg`）不会被扫描，目录列表与 Filmstrip 中均不显示。
- 动图（GIF）：缩略图 / 预览图取首帧，**点击加载原图时直接返回原文件**，浏览器中正常播放动画。
- 以 `.` 开头的隐藏文件与目录一律跳过。

## 技术栈

| 层 | 技术 |
| --- | --- |
| Backend | Go 1.24+ / Gin |
| Frontend | Vue 3 + TypeScript + Vite + Pinia + Tailwind CSS |
| 图片处理 | libvips (govips) |
| 数据库 | SQLite（仅配置/缓存/用户数据，图片不入库） |
| 部署 | Docker / docker compose |

## 快速开始

### Docker Compose 部署（推荐）

镜像已发布至 Docker Hub：[`jayvzh/albumshelf:latest`](https://hub.docker.com/r/jayvzh/albumshelf)。

1. 准备部署目录，创建 `docker-compose.yml`：

```yaml
services:
  albumshelf:
    image: jayvzh/albumshelf:latest
    container_name: albumshelf
    ports:
      - "${PORT:-8160}:8080"                       # 左侧宿主端口 = 浏览器访问端口，默认 8160；容器内固定 8080
    environment:
      - IMAGE_ROOT=/images                         # 容器内挂载点，勿改
      - DATA_DIR=/data                             # 容器内固定，勿改
      - PORT=8080                                  # 容器内监听端口，勿改
      - AUTH_USERNAME=${AUTH_USERNAME:-}           # 管理员用户名（示例缺省值 admin123）
      - AUTH_PASSWORD=${AUTH_PASSWORD:-}           # 管理员密码（示例缺省值 admin12345）：留空 = 游客模式
      - SESSION_MAX_AGE=${SESSION_MAX_AGE:-168h}   # 登录会话有效期
    volumes:
      - ${IMAGE_ROOT:-./images}:/images:ro         # 左侧改为你的照片目录
      - ${DATA_DIR:-./data}:/data                  # 缩略图缓存与 SQLite 数据，建议持久目录
    restart: unless-stopped
```

2. 可选：在同目录创建 `.env` 集中管理配置（不创建则使用上方默认值）：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| IMAGE_ROOT | ./images | 宿主机照片目录（只读挂载到容器 `/images`） |
| DATA_DIR | ./data | 缩略图缓存与 SQLite 数据目录（挂载到容器 `/data`） |
| PORT | 8160 | 浏览器访问端口（宿主端口，映射到容器内 8080；容器内监听端口固定 8080。开发模式 `scripts/dev.sh` 不用此变量） |
| AUTH_USERNAME | admin123 | 管理员用户名（示例缺省值，部署后请改成自己的） |
| AUTH_PASSWORD | admin12345 | 管理员密码（示例缺省值）；**置空即关闭登录系统，回到游客模式（所有接口开放）** |
| SESSION_MAX_AGE | 168h | 登录会话有效期（Go duration 格式，如 `24h`、`720h`） |

3. 启动并访问：

```bash
docker compose up -d
docker compose logs -f albumshelf    # 查看日志
```

浏览器打开 `http://<NAS_IP>:8160`（或你配置的 `PORT`）。

> **初始化引导**：首次访问若自动跳转到 `/setup`，说明图片目录尚未就绪（未挂载 / 路径不存在 / 无读取权限）。按页面指引修正 `volumes` 挂载或目录权限后，点击「重新检测」即可进入。
> 容器内置健康检查，`docker ps` 的 STATUS 显示 `(healthy)` 即服务正常。

### Docker Run（单命令替代）

```bash
docker run -d --name albumshelf \
  -p 8160:8080 \
  -v /path/to/your/photos:/images:ro \
  -v /path/to/data:/data \
  -e AUTH_USERNAME=admin123 \
  -e AUTH_PASSWORD=admin12345 \
  --restart unless-stopped \
  jayvzh/albumshelf:latest
```

### 登录系统

- 缺省凭证：`.env.example` / compose 注释给出的示例缺省值为 `admin123` / `admin12345`，**上线前请改成自己的**
- 默认关闭：`AUTH_PASSWORD` 置空（或未设置）即游客模式，行为与无登录版本一致
- 启用：设置 `AUTH_PASSWORD`（可配合 `AUTH_USERNAME`）后 `docker compose up -d` 重建容器；登录后可在设置页将目录设为「私有」，游客完全不可见
- 忘记凭证：修改 `.env` / compose 中的变量后重建容器即可
- HTTPS 建议由 NAS 侧反向代理处理

### 自行构建镜像（开发者）

```bash
make docker-build    # 构建为 jayvzh/albumshelf:latest（arm64 NAS 追加 --platform linux/arm64，见 Makefile）
make docker-push     # 推送到 Docker Hub（需先 docker login）
```

> 仓库内 `docker-compose.yml` 已为镜像形态（`jayvzh/albumshelf:latest`）；`make docker-up` 保留用于本地 build 形态验证（见 [docs/SPRINT_PLAN.md](docs/SPRINT_PLAN.md) 与 [docs/SPRINT8_TASK.md](docs/SPRINT8_TASK.md)）。

### 本地开发

```bash
# 推荐：开发模式一键管理前后端（自动清理端口冲突）
./scripts/dev.sh start       # 后端 :8080 + 前端 :5160（/api 代理到后端）
./scripts/dev.sh status      # 查看运行状态
./scripts/dev.sh restart     # 修改代码后重启
./scripts/dev.sh logs        # 查看前后端日志
./scripts/dev.sh stop        # 停止

# 或手动分开启动（另开终端）
cd backend  && go run ./cmd/server
cd frontend && pnpm install && pnpm dev    # Vite dev server 固定 :5160
```

> 开发/自测一律走开发模式（`scripts/dev.sh`），无需反复构建 Docker 镜像；
> 镜像构建与推送仅部署/发布时进行（见上方「自行构建镜像」）。

### 开发环境要求

| 工具 | 版本 | 用途 |
| --- | --- | --- |
| Go | 1.24+ | 后端编译 |
| Node | nvm node 22+ | 前端工具链 |
| pnpm | 11（`npm i -g pnpm@11`） | 前端依赖管理 |
| Docker | Docker + Compose v2 | 容器构建与部署 |
| libvips | 系统级（`libvips-dev`） | 后端缩略图/预览图生成（Sprint 3 起需要） |

构建本地产物用 `scripts/build.sh`（对应 Makefile 的 `build-backend` / `build-frontend`）。

## 项目文档

| 文档 | 内容 |
| --- | --- |
| [docs/PRD.md](docs/PRD.md) | 产品需求：背景、场景、功能需求（F001-F010） |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 技术选型与架构设计决策 |
| [docs/PROJECT_STRUCTURE.md](docs/PROJECT_STRUCTURE.md) | 目录结构、模块职责、核心数据流 |
| [docs/DATA_MODEL.md](docs/DATA_MODEL.md) | SQLite 表结构与前端类型 |
| [docs/API.md](docs/API.md) | REST API 契约（前后端必读） |
| [docs/SORT_ENGINE.md](docs/SORT_ENGINE.md) | 排序引擎：自然排序与 Regex 多规则排序 |
| [docs/SPREAD_ENGINE.md](docs/SPREAD_ENGINE.md) | 双页拼凑引擎：NeeView 見開き逻辑解读与实现设计 |
| [docs/UI_DESIGN.md](docs/UI_DESIGN.md) | 界面布局与交互设计 |
| [docs/SPRINT_PLAN.md](docs/SPRINT_PLAN.md) | Sprint 0-8 开发计划 |
| [docs/DEVELOPMENT_RULES.md](docs/DEVELOPMENT_RULES.md) | 开发规则与模块依赖约束（永久生效） |
| [docs/SPRINT0_TASK.md](docs/SPRINT0_TASK.md) | Sprint 0 AI IDE 开发任务书 |

## 开发方式

本项目按 Sprint 阶段化开发（见 SPRINT_PLAN.md），**每次只交给 AI IDE 一个 Sprint 的任务书**。
任何开发前请先阅读 DEVELOPMENT_RULES.md 中的模块依赖与边界约束。

## License

GPL-3.0
