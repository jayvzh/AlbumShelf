#!/usr/bin/env bash
# AlbumShelf・NAS图集馆 开发模式管理脚本
#
# 用途：本地并行启动/管理前后端开发服务，供日常开发与 AI 迭代自测使用。
#       功能验证一律用本脚本的开发模式，无需反复构建 Docker 镜像
#       Docker 镜像优化/部署完善已在 Sprint 8 收尾完成。
#
#   backend  :8080（go run ./cmd/server）
#   frontend :5160（pnpm dev，/api 代理到 8080，端口须与 frontend/vite.config.ts 一致）
#
# 用法: ./scripts/dev.sh {start|stop|restart|status|logs} [backend|frontend]
set -euo pipefail

# ── 路径 ──
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"
FRONTEND_DIR="$PROJECT_ROOT/frontend"
LOG_DIR="$PROJECT_ROOT/logs"
PID_DIR="$PROJECT_ROOT/.pids"

# ── 初始化（日志目录和 PID 目录自动创建）──
mkdir -p "$LOG_DIR" "$PID_DIR"

# ── 本地开发默认环境（仓库内 images/ 与 data/；已显式设置则沿用）──
export IMAGE_ROOT="${IMAGE_ROOT:-$PROJECT_ROOT/images}"
export DATA_DIR="${DATA_DIR:-$PROJECT_ROOT/data}"
# 登录系统（Sprint 7）：留空 = 关闭（游客模式），设置后启用登录
export AUTH_USERNAME="${AUTH_USERNAME:-}"
export AUTH_PASSWORD="${AUTH_PASSWORD:-}"
export SESSION_MAX_AGE="${SESSION_MAX_AGE:-168h}"

# ── 颜色 ──
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ── 配置 ──
BACKEND_PORT="${PORT:-8080}"        # 后端监听端口（可被 PORT 环境变量覆盖）
FRONTEND_PORT=5160                  # 前端 Vite dev 端口（vite.config.ts 中 strictPort 固定）
BACKEND_PID_FILE="$PID_DIR/backend.pid"
FRONTEND_PID_FILE="$PID_DIR/frontend.pid"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"
BACKEND_HEALTH_URL="http://localhost:${BACKEND_PORT}/api/v1/health"

# ── 检查端口占用 ──
check_port() {
    local port=$1
    if command -v ss &>/dev/null; then
        ss -tlnp 2>/dev/null | grep -q ":$port " && return 0
    elif command -v lsof &>/dev/null; then
        lsof -i :$port -t &>/dev/null && return 0
    fi
    return 1
}

# ── 获取占用端口的 PID ──
port_pids() {
    local port=$1
    if command -v ss &>/dev/null; then
        ss -tlnp 2>/dev/null | grep ":$port " | grep -oP 'pid=\K[0-9]+' | sort -u
    elif command -v lsof &>/dev/null; then
        lsof -i :$port -t 2>/dev/null | sort -u
    fi
}

# ── 获取 PID 状态 ──
get_pid() {
    local pid_file=$1
    [ -f "$pid_file" ] && cat "$pid_file" 2>/dev/null || echo ""
}

is_running() {
    local pid=$1
    [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null
}

# ── 强制清理端口占用 ──
kill_port() {
    local name=$1
    local port=$2
    local pids=$(port_pids "$port")

    if [ -z "$pids" ]; then
        info "$name 端口 $port 无占用"
        return 0
    fi

    warn "$name 端口 $port 被 PID $(echo $pids | tr '\n' ' ')占用，强制清理..."
    for pid in $pids; do
        kill "$pid" 2>/dev/null || true
    done
    sleep 1
    for pid in $pids; do
        is_running "$pid" && kill -9 "$pid" 2>/dev/null || true
    done
    info "$name 端口 $port 已清理"
}

# ── 处理端口占用：空闲放行；占用时交互询问清理 ──
# 返回 0 = 可继续启动，1 = 中止
handle_port_conflict() {
    local name=$1
    local port=$2
    local pid_file=$3

    if ! check_port "$port"; then
        return 0
    fi

    # 端口被占用，交互询问
    warn "$name 端口 $port 已被占用"
    if [ ! -t 0 ]; then
        error "非交互模式无法询问，请用 ./scripts/dev.sh restart 强制清理后启动"
        return 1
    fi
    local answer=""
    read -rp "是否清理占用进程并继续启动？[Y/n] " answer || true
    case "$answer" in
        [Nn]*)
            warn "已取消，跳过 $name 启动"
            return 1
            ;;
        *)
            kill_port "$name" "$port"
            rm -f "$pid_file"
            ;;
    esac

    # 再次确认端口已释放
    if check_port "$port"; then
        error "$name 端口 $port 仍被占用，无法启动"
        return 1
    fi
    return 0
}

