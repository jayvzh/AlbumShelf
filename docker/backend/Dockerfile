# syntax=docker/dockerfile:1
# AlbumShelf・NAS图集馆 单容器镜像：Go 后端托管前端构建产物（MVP 部署形态）

# ============ 阶段 1：构建 Go 后端 ============
# 用 bookworm 变体保证 builder 与 runtime（debian:bookworm-slim）glibc 版本一致，
# 避免自编译 libvips 链接到更高版本 glibc 导致运行时加载失败。
FROM golang:1.25-bookworm AS backend-builder
WORKDIR /src/backend
# 方案 A：从源码编译裁剪版 libvips（仅 jpeg/png/webp/exif/lcms），
# 避免 debian libvips42 包拖入数十种不需要的图像格式库（约 100MB）。
# 同时安装 upx 用于压缩最终二进制（方案 D）。
RUN sed -i 's|deb.debian.org|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list.d/debian.sources \
    && unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        meson ninja-build pkg-config xz-utils \
        libglib2.0-dev libexpat1-dev \
        libjpeg62-turbo-dev libpng-dev libwebp-dev \
        libexif-dev liblcms2-dev \
    && rm -rf /var/lib/apt/lists/*
# upx 不在 bookworm 清华源中，直接下载官方预编译二进制（方案 D：压缩 Go 二进制）
RUN unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && curl -fsSL "https://gh-proxy.com/https://github.com/upx/upx/releases/download/v4.2.4/upx-4.2.4-amd64_linux.tar.xz" \
        | tar -xJ -C /tmp \
    && cp /tmp/upx-4.2.4-amd64_linux/upx /usr/local/bin/upx \
    && rm -rf /tmp/upx-4.2.4-amd64_linux
# 下载并编译裁剪版 libvips 8.15.3：auto_features=disabled 禁用全部可选格式，
# 仅启用项目实际需要的 jpeg/png/webp 输入与 exif/lcms 元数据/色彩支持。
# 通过 gh-proxy.com 镜像下载（容器内直连 github 不稳定）。
RUN unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && curl -fsSL "https://gh-proxy.com/https://github.com/libvips/libvips/releases/download/v8.15.3/vips-8.15.3.tar.xz" \
        | tar -xJ -C /tmp \
    && cd /tmp/vips-8.15.3 \
    && meson setup build --buildtype=release --prefix=/usr/local \
        -Dauto_features=disabled \
        -Djpeg=enabled -Dpng=enabled -Dwebp=enabled \
        -Dexif=enabled -Dlcms=enabled \
        -Dintrospection=disabled \
    && meson compile -C build \
    && meson install -C build \
    && rm -rf /tmp/vips-8.15.3
# 先拷贝依赖清单以利用构建缓存
COPY backend/go.mod backend/go.sum ./
# 构建阶段直连外网（清除宿主机注入的无效代理环境变量），Go 模块走国内代理
RUN unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && GOPROXY=https://goproxy.cn,direct go mod download
COPY backend/ ./
# Sprint 3 起 govips 依赖 cgo，必须 CGO_ENABLED=1（Sprint 8 修复）
# -s -w 去除调试信息，upx --best --lzma 进一步压缩体积
# PKG_CONFIG_PATH 指向自编译 libvips 的 pkgconfig 目录（meson 默认装到 lib64）
RUN PKG_CONFIG_PATH=/usr/local/lib64/pkgconfig \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
    && upx --best --lzma -q /out/server

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
# 方案 A：不安装 debian libvips42 包（会拖入数十种格式库），
# 仅安装裁剪版 libvips 运行时实际依赖的格式库，再从 builder 复制编译好的 libvips。
# 方案 D 优化：不安装 curl（healthcheck 用 Go 内置），清理 doc/man/locale。
RUN sed -i 's|deb.debian.org|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list.d/debian.sources \
    && unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libglib2.0-0 libexpat1 \
        libjpeg62-turbo libpng16-16 \
        libwebp7 libwebpdemux2 libwebpmux3 \
        libexif12 liblcms2-2 \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /usr/share/doc /usr/share/man /usr/share/locale /usr/share/info \
    && rm -rf /var/cache/apt /var/cache/debconf

WORKDIR /app
# 从 builder 复制裁剪版 libvips 运行时共享库（仅 jpeg/png/webp/exif/lcms）
# meson 默认安装到 lib64，需同步 ldconfig 搜索路径
COPY --from=backend-builder /usr/local/lib64/libvips.so* /usr/local/lib/
RUN ldconfig
COPY --from=backend-builder /out/server /app/server
COPY --from=frontend-builder /src/frontend/dist /app/dist

ENV IMAGE_ROOT=/images \
    DATA_DIR=/data \
    PORT=8080

EXPOSE 8080
# 健康检查（Sprint 8）：用 Go 二进制内置 healthcheck 子命令探测 /api/v1/health，
# 避免在运行时镜像中安装 curl（方案 D 体积优化）。
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD /app/server healthcheck || exit 1
ENTRYPOINT ["/app/server"]
