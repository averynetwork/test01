package archiver

import (
    "archive/tar"
    "archive/zip"
    "compress/bzip2"
    "compress/gzip"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

    "github.com/klauspost/compress/zstd"
    "github.com/pierrec/lz4/v4"
    "github.com/ulikunitz/xz"
)

// Archiver 处理各种压缩包格式
type Archiver struct {
    SupportedArchives []string
}

// NewArchiver 创建新的压缩包处理器
func NewArchiver(supported []string) *Archiver {
    return &Archiver{
        SupportedArchives: supported,
    }
}

// IsSupported 检查是否为支持的压缩包格式
func (a *Archiver) IsSupported(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    for _, supported := range a.SupportedArchives {
        if strings.EqualFold(ext, supported) {
            return true
        }
    }
    return false
}

// Extract 解压压缩包到指定目录
func (a *Archiver) Extract(archivePath, destDir string) error {
    ext := strings.ToLower(filepath.Ext(archivePath))
    
    // 处理 .mcpack 作为 zip 格式
    if ext == ".mcpack" {
        ext = ".zip"
    }
    
    switch ext {
    case ".zip":
        return a.extractZip(archivePath, destDir)
    case ".tar":
        return a.extractTar(archivePath, destDir)
    case ".gz", ".tgz":
        return a.extractTarGz(archivePath, destDir)
    case ".xz":
        return a.extractTarXz(archivePath, destDir)
    case ".zst":
        return a.extractTarZstd(archivePath, destDir)
    case ".bz2":
        return a.extractTarBz2(archivePath, destDir)
    case ".lz4":
        return a.extractTarLz4(archivePath, destDir)
    case ".7z":
        return a.extract7z(archivePath, destDir)
    default:
        return fmt.Errorf("unsupported archive format: %s", ext)
    }
}

func (a *Archiver) extractZip(zipPath, destDir string) error {
    r, err := zip.OpenReader(zipPath)
    if err != nil {
        return err
    }
    defer r.Close()

    for _, f := range r.File {
        fpath := filepath.Join(destDir, f.Name)
        
        // 安全检查：防止路径遍历攻击
        if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
            return fmt.Errorf("invalid file path: %s", fpath)
        }

        if f.FileInfo().IsDir() {
            os.MkdirAll(fpath, os.ModePerm)
            continue
        }

        if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
            return err
        }

        outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
        if err != nil {
            return err
        }

        rc, err := f.Open()
        if err != nil {
            outFile.Close()
            return err
        }

        _, err = io.Copy(outFile, rc)
        outFile.Close()
        rc.Close()
        if err != nil {
            return err
        }
    }
    return nil
}

func (a *Archiver) extractTar(tarPath, destDir string) error {
    f, err := os.Open(tarPath)
    if err != nil {
        return err
    }
    defer f.Close()

    tr := tar.NewReader(f)
    return a.extractTarFromReader(tr, destDir)
}

func (a *Archiver) extractTarGz(tarGzPath, destDir string) error {
    f, err := os.Open(tarGzPath)
    if err != nil {
        return err
    }
    defer f.Close()

    gzr, err := gzip.NewReader(f)
    if err != nil {
        return err
    }
    defer gzr.Close()

    tr := tar.NewReader(gzr)
    return a.extractTarFromReader(tr, destDir)
}

func (a *Archiver) extractTarXz(tarXzPath, destDir string) error {
    f, err := os.Open(tarXzPath)
    if err != nil {
        return err
    }
    defer f.Close()

    xzr, err := xz.NewReader(f)
    if err != nil {
        return err
    }

    tr := tar.NewReader(xzr)
    return a.extractTarFromReader(tr, destDir)
}

func (a *Archiver) extractTarZstd(tarZstdPath, destDir string) error {
    f, err := os.Open(tarZstdPath)
    if err != nil {
        return err
    }
    defer f.Close()

    zstdr, err := zstd.NewReader(f)
    if err != nil {
        return err
    }
    defer zstdr.Close()

    tr := tar.NewReader(zstdr)
    return a.extractTarFromReader(tr, destDir)
}

func (a *Archiver) extractTarBz2(tarBz2Path, destDir string) error {
    f, err := os.Open(tarBz2Path)
    if err != nil {
        return err
    }
    defer f.Close()

    bz2r := bzip2.NewReader(f)
    tr := tar.NewReader(bz2r)
    return a.extractTarFromReader(tr, destDir)
}

func (a *Archiver) extractTarLz4(tarLz4Path, destDir string) error {
    f, err := os.Open(tarLz4Path)
    if err != nil {
        return err
    }
    defer f.Close()

    lz4r := lz4.NewReader(f)
    tr := tar.NewReader(lz4r)
    return a.extractTarFromReader(tr, destDir)
}