# ── 启动后端 ──
start_backend() {
    local pid=$(get_pid "$BACKEND_PID_FILE")
    if is_running "$pid"; then
        warn "后端已在运行 (PID: $pid)"
        return 0
    fi

    handle_port_conflict "后端" "$BACKEND_PORT" "$BACKEND_PID_FILE" || return 1

    info "启动后端 (Go :$BACKEND_PORT, IMAGE_ROOT=$IMAGE_ROOT)..."
    cd "$BACKEND_DIR"
    nohup go run ./cmd/server > "$BACKEND_LOG" 2>&1 &
    echo $! > "$BACKEND_PID_FILE"
    cd "$PROJECT_ROOT"

    sleep 2
    pid=$(get_pid "$BACKEND_PID_FILE")
    if ! is_running "$pid"; then
        error "后端启动失败，查看日志: $BACKEND_LOG"
        tail -20 "$BACKEND_LOG" 2>/dev/null || true
        return 1
    fi

    # 健康检查：进程存活 ≠ 应用就绪（编译/启动失败时进程可能很快退出）
    local healthy=""
    for i in $(seq 1 30); do
        if curl -sf "$BACKEND_HEALTH_URL" >/dev/null 2>&1; then
            healthy=1
            break
        fi
        if ! is_running "$pid"; then
            break
        fi
        sleep 1
    done
    if [ -n "$healthy" ]; then
        info "后端已就绪 (PID: $pid) -> http://localhost:${BACKEND_PORT}"
    else
        warn "后端进程已退出或 /api/v1/health 未就绪 (30s)，请检查日志: $BACKEND_LOG"
        tail -20 "$BACKEND_LOG" 2>/dev/null || true
        return 1
    fi
}

# ── 启动前端 ──
start_frontend() {
    local pid=$(get_pid "$FRONTEND_PID_FILE")
    if is_running "$pid"; then
        warn "前端已在运行 (PID: $pid)"
        return 0
    fi

    handle_port_conflict "前端" "$FRONTEND_PORT" "$FRONTEND_PID_FILE" || return 1

    info "启动前端 (Vite :$FRONTEND_PORT)..."
    cd "$FRONTEND_DIR"
    nohup pnpm dev > "$FRONTEND_LOG" 2>&1 &
    echo $! > "$FRONTEND_PID_FILE"
    cd "$PROJECT_ROOT"

    sleep 3
    pid=$(get_pid "$FRONTEND_PID_FILE")
    if is_running "$pid" && check_port "$FRONTEND_PORT"; then
        info "前端已就绪 (PID: $pid) -> http://localhost:${FRONTEND_PORT}"
    else
        error "前端启动失败，查看日志: $FRONTEND_LOG"
        tail -20 "$FRONTEND_LOG" 2>/dev/null || true
        return 1
    fi
}

# ── 停止进程（优雅退出）──
stop_process() {
    local name=$1
    local pid_file=$2
    local pid=$(get_pid "$pid_file")

    if [ -z "$pid" ]; then
        return 0
    fi

    if is_running "$pid"; then
        info "停止 $name (PID: $pid)..."
        kill "$pid" 2>/dev/null
        for i in $(seq 1 10); do
            is_running "$pid" || break
            sleep 0.5
        done
        if is_running "$pid"; then
            warn "$name 未响应 SIGTERM，发送 SIGKILL..."
            kill -9 "$pid" 2>/dev/null
            sleep 1
        fi
        info "$name 已停止"
    fi
    rm -f "$pid_file"
}

# ── 命令 ──
cmd_start() {
    local target="${1:-all}"
    case "$target" in
        backend|b) start_backend ;;
        frontend|f) start_frontend ;;
        all|"")
            local fail=0
            start_backend  || fail=1
            start_frontend || fail=1
            echo ""
            if [ "$fail" -eq 0 ]; then
                info "开发模式已就绪:"
                echo -e "  ${CYAN}前端${NC}: http://localhost:${FRONTEND_PORT}  (/api 代理到后端 :${BACKEND_PORT})"
                echo -e "  ${CYAN}后端${NC}: http://localhost:${BACKEND_PORT}  (health: /api/v1/health)"
                echo ""
                echo "查看日志: ./scripts/dev.sh logs | 停止: ./scripts/dev.sh stop | 重启: ./scripts/dev.sh restart"
            else
                warn "有服务启动失败，请用 ./scripts/dev.sh logs 查看日志"
            fi
            return "$fail"
            ;;
        *)
            echo "用法: ./scripts/dev.sh start [backend|frontend]"
            return 1
            ;;
    esac
}

