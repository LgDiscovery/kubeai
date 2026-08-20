#!/bin/bash

# KubeAI 一键部署脚本
# 用法: ./deploy.sh [up|down|restart|logs|status]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    if ! docker compose version &> /dev/null && ! docker-compose --version &> /dev/null; then
        error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    success "Docker 和 Docker Compose 已安装"
}

# 检查环境变量文件
check_env() {
    if [ ! -f .env ]; then
        warning ".env 文件不存在，从 .env.example 复制"
        cp .env.example .env
        info "请编辑 .env 文件配置必要的参数"
    fi
    success "环境变量文件已就绪"
}

# 获取 docker compose 命令
get_compose_cmd() {
    if docker compose version &> /dev/null; then
        echo "docker compose"
    else
        echo "docker-compose"
    fi
}

# 启动服务
start_services() {
    info "启动 KubeAI 服务..."
    COMPOSE_CMD=$(get_compose_cmd)
    $COMPOSE_CMD up -d
    success "服务启动完成"
    info "访问地址:"
    echo "  - 前端: http://localhost:${FRONTEND_PORT:-3000}"
    echo "  - API 网关: http://localhost:${API_GATEWAY_PORT:-8080}"
    echo "  - MinIO 控制台: http://localhost:${MINIO_CONSOLE_PORT:-9001}"
}

# 停止服务
stop_services() {
    info "停止 KubeAI 服务..."
    COMPOSE_CMD=$(get_compose_cmd)
    $COMPOSE_CMD down
    success "服务已停止"
}

# 重启服务
restart_services() {
    info "重启 KubeAI 服务..."
    stop_services
    start_services
}

# 查看日志
show_logs() {
    COMPOSE_CMD=$(get_compose_cmd)
    if [ -z "$1" ]; then
        $COMPOSE_CMD logs -f
    else
        $COMPOSE_CMD logs -f "$1"
    fi
}

# 查看状态
show_status() {
    COMPOSE_CMD=$(get_compose_cmd)
    $COMPOSE_CMD ps
}

# 健康检查
health_check() {
    info "执行健康检查..."

    # 等待服务启动
    sleep 5

    # 检查 API 网关
    if curl -s http://localhost:${API_GATEWAY_PORT:-8080}/api/v1/auth/health | grep -q "UP"; then
        success "API 网关健康检查通过"
    else
        warning "API 网关健康检查未通过"
    fi

    # 检查模型管理服务
    if curl -s http://localhost:${MODEL_MANAGER_PORT:-58080}/api/v1/model/health | grep -q "UP"; then
        success "模型管理服务健康检查通过"
    else
        warning "模型管理服务健康检查未通过"
    fi

    # 检查任务调度服务
    if curl -s http://localhost:${JOB_SCHEDULER_PORT:-58081}/api/v1/job/health | grep -q "UP"; then
        success "任务调度服务健康检查通过"
    else
        warning "任务调度服务健康检查未通过"
    fi

    # 检查推理网关服务
    if curl -s http://localhost:${INFERENCE_GATEWAY_PORT:-58082}/api/v1/inference/health | grep -q "UP"; then
        success "推理网关服务健康检查通过"
    else
        warning "推理网关服务健康检查未通过"
    fi
}

# 主函数
main() {
    echo ""
    echo "=========================================="
    echo "     KubeAI 一键部署工具"
    echo "=========================================="
    echo ""

    check_docker
    check_env

    # 加载环境变量
    if [ -f .env ]; then
        export $(cat .env | grep -v '^#' | xargs)
    fi

    case "${1:-help}" in
        up)
            start_services
            health_check
            ;;
        down)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        logs)
            show_logs "$2"
            ;;
        status)
            show_status
            ;;
        health)
            health_check
            ;;
        help|*)
            echo "用法: $0 {up|down|restart|logs|status|health}"
            echo ""
            echo "命令:"
            echo "  up       启动所有服务"
            echo "  down     停止所有服务"
            echo "  restart  重启所有服务"
            echo "  logs     查看所有服务日志"
            echo "  logs <service>  查看指定服务日志"
            echo "  status   查看服务状态"
            echo "  health   执行健康检查"
            echo "  help     显示帮助信息"
            echo ""
            ;;
    esac
}

main "$@"
