#!/bin/bash

APP_NAME="brarchive"
BIN_DIR="./bin"

# 英文消息
EN_MESSAGES_0="Select build option:"
EN_MESSAGES_1="1) 64-bit only (default)"
EN_MESSAGES_2="2) 32-bit only"
EN_MESSAGES_3="3) Both 32-bit and 64-bit"
EN_MESSAGES_4="Enter choice [1-3]: "
EN_MESSAGES_5="Error: Go is not installed. Please install Go first."
EN_MESSAGES_6="Error: go.mod not found. Please run 'go mod init' first."
EN_MESSAGES_7="Running go mod tidy..."
EN_MESSAGES_8="go mod tidy completed successfully."
EN_MESSAGES_9="go mod tidy failed."
EN_MESSAGES_10="Building for"
EN_MESSAGES_11="Success"
EN_MESSAGES_12="Failed"
EN_MESSAGES_13="Build complete!"
EN_MESSAGES_14="Files generated:"
EN_MESSAGES_15="Size:"
EN_MESSAGES_16="Invalid choice. Using default (64-bit only)."
EN_MESSAGES_17="Setting up Go proxy (GOPROXY)..."
EN_MESSAGES_18="Using proxy"
EN_MESSAGES_19="Creating bin directory..."
EN_MESSAGES_20="Downloading dependencies..."
EN_MESSAGES_21="Warning: go mod download failed, but continuing..."
EN_MESSAGES_22="Attempting to fix dependencies..."
EN_MESSAGES_23="Created"
EN_MESSAGES_24="directory already exists"
EN_MESSAGES_25="Failed to create directory"
EN_MESSAGES_26="Files are located in:"
EN_MESSAGES_27="No files found"

# 中文消息
ZH_MESSAGES_0="选择编译选项："
ZH_MESSAGES_1="1) 仅64位（默认）"
ZH_MESSAGES_2="2) 仅32位"
ZH_MESSAGES_3="3) 32位和64位都编译"
ZH_MESSAGES_4="输入选择 [1-3]："
ZH_MESSAGES_5="错误：未安装Go。请先安装Go。"
ZH_MESSAGES_6="错误：未找到go.mod。请先运行 'go mod init'。"
ZH_MESSAGES_7="正在运行 go mod tidy..."
ZH_MESSAGES_8="go mod tidy 完成。"
ZH_MESSAGES_9="go mod tidy 失败。"
ZH_MESSAGES_10="正在编译"
ZH_MESSAGES_11="成功"
ZH_MESSAGES_12="失败"
ZH_MESSAGES_13="编译完成！"
ZH_MESSAGES_14="生成的文件："
ZH_MESSAGES_15="大小："
ZH_MESSAGES_16="无效选择，使用默认选项（仅64位）。"
ZH_MESSAGES_17="正在设置 Go 代理（GOPROXY）..."
ZH_MESSAGES_18="使用代理"
ZH_MESSAGES_19="正在创建 bin 目录..."
ZH_MESSAGES_20="正在下载依赖..."
ZH_MESSAGES_21="警告：go mod download 失败，但继续执行..."
ZH_MESSAGES_22="尝试修复依赖..."
ZH_MESSAGES_23="已创建"
ZH_MESSAGES_24="目录已存在"
ZH_MESSAGES_25="创建目录失败"
ZH_MESSAGES_26="文件位于："
ZH_MESSAGES_27="未找到文件"

select_language() {
    echo "Select language / 选择语言："
    echo "1) English"
    echo "2) 中文"
    printf "Enter 1 or 2: "
    read lang_choice
    
    if [ "$lang_choice" = "2" ]; then
        LANG_INDEX=1
    else
        LANG_INDEX=0
    fi
}

get_msg() {
    local index=$1
    if [ "$LANG_INDEX" -eq 1 ]; then
        eval echo "\$ZH_MESSAGES_$index"
    else
        eval echo "\$EN_MESSAGES_$index"
    fi
}

check_go() {
    if ! command -v go >/dev/null 2>&1; then
        echo "$(get_msg 5)"
        exit 1
    fi
}