cmd_stop() {
    local target="${1:-all}"
    info "=== 停止 AlbumShelf・NAS图集馆 开发环境 ==="
    case "$target" in
        backend|b)
            stop_process "后端" "$BACKEND_PID_FILE"
            check_port "$BACKEND_PORT" && kill_port "后端" "$BACKEND_PORT" || true
            ;;
        frontend|f)
            stop_process "前端" "$FRONTEND_PID_FILE"
            check_port "$FRONTEND_PORT" && kill_port "前端" "$FRONTEND_PORT" || true
            ;;
        all|"")
            stop_process "前端" "$FRONTEND_PID_FILE"
            stop_process "后端" "$BACKEND_PID_FILE"
            # 兜底：清理 go run / pnpm 残留子进程占用的端口
            check_port "$FRONTEND_PORT" && kill_port "前端" "$FRONTEND_PORT" || true
            check_port "$BACKEND_PORT" && kill_port "后端" "$BACKEND_PORT" || true
            ;;
        *)
            echo "用法: ./scripts/dev.sh stop [backend|frontend]"
            return 1
            ;;
    esac
    info "已停止"
}

cmd_restart() {
    info "=== 重启 AlbumShelf・NAS图集馆 开发环境 ==="
    # 强制清理占用端口（跳过确认，保证非交互可用）
    kill_port "前端" "$FRONTEND_PORT"; rm -f "$FRONTEND_PID_FILE"
    kill_port "后端" "$BACKEND_PORT"; rm -f "$BACKEND_PID_FILE"
    sleep 1
    local fail=0
    start_backend  || fail=1
    start_frontend || fail=1
    if [ "$fail" -ne 0 ]; then
        warn "有服务启动失败，请用 ./scripts/dev.sh logs 查看日志"
        return 1
    fi
    info "重启完成"
}

cmd_status() {
    echo -e "${CYAN}=== AlbumShelf・NAS图集馆 服务状态 ===${NC}"
    echo ""
    local bpid=$(get_pid "$BACKEND_PID_FILE")
    if is_running "$bpid"; then
        echo -e "  后端: ${GREEN}运行中${NC} (PID: $bpid) -> http://localhost:${BACKEND_PORT}"
    else
        echo -e "  后端: ${RED}已停止${NC}"
    fi
    local fpid=$(get_pid "$FRONTEND_PID_FILE")
    if is_running "$fpid"; then
        echo -e "  前端: ${GREEN}运行中${NC} (PID: $fpid) -> http://localhost:${FRONTEND_PORT}"
    else
        echo -e "  前端: ${RED}已停止${NC}"
    fi
    echo ""
}

cmd_logs() {
    local target="${1:-all}"
    case "$target" in
        backend|b)
            info "后端日志 (Ctrl+C 退出):"
            tail -f "$BACKEND_LOG"
            ;;
        frontend|f)
            info "前端日志 (Ctrl+C 退出):"
            tail -f "$FRONTEND_LOG"
            ;;
        all|"")
            info "合并日志 (Ctrl+C 退出):"
            tail -f "$BACKEND_LOG" "$FRONTEND_LOG"
            ;;
        *)
            echo "用法: ./scripts/dev.sh logs [backend|frontend]"
            return 1
            ;;
    esac
}

# ── 入口 ──
case "${1:-}" in
    start)   cmd_start "${2:-all}" ;;
    stop)    cmd_stop "${2:-all}" ;;
    restart) cmd_restart ;;
    status)  cmd_status ;;
    logs)    cmd_logs "${2:-all}" ;;
    *)
        echo "AlbumShelf・NAS图集馆 开发模式管理脚本"
        echo ""
        echo "用法: ./scripts/dev.sh <命令> [backend|frontend]"
        echo ""
        echo "命令:"
        echo "  start              启动前后端开发服务（如遇端口占用，提示确认后清理并启动）"
        echo "  stop               停止全部服务"
        echo "  restart            强制清理占用端口后重启全部（跳过确认）"
        echo "  status             查看服务运行状态"
        echo "  logs [target]      查看日志（默认合并，可指定 backend/frontend）"
        echo ""
        echo "端口:"
        echo "  后端: ${BACKEND_PORT} (go run ./cmd/server)"
        echo "  前端: ${FRONTEND_PORT} (Vite dev server，strictPort，/api 代理到后端)"
        echo ""
        echo "示例:"
        echo "  ./scripts/dev.sh start       # 启动前后端开发模式（默认游客模式）"
        echo "  AUTH_USERNAME=admin AUTH_PASSWORD=secret ./scripts/dev.sh restart  # 登录模式"
        echo "  ./scripts/dev.sh restart     # 改代码后强制清理端口并重启"
        echo "  ./scripts/dev.sh status      # 查看状态"
        echo "  ./scripts/dev.sh logs        # 查看合并日志"
        echo "  ./scripts/dev.sh logs backend # 仅后端日志"
        ;;
esac
