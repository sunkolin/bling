#!/bin/bash

# windows
GOOS=windows GOARCH=amd64 go build -o bling.exe main.go

# mac arm
GOOS=darwin GOARCH=arm64 go build -o bling main.go

# mac intel
GOOS=darwin GOARCH=amd64 go build -o bling main.go

# 生成压缩包
echo "正在创建压缩包..."
zip bling.zip bling bling.exe config.yaml
echo "✅ 压缩包 bling.zip 已生成"


