@echo off
setlocal enabledelayedexpansion

set BIN_DIR=bin

set EN_MSG[0]=Select build option:
set EN_MSG[1]=1) 64-bit only (default)
set EN_MSG[2]=2) 32-bit only
set EN_MSG[3]=3) Both 32-bit and 64-bit
set EN_MSG[4]=Enter choice [1-3]: 
set EN_MSG[5]=Error: Go is not installed. Please install Go first.
set EN_MSG[6]=Error: go.mod not found. Please run 'go mod init' first.
set EN_MSG[7]=Running go mod tidy...
set EN_MSG[8]=go mod tidy completed successfully.
set EN_MSG[9]=go mod tidy failed.
set EN_MSG[10]=Building for
set EN_MSG[11]=Success
set EN_MSG[12]=Failed
set EN_MSG[13]=Build complete!
set EN_MSG[14]=Files generated:
set EN_MSG[15]=Size:
set EN_MSG[16]=Invalid choice. Using default (64-bit only).
set EN_MSG[17]=Setting up Go proxy (GOPROXY)...
set EN_MSG[18]=Using proxy
set EN_MSG[19]=Creating bin directory...
set EN_MSG[20]=Downloading dependencies...
set EN_MSG[21]=Warning: Download failed, but continuing...
set EN_MSG[22]=Attempting to fix dependencies...
set EN_MSG[23]=Created
set EN_MSG[24]=directory already exists
set EN_MSG[25]=Failed to create directory
set EN_MSG[26]=Files are located in:
set EN_MSG[27]=No files found

set ZH_MSG[0]=选择编译选项：
set ZH_MSG[1]=1) 仅64位（默认）
set ZH_MSG[2]=2) 仅32位
set ZH_MSG[3]=3) 32位和64位都编译
set ZH_MSG[4]=输入选择 [1-3]：
set ZH_MSG[5]=错误：未安装Go。请先安装Go。
set ZH_MSG[6]=错误：未找到go.mod。请先运行 'go mod init'。
set ZH_MSG[7]=正在运行 go mod tidy...
set ZH_MSG[8]=go mod tidy 完成。
set ZH_MSG[9]=go mod tidy 失败。
set ZH_MSG[10]=正在编译
set ZH_MSG[11]=成功
set ZH_MSG[12]=失败
set ZH_MSG[13]=编译完成！
set ZH_MSG[14]=生成的文件：
set ZH_MSG[15]=大小：
set ZH_MSG[16]=无效选择，使用默认选项（仅64位）。
set ZH_MSG[17]=正在设置 Go 代理（GOPROXY）...
set ZH_MSG[18]=使用代理
set ZH_MSG[19]=正在创建 bin 目录...
set ZH_MSG[20]=正在下载依赖...
set ZH_MSG[21]=警告：下载失败，但继续执行...
set ZH_MSG[22]=尝试修复依赖...
set ZH_MSG[23]=已创建
set ZH_MSG[24]=目录已存在
set ZH_MSG[25]=创建目录失败
set ZH_MSG[26]=文件位于：
set ZH_MSG[27]=未找到文件

:: 语言选择
:select_language
echo Select language / 选择语言：
echo 1) English
echo 2) 中文
set /p lang_choice=Enter 1 or 2: 

if "%lang_choice%"=="2" (
    set LANG_INDEX=1
) else (
    set LANG_INDEX=0
)
goto :eof

:: 获取消息
:get_msg
set MSG_INDEX=%1
if "%LANG_INDEX%"=="1" (
    echo !ZH_MSG[%MSG_INDEX%]!
) else (
    echo !EN_MSG[%MSG_INDEX%]!
)
goto :eof

:: 检查Go安装
:check_go
go version >nul 2>&1
if errorlevel 1 (
    call :get_msg 5
    exit /b 1
)
goto :eof

:: 设置Go代理
:setup_go_proxy
call :get_msg 17
set GOPROXY=https://goproxy.cn,https://goproxy.io,https://proxy.golang.org,direct
set GO111MODULE=on
call :get_msg 18
goto :eof

