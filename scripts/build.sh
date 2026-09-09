#!/usr/bin/env bash
# 构建前后端产物：
#   backend/bin/server    Go 单二进制
#   frontend/dist/        Vite 静态产物（由后端托管）
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> backend: go build ./cmd/server ..."
(cd backend && go build -o bin/server ./cmd/server)

echo "==> frontend: pnpm build ..."
(cd frontend && pnpm build)

echo "==> done. backend/bin/server + frontend/dist/"