func (a *Archiver) extractTarFromReader(tr *tar.Reader, destDir string) error {
    for {
        header, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        target := filepath.Join(destDir, header.Name)
        
        // 安全检查：防止路径遍历攻击
        if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
            return fmt.Errorf("invalid file path: %s", target)
        }

        switch header.Typeflag {
        case tar.TypeDir:
            if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
                return err
            }
        case tar.TypeReg:
            if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
                return err
            }
            outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
            if err != nil {
                return err
            }
            if _, err := io.Copy(outFile, tr); err != nil {
                outFile.Close()
                return err
            }
            outFile.Close()
        case tar.TypeSymlink:
            if err := os.Symlink(header.Linkname, target); err != nil {
                return err
            }
        }
    }
    return nil
}

func (a *Archiver) extract7z(sevenZPath, destDir string) error {
    // 7z 格式需要外部工具或更复杂的库
    // 这里返回友好的错误信息
    return fmt.Errorf("7z extraction requires external tool. Please extract manually using: 7z x %s -o%s", sevenZPath, destDir)
}

// Pack 打包目录为压缩包
func (a *Archiver) Pack(sourceDir, outputPath string, format string) error {
    ext := strings.ToLower(filepath.Ext(format))
    if ext == "" {
        ext = strings.ToLower(filepath.Ext(outputPath))
    }
    
    // 处理 .mcpack 作为 zip 格式
    if ext == ".mcpack" {
        ext = ".zip"
    }
    
    switch ext {
    case ".zip":
        return a.packZip(sourceDir, outputPath)
    case ".tar":
        return a.packTar(sourceDir, outputPath)
    case ".gz", ".tgz":
        return a.packTarGz(sourceDir, outputPath)
    case ".xz":
        return a.packTarXz(sourceDir, outputPath)
    default:
        return fmt.Errorf("unsupported pack format: %s", ext)
    }
}

func (a *Archiver) packZip(sourceDir, outputPath string) error {
    outFile, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    zipWriter := zip.NewWriter(outFile)
    defer zipWriter.Close()

    return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        relPath, err := filepath.Rel(sourceDir, path)
        if err != nil {
            return err
        }

        if relPath == "." {
            return nil
        }

        // 跳过 __brarchive 目录的父目录处理？不，需要包含
        if info.IsDir() {
            return nil
        }

        header, err := zip.FileInfoHeader(info)
        if err != nil {
            return err
        }
        header.Name = filepath.ToSlash(relPath)
        header.Method = zip.Deflate

        writer, err := zipWriter.CreateHeader(header)
        if err != nil {
            return err
        }

        file, err := os.Open(path)
        if err != nil {
            return err
        }
        defer file.Close()

        _, err = io.Copy(writer, file)
        return err
    })
}

func (a *Archiver) packTar(sourceDir, outputPath string) error {
    outFile, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    tarWriter := tar.NewWriter(outFile)
    defer tarWriter.Close()

    return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        relPath, err := filepath.Rel(sourceDir, path)
        if err != nil {
            return err
        }

        if relPath == "." {
            return nil
        }

        header, err := tar.FileInfoHeader(info, "")
        if err != nil {
            return err
        }
        header.Name = filepath.ToSlash(relPath)

        if err := tarWriter.WriteHeader(header); err != nil {
            return err
        }

        if info.IsDir() {
            return nil
        }

        file, err := os.Open(path)
        if err != nil {
            return err
        }
        defer file.Close()

        _, err = io.Copy(tarWriter, file)
        return err
    })
}

func (a *Archiver) packTarGz(sourceDir, outputPath string) error {
    outFile, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    gzWriter := gzip.NewWriter(outFile)
    defer gzWriter.Close()

    tarWriter := tar.NewWriter(gzWriter)
    defer tarWriter.Close()

    return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        relPath, err := filepath.Rel(sourceDir, path)
        if err != nil {
            return err
        }

        if relPath == "." {
            return nil
        }

        header, err := tar.FileInfoHeader(info, "")
        if err != nil {
            return err
        }
        header.Name = filepath.ToSlash(relPath)

        if err := tarWriter.WriteHeader(header); err != nil {
            return err
        }

        if info.IsDir() {
            return nil
        }

        file, err := os.Open(path)
        if err != nil {
            return err
        }
        defer file.Close()

        _, err = io.Copy(tarWriter, file)
        return err
    })
}

func (a *Archiver) packTarXz(sourceDir, outputPath string) error {
    outFile, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    xzWriter, err := xz.NewWriter(outFile)
    if err != nil {
        return err
    }
    defer xzWriter.Close()

    tarWriter := tar.NewWriter(xzWriter)
    defer tarWriter.Close()

    return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        relPath, err := filepath.Rel(sourceDir, path)
        if err != nil {
            return err
        }

        if relPath == "." {
            return nil
        }

        header, err := tar.FileInfoHeader(info, "")
        if err != nil {
            return err
        }
        header.Name = filepath.ToSlash(relPath)

        if err := tarWriter.WriteHeader(header); err != nil {
            return err
        }

        if info.IsDir() {
            return nil
        }

        file, err := os.Open(path)
        if err != nil {
            return err
        }
        defer file.Close()

        _, err = io.Copy(tarWriter, file)
        return err
    })
}
