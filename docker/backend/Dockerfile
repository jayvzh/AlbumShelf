# syntax=docker/dockerfile:1
# AlbumShelf・NAS图集馆 单容器镜像：Go 后端托管前端构建产物（MVP 部署形态）

# ============ 阶段 1：构建 Go 后端 ============
FROM golang:1.25 AS backend-builder
WORKDIR /src/backend
# govips 为 cgo 包：builder 需安装 libvips 头文件与 pkg-config（Sprint 8 修复，同 runtime 切清华源）
RUN sed -i 's|deb.debian.org|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list.d/debian.sources \
    && unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && apt-get update \
    && apt-get install -y --no-install-recommends libvips-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*
# 先拷贝依赖清单以利用构建缓存
COPY backend/go.mod backend/go.sum ./
# 构建阶段直连外网（清除宿主机注入的无效代理环境变量），Go 模块走国内代理
RUN unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && GOPROXY=https://goproxy.cn,direct go mod download
COPY backend/ ./
# Sprint 3 起 govips 依赖 cgo，必须 CGO_ENABLED=1（Sprint 8 修复）
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ============ 阶段 2：构建前端 ============
FROM node:22-alpine AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
# 构建阶段直连外网（清除宿主机注入的无效代理环境变量）
RUN unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && npm install -g pnpm@11 \
    && pnpm install --frozen-lockfile
COPY frontend/ ./
# 仓库无 .dockerignore：宿主机 frontend/node_modules（pnpm 符号链接指向宿主 store）与 dist
# 会随 COPY 污染镜像，导致 pnpm 依赖状态检查失败，先清理再校验安装（Sprint 8 修复）
RUN unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && rm -rf node_modules dist \
    && pnpm install --frozen-lockfile \
    && pnpm build

# ============ 阶段 3：运行镜像 ============
FROM debian:bookworm-slim AS runtime
# libvips 运行库（Sprint 3 起缩略图/预览图生成需要，提前固化到依赖层）。
# 默认官方源 deb.debian.org 在部分网络不可达，这里切换到清华镜像；海外部署可改回官方源。
RUN sed -i 's|deb.debian.org|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list.d/debian.sources \
    && unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libvips42 curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=backend-builder /out/server /app/server
COPY --from=frontend-builder /src/frontend/dist /app/dist

ENV IMAGE_ROOT=/images \
    DATA_DIR=/data \
    PORT=8080

EXPOSE 8080
# 健康检查（Sprint 8）：探测 /api/v1/health，docker ps 显示 (healthy)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS http://localhost:${PORT:-8080}/api/v1/health || exit 1
ENTRYPOINT ["/app/server"]
