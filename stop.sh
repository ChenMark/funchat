#!/bin/bash
# FunChat 停止脚本

if [ -f /tmp/funchat.pid ]; then
    PID=$(cat /tmp/funchat.pid)
    if kill -0 "$PID" 2>/dev/null; then
        echo "[STOP] 停止 FunChat (PID=$PID)..."
        kill "$PID"
        sleep 1
        if kill -0 "$PID" 2>/dev/null; then
            echo "[STOP] 强制停止..."
            kill -9 "$PID"
        fi
        rm -f /tmp/funchat.pid
        echo "[STOP] ✅ 已停止"
    else
        echo "[STOP] 进程不存在"
        rm -f /tmp/funchat.pid
    fi
else
    echo "[STOP] 未找到 PID 文件，尝试查找进程..."
    pkill -f "bin/server" 2>/dev/null && echo "[STOP] ✅ 已停止" || echo "[STOP] 无运行中的进程"
fi
