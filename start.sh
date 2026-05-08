#!/bin/bash

echo "🎮 游戏修改器启动脚本"
echo "⚠️  重要提示: 此程序仅支持 Windows 系统"
echo "⚠️  请以管理员身份运行此脚本"
echo ""

# 整理依赖
echo "📦 正在整理依赖..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ 依赖整理失败，请检查 go.mod 文件"
    exit 1
fi

echo "✅ 依赖整理完成"
echo ""

# 运行程序
echo "🚀 正在启动游戏修改器..."
echo "🌐 请在浏览器中打开: http://localhost:8080"
echo "⚠️  按 Ctrl+C 停止服务器"
echo ""

go run main.go
