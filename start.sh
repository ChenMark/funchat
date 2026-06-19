#!/bin/bash
# FunChat 一键启动脚本
# 用法: ./start.sh [dev|prod]

set -e

MODE="${1:-dev}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/backend"

echo "========================================="
echo "  FunChat 后端服务启动"
echo "  模式: $MODE"
echo "========================================="

# 检查 MySQL
if ! mysqladmin ping -h 127.0.0.1 --silent 2>/dev/null; then
    echo "[INFO] 启动 MySQL..."
    sudo service mysql start 2>/dev/null || true
    sleep 2
fi

# 编译
if [ ! -f bin/server ] || [ "$MODE" = "dev" ]; then
    echo "[BUILD] 编译后端..."
    go build -o bin/server ./cmd/server/
    echo "[BUILD] ✅ 编译完成"
fi

# 加载环境变量
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# 停止旧进程
if [ -f /tmp/funchat.pid ]; then
    OLD_PID=$(cat /tmp/funchat.pid)
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "[STOP] 停止旧进程 (PID=$OLD_PID)..."
        kill "$OLD_PID"
        sleep 1
    fi
fi

# 启动
echo "[START] 启动服务..."
nohup ./bin/server > /tmp/funchat.log 2>&1 &
PID=$!
echo $PID > /tmp/funchat.pid
sleep 2

# 验证
if kill -0 "$PID" 2>/dev/null; then
    echo "[START] ✅ 服务已启动 (PID=$PID)"
    echo "[START] 日志: tail -f /tmp/funchat.log"
    echo "[START] 健康检查:"
    curl -s http://localhost:8080/health | python3 -m json.tool 2>/dev/null || echo "  (等待服务就绪...)"
else
    echo "[START] ❌ 启动失败，查看日志: cat /tmp/funchat.log"
    exit 1
fi

echo ""
echo "========================================="
echo "  FunChat 已就绪: http://localhost:8080"
echo "  停止: ./stop.sh"
echo "========================================="
