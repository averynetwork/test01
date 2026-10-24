# brarchive-go

A Go implementation of Bedrock Archive format - a simple archive format for encoding/decoding files and folders.

[English](/README.md) / [简体中文](/README.zh-CN.md)

## Features

- Encode files/folders into brarchive format
- Decode brarchive files back to original files/folders
- Support for multiple platforms (Windows, Linux, macOS, Android, FreeBSD)
- Can be used as both CLI tool and library
- Colored logging output
- Optional file logging

## Installation

### As CLI Tool

Download pre-built binaries from releases or build yourself:

```bash
git clone https://github.com/yourusername/brarchive-go.git
cd brarchive-go
chmod +x build.sh
./build.sh
```

### As Library

```bash
go get github.com/yourusername/brarchive-go
```

## Usage

### CLI Usage

```bash
# Encode a directory
./bin/brarchive encode ./mydir -o myarchive.brarchive

# Encode a single file
./bin/brarchive encode ./file.txt -o file.brarchive

# Decode an archive
./bin/brarchive decode ./myarchive.brarchive -o ./output

# With logging
./bin/brarchive --log-path ./logs encode ./mydir

# Show help
./bin/brarchive --help
```

### Library Usage

```go
package main

import (
    "fmt"
    "github.com/yourusername/brarchive-go/pkg/brarchive"
)

func main() {
    // Create entries
    entries := map[string]string{
        "file1.txt": "Content 1",
        "file2.txt": "Content 2",
    }

    // Serialize to brarchive format
    archive, err := brarchive.Serialize(entries)
    if err != nil {
        panic(err)
    }

    // Deserialize back
    decoded, err := brarchive.Deserialize(archive)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Decoded %d files\n", len(decoded))
}
```

## Build

Use supplied compilation script: `build.sh`, `build.bat`

The script supports compilation for multiple platforms and schemas.

## Licenses

This project is licensed under the MPL-2.0 license - see the [LICENSE](LICENSE) file for details.
