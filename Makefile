# AlbumShelf・NAS图集馆 常用开发命令
# 用法：make <target> ｜ 详细说明见 docs/SPRINT_PLAN.md 与 README.md

.PHONY: help tidy dev-backend dev-frontend build-backend build-frontend test lint docker-build docker-push docker-run docker-up docker-down clean

IMAGE ?= jayvzh/nasimageshelf:latest

help: ## 显示可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

tidy: ## 整理 Go 依赖
	cd backend && go mod tidy

dev-backend: ## 本地启动 Go 后端（默认 :8080）
	cd backend && go run ./cmd/server

dev-frontend: ## 本地启动前端开发服务器（pnpm dev，Vite :5160，/api 代理到 :8080）
	cd frontend && pnpm dev

build-backend: ## 编译 Go 后端
	cd backend && go build -o bin/server ./cmd/server

build-frontend: ## 构建前端产物
	cd frontend && pnpm build

test: ## 运行后端测试
	cd backend && go test ./...

lint: ## 后端静态检查（需 golangci-lint）
	cd backend && golangci-lint run

docker-build: ## 构建生产镜像（$(IMAGE)；arm64 追加 --platform linux/arm64）
	docker build -f docker/backend/Dockerfile -t $(IMAGE) .

docker-push: ## 推送镜像到 Docker Hub（需先 docker login）
	docker push $(IMAGE)

docker-run: ## 本地冒烟运行生产镜像
	docker run --rm -p 8160:8080 -v $(CURDIR)/images:/images:ro -v $(CURDIR)/data:/data $(IMAGE)

docker-up: ## Docker 启动（本地 build 形态验证用；image 形态直接 docker compose up -d）
	docker compose up --build -d

docker-down: ## 停止并移除容器
	docker compose down

clean: ## 清理构建产物
	rm -rf backend/bin frontend/dist
