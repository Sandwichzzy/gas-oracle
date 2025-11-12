#!/bin/bash

# 检查是否安装了 Go
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed" >&2
    exit 1
fi

# 确保 GOPATH 已设置
if [ -z "$GOPATH" ]; then
    export GOPATH=$HOME/go
    echo "GOPATH was not set, using default: $GOPATH"
fi

# 创建必要的目录
mkdir -p $GOPATH/bin

# 安装必要的 protoc 插件
echo "Installing required Go plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
if [ $? -ne 0 ]; then
    echo "Failed to install protoc-gen-go" >&2
    exit 1
fi

go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
if [ $? -ne 0 ]; then
    echo "Failed to install protoc-gen-go-grpc" >&2
    exit 1
fi

# 添加 GOBIN 到 PATH
export PATH=$PATH:$GOPATH/bin

echo "Compiling protobuf files..."
protoc -I ./ \
    --go_out=./ \
    --go-grpc_out=require_unimplemented_servers=false:. \
    services/grpc/protobuf/*.proto

if [ $? -eq 0 ]; then
    echo "Compilation completed successfully"
else
    echo "Compilation failed" >&2
    exit 1
fi

# 检查 protoc-gen-go 是否安装
#if [ ! -f "$(go env GOPATH)/bin/protoc-gen-go" ]
#then
#    echo 'Protocol Buffers plugin for Go is not installed.' >&2
#    echo 'Please install it using:' >&2
#    echo 'go install google.golang.org/protobuf/cmd/protoc-gen-go@latest' >&2
#    echo 'go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest' >&2
#    exit 1
#else
#    echo Compiling go interfaces...
#    export GO_PATH=$(go env GOPATH)
#    export GOBIN=$GO_PATH/bin
#    export PATH=$PATH:$GO_PATH/bin
#
#    protoc -I ./ --go_out=./ --go-grpc_out=require_unimplemented_servers=false:. protobuf/*.protobuf
#
#    exit_if $?
#    echo Done
#fi