:: 检查go.mod
:check_gomod
if not exist go.mod (
    call :get_msg 6
    echo Running go mod init...
    go mod init github.com/wisebreeze/brarchive-go
)
goto :eof

:: 运行go mod tidy
:run_gomod_tidy
call :get_msg 7
call :get_msg 20
go mod download
if errorlevel 1 (
    call :get_msg 21
)

call :get_msg 7
go mod tidy
if errorlevel 1 (
    call :get_msg 9
    call :get_msg 22
    go clean -modcache
    go mod download
    go mod tidy
    if errorlevel 1 (
        call :get_msg 9
        exit /b 1
    )
) else (
    call :get_msg 8
)
goto :eof

:: 创建bin目录
:create_bin_directory
call :get_msg 19
if not exist "%BIN_DIR%" (
    mkdir "%BIN_DIR%"
    if errorlevel 1 (
        call :get_msg 25
        exit /b 1
    ) else (
        echo   !EN_MSG[23]! %BIN_DIR% directory
    )
) else (
    echo   %BIN_DIR% !EN_MSG[24]!
)
goto :eof

:: 选择编译选项
:select_build_option
echo.
call :get_msg 0
call :get_msg 1
call :get_msg 2
call :get_msg 3
set /p choice=!EN_MSG[4]!

if "%choice%"=="" set choice=1

if "%choice%"=="1" (
    set BUILD_64=true
    set BUILD_32=false
) else if "%choice%"=="2" (
    set BUILD_64=false
    set BUILD_32=true
) else if "%choice%"=="3" (
    set BUILD_64=true
    set BUILD_32=true
) else (
    call :get_msg 16
    set BUILD_64=true
    set BUILD_32=false
)
goto :eof

:: 编译平台
:build_platform
set GOOS=%1
set GOARCH=%2
set OUTPUT_NAME=%GOOS%_%GOARCH%

if "%GOOS%"=="windows" (
    set OUTPUT_NAME=%OUTPUT_NAME%.exe
)

set OUTPUT_PATH=%BIN_DIR%\%OUTPUT_NAME%

echo.
call :get_msg 10
echo %GOOS%/%GOARCH%...

set GOOS=%GOOS%
set GOARCH=%GOARCH%
set CGO_ENABLED=0

go build -ldflags="-s -w" -o "%OUTPUT_PATH%" ./cmd/brarchive

if errorlevel 1 (
    call :get_msg 12
) else (
    call :get_msg 11
    if exist "%OUTPUT_PATH%" (
        for %%F in ("%OUTPUT_PATH%") do set FILESIZE=%%~zF
        set /a FILESIZEKB=!FILESIZE! / 1024
        call :get_msg 15
        echo !FILESIZEKB! KB
    )
)
goto :eof

:: 显示生成的文件
:show_generated_files
echo.
call :get_msg 13
call :get_msg 14

if exist "%BIN_DIR%" (
    echo.
    call :get_msg 26
    echo %BIN_DIR%/
    echo.
    dir "%BIN_DIR%\*" 2>nul
    if errorlevel 1 (
        call :get_msg 27
    )
) else (
    call :get_msg 27
)
goto :eof

:: 主函数
:main
call :select_language
call :check_go
if errorlevel 1 exit /b 1
call :setup_go_proxy
call :check_gomod
call :run_gomod_tidy
if errorlevel 1 exit /b 1
call :create_bin_directory
call :select_build_option

if "%BUILD_64%"=="true" (
    call :build_platform windows amd64
    call :build_platform linux amd64
    call :build_platform linux arm64
    call :build_platform darwin amd64
    call :build_platform darwin arm64
    call :build_platform android arm64
    call :build_platform freebsd amd64
)

if "%BUILD_32%"=="true" (
    call :build_platform windows 386
    call :build_platform linux 386
    call :build_platform linux arm
    call :build_platform android arm
    call :build_platform freebsd 386
)

call :show_generated_files
goto :eof

:: 运行主程序
call :main
pause