setup_go_proxy() {
    echo "$(get_msg 17)"
    # 设置多个代理，提高下载成功率
    export GOPROXY="https://goproxy.cn,https://goproxy.io,https://proxy.golang.org,direct"
    export GO111MODULE=on
    echo "$(get_msg 18)"
}

check_gomod() {
    if [ ! -f "go.mod" ]; then
        echo "$(get_msg 6)"
        echo "Running go mod init..."
        go mod init github.com/wisebreeze/brarchive-go
    fi
}

run_gomod_tidy() {
    echo "$(get_msg 7)"
    
    echo "$(get_msg 20)"
    go mod download
    
    if [ $? -ne 0 ]; then
        echo "$(get_msg 21)"
    fi
    
    if go mod tidy; then
        echo "$(get_msg 8)"
    else
        echo "$(get_msg 9)"
        echo "$(get_msg 22)"
        
        go clean -modcache
        go mod download
        
        if go mod tidy; then
            echo "$(get_msg 8)"
        else
            echo "$(get_msg 9)"
            exit 1
        fi
    fi
}

create_bin_directory() {
    echo "$(get_msg 19)"
    if [ ! -d "$BIN_DIR" ]; then
        mkdir -p "$BIN_DIR"
        if [ $? -eq 0 ]; then
            echo "  $(get_msg 23) $BIN_DIR"
        else
            echo "  $(get_msg 25) $BIN_DIR"
            exit 1
        fi
    else
        echo "  $BIN_DIR $(get_msg 24)"
    fi
}

select_build_option() {
    echo ""
    echo "$(get_msg 0)"
    echo "$(get_msg 1)"
    echo "$(get_msg 2)"
    echo "$(get_msg 3)"
    printf "$(get_msg 4)"
    read choice
    
    if [ -z "$choice" ]; then
        choice="1"
    fi
    
    case $choice in
        1)
            BUILD_64=true
            BUILD_32=false
            ;;
        2)
            BUILD_64=false
            BUILD_32=true
            ;;
        3)
            BUILD_64=true
            BUILD_32=true
            ;;
        *)
            echo "$(get_msg 16)"
            BUILD_64=true
            BUILD_32=false
            ;;
    esac
}

build_platform() {
    local goos=$1
    local goarch=$2
    local output_name="${goos}_${goarch}"
    
    if [ "$goos" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    local output_path="${BIN_DIR}/${output_name}"
    
    echo "$(get_msg 10) $goos/$goarch..."
    
    if env GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build -ldflags="-s -w" -o "$output_path" ./cmd/brarchive; then
        echo "  $(get_msg 11)"
        if [ -f "$output_path" ]; then
            local size=$(ls -lh "$output_path" | awk '{print $5}')
            echo "  $(get_msg 15) $size"
        fi
    else
        echo "  $(get_msg 12)"
    fi
}

show_generated_files() {
    echo ""
    echo "$(get_msg 13)"
    echo "$(get_msg 14)"
    
    if [ -d "$BIN_DIR" ] && [ "$(ls -A $BIN_DIR 2>/dev/null)" ]; then
        ls -lh "$BIN_DIR" | grep -v "^total" | awk '{print "  " $9 " (" $5 ")"}'
        echo ""
        echo "$(get_msg 26) $BIN_DIR/"
    else
        echo "  $(get_msg 27)"
    fi
}

main() {
    select_language
    check_go
    setup_go_proxy
    check_gomod
    run_gomod_tidy
    create_bin_directory
    select_build_option
    
    if [ "$BUILD_64" = true ]; then
        build_platform "windows" "amd64"
        build_platform "linux" "amd64"
        build_platform "linux" "arm64"
        build_platform "darwin" "amd64"
        build_platform "darwin" "arm64"
        build_platform "android" "arm64"
        build_platform "freebsd" "amd64"
    fi
    
    if [ "$BUILD_32" = true ]; then
        build_platform "windows" "386"
        build_platform "linux" "386"
        build_platform "linux" "arm"
        build_platform "android" "arm"
        build_platform "freebsd" "386"
    fi
    
    show_generated_files
}

main