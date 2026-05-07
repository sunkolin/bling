#!/bin/bash

# windows
GOOS=windows GOARCH=amd64 go build -o sun.exe main.go

# mac arm
GOOS=darwin GOARCH=arm64 go build -o sun main.go

# mac intel
GOOS=darwin GOARCH=amd64 go build -o sun main.go

# 生成压缩包
echo "正在创建压缩包..."
zip sun.zip sun sun.exe config.yaml
echo "✅ 压缩包 sun.zip 已生成"


