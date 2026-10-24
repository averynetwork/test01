# brarchive-go

Bedrock Archive格式的Go实现 - 一个用于编码/解码文件和文件夹的简单归档格式。

[English](/README.md) / [简体中文](/README.zh-CN.md)

## 特性

- 将文件/文件夹编码为brarchive格式
- 将brarchive文件解码回原始文件/文件夹
- 支持多平台（Windows、Linux、macOS、Android、FreeBSD）
- 既可作CLI工具使用，也可作库使用
- 彩色日志输出
- 可选的文件日志记录

## 安装

### 作为CLI工具

从发布页面下载预编译的二进制文件或自行编译：

```bash
git clone https://github.com/yourusername/brarchive-go.git
cd brarchive-go
chmod +x build.sh
./build.sh
```

### 作为库

```bash
go get github.com/yourusername/brarchive-go
```

## 使用方法

### CLI使用

```bash
# 编码目录
./bin/brarchive encode ./mydir -o myarchive.brarchive

# 编码单个文件
./bin/brarchive encode ./file.txt -o file.brarchive

# 解码归档
./bin/brarchive decode ./myarchive.brarchive -o ./output

# 使用日志
./bin/brarchive --log-path ./logs encode ./mydir

# 显示帮助
./bin/brarchive --help
```

### 库使用

```go
package main

import (
    "fmt"
    "github.com/yourusername/brarchive-go/pkg/brarchive"
)

func main() {
    // 创建条目
    entries := map[string]string{
        "file1.txt": "内容1",
        "file2.txt": "内容2",
    }

    // 序列化为brarchive格式
    archive, err := brarchive.Serialize(entries)
    if err != nil {
        panic(err)
    }

    // 反序列化回来
    decoded, err := brarchive.Deserialize(archive)
    if err != nil {
        panic(err)
    }

    fmt.Printf("解码了 %d 个文件\n", len(decoded))
}
```

## 编译

使用提供的编译脚本： `build.sh`, `build.bat`

该脚本支持为多个平台和架构编译。

## 许可证

本项目采用 MPL-2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。
