package main

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    "strings"

    "github.com/urfave/cli/v2"
    
    "github.com/wisebreeze/brarchive-go/internal/logger"
    "github.com/wisebreeze/brarchive-go/pkg/archiver"
    "github.com/wisebreeze/brarchive-go/pkg/brarchive"
    "github.com/wisebreeze/brarchive-go/pkg/config"
)

// 全局配置变量，在 beforeAction 中设置
var globalConfig *config.Config

func main() {
    app := &cli.App{
        Name:     "brarchive",
        Usage:    "Bedrock Archive tool - Encode and decode files/folders to/from brarchive format",
        Version:  "1.0.0",
        Authors: []*cli.Author{
            {
                Name:  "brarchive-go",
                Email: "your.email@example.com",
            },
        },
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "log-path",
                Aliases: []string{"l"},
                Usage:   "Path to store log files",
            },
            &cli.StringFlag{
                Name:    "config",
                Aliases: []string{"c"},
                Usage:   "Path to TOML configuration file (default: ./config.toml)",
            },
        },
        Before: beforeAction,
        Commands: []*cli.Command{
            {
                Name:    "encode",
                Aliases: []string{"compress", "e"},
                Usage:   "Encode a given folder or single file into the bedrock-archive format",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:    "output",
                        Aliases: []string{"o", "out"},
                        Usage:   "Output file path (optional)",
                    },
                },
                Action: encodeAction,
            },
            {
                Name:    "decode",
                Aliases: []string{"decompress", "d"},
                Usage:   "Decode a given file from the bedrock-archive format into a folder or single file",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:    "output",
                        Aliases: []string{"o", "out"},
                        Usage:   "Output directory path (optional)",
                    },
                },
                Action: decodeAction,
            },
            {
                Name:    "init-config",
                Aliases: []string{"init"},
                Usage:   "Generate a default configuration file",
                Action: initConfigAction,
            },
        },
    }

    if err := app.Run(os.Args); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

// beforeAction 在所有命令之前运行
func beforeAction(c *cli.Context) error {
    // 确定配置文件路径
    configPath := c.String("config")
    
    // 如果没有指定配置文件，使用默认路径 ./config.toml
    if configPath == "" {
        configPath = "./config.toml"
        
        // 检查默认配置文件是否存在
        if _, err := os.Stat(configPath); os.IsNotExist(err) {
            // 配置文件不存在，使用内置默认配置
            fmt.Fprintf(os.Stderr, "Config file %s not found, using built-in default configuration\n", configPath)
            
            // 使用内置默认配置
            cfg := config.DefaultConfig()
            globalConfig = cfg
            
            // 初始化日志器
            logPath := c.String("log-path")
            if err := logger.InitWithConfig(cfg, logPath); err != nil {
                return fmt.Errorf("failed to initialize logger: %v", err)
            }
            
            return nil
        }
    }
    
    // 配置文件存在，加载它
    cfg, err := config.LoadConfig(configPath)
    if err != nil {
        return fmt.Errorf("failed to load config from %s: %v", configPath, err)
    }
    globalConfig = cfg

    // 初始化日志器
    logPath := c.String("log-path")
    if err := logger.InitWithConfig(cfg, logPath); err != nil {
        return fmt.Errorf("failed to initialize logger: %v", err)
    }

    return nil
}

// initConfigAction 生成默认配置文件
func initConfigAction(c *cli.Context) error {
    // 确定配置文件路径
    configPath := c.String("config")
    if configPath == "" {
        configPath = "./config.toml"
    }
    
    // 检查文件是否已存在
    if _, err := os.Stat(configPath); err == nil {
        return fmt.Errorf("config file %s already exists", configPath)
    }

    // 保存默认配置
    if err := config.SaveDefaultConfig(configPath); err != nil {
        return fmt.Errorf("failed to save config file: %v", err)
    }

    fmt.Printf("Default configuration file created at: %s\n", configPath)
    
    // 显示配置内容示例
    fmt.Println("\nConfiguration example:")
    fmt.Println("[log]")
    fmt.Println("level = \"info\"      # silent, error, warn, info, debug")
    fmt.Println("color = true          # enable colored output")
    fmt.Println("timestamp = true      # show timestamps")
    fmt.Println("# file = \"./logs/brarchive.log\"  # optional log file")
    
    return nil
}

func encodeAction(c *cli.Context) error {
    startTime := time.Now()
    
    if c.NArg() < 1 {
        return fmt.Errorf("input path is required")
    }
    
    inputPath := c.Args().Get(0)
    
    // 获取输出路径
    outputPath := c.String("output")
    
    // 如果命令行没有指定，手动解析参数
    if outputPath == "" {
        for i, arg := range os.Args {
            if arg == "-o" || arg == "--out" || arg == "--output" {
                if i+1 < len(os.Args) {
                    outputPath = os.Args[i+1]
                    break
                }
            } else if strings.HasPrefix(arg, "-o=") || strings.HasPrefix(arg, "--out=") || strings.HasPrefix(arg, "--output=") {
                parts := strings.SplitN(arg, "=", 2)
                if len(parts) == 2 {
                    outputPath = parts[1]
                    break
                }
            }
        }
    }
    
    logger.Debug("Input path: %s", inputPath)
    logger.Debug("Output path: '%s'", outputPath)
    
    // 处理输出路径
    if outputPath == "" {
        outputPath = extractFileName(inputPath)
        if outputPath == "" {
            outputPath = "brarchive"
        }
        logger.Debug("No output specified, using default: %s", outputPath)
    }
    
    outputPath = addExtensionIfMissing(outputPath, "brarchive")
    logger.Debug("Final output path: %s", outputPath)
    
    logger.Info("Beginning to encode \"%s\" into \"%s\" archive", inputPath, outputPath)
    
    // Check input path exists
    if _, err := os.Stat(inputPath); os.IsNotExist(err) {
        return fmt.Errorf("input path \"%s\" does not exist", inputPath)
    }
    
    // 确保输出目录存在
    outputDir := filepath.Dir(outputPath)
    if outputDir != "" && outputDir != "." {
        if err := os.MkdirAll(outputDir, 0755); err != nil {
            return fmt.Errorf("failed to create output directory: %v", err)
        }
    }
    
    // Check if output file already exists
    if _, err := os.Stat(outputPath); err == nil {
        return fmt.Errorf("output file \"%s\" already exists", outputPath)
    }
    
    // 检查输入是否为支持的压缩包格式
    if globalConfig.IsSupportedArchive(inputPath) {
        logger.Info("Detected supported archive format, processing as archive...")
        return encodeArchiveAction(inputPath, outputPath, startTime)
    }
    
    // 原有逻辑：处理普通文件/文件夹
    // Process input path
    entriesMap := make(map[string]string)
    
    fileInfo, err := os.Stat(inputPath)
    if err != nil {
        return fmt.Errorf("failed to stat input path: %v", err)
    }
    
    if fileInfo.IsDir() {
        files, err := os.ReadDir(inputPath)
        if err != nil {
            return fmt.Errorf("failed to read directory: %v", err)
        }
        
        for _, file := range files {
            if file.IsDir() {
                continue
            }
            
            filePath := filepath.Join(inputPath, file.Name())
            content, err := os.ReadFile(filePath)
            if err != nil {
                return fmt.Errorf("failed to read file %s: %v", filePath, err)
            }
            
            entriesMap[file.Name()] = string(content)
        }
    } else {
        content, err := os.ReadFile(inputPath)
        if err != nil {
            return fmt.Errorf("failed to read file: %v", err)
        }
        
        fileName := filepath.Base(inputPath)
        entriesMap[fileName] = string(content)
    }
    
    logger.Debug("Read %d files from input", len(entriesMap))
    
    archive, err := brarchive.Serialize(entriesMap)
    if err != nil {
        return fmt.Errorf("failed to encode directory entries: %v", err)
    }
    
    logger.Debug("Serialized archive size: %d bytes", len(archive))
    
    if err := os.WriteFile(outputPath, archive, 0644); err != nil {
        return fmt.Errorf("failed to write archive to %s: %v", outputPath, err)
    }
    
    fileInfo, _ = os.Stat(outputPath)
    logger.Info("Successfully encoded %d files (%d bytes) to %s in %v!", 
        len(entriesMap), fileInfo.Size(), outputPath, time.Since(startTime))
    
    return nil
}

// encodeArchiveAction 处理压缩包文件的编码
func encodeArchiveAction(inputPath, outputPath string, startTime time.Time) error {
    // 创建临时目录
    tempDir, err := os.MkdirTemp("", "brarchive_*")
    if err != nil {
        return fmt.Errorf("failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)
    
    logger.Info("Extracting archive to temporary directory: %s", tempDir)
    
    // 解压压缩包
    archiverInstance := archiver.NewArchiver(globalConfig.Archive.SupportedArchives)
    if err := archiverInstance.Extract(inputPath, tempDir); err != nil {
        return fmt.Errorf("failed to extract archive: %v", err)
    }
    
    // 创建输出目录
    outputDirName := globalConfig.Archive.OutputDir
    brarchiveDir := filepath.Join(tempDir, outputDirName)
    if err := os.MkdirAll(brarchiveDir, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %v", err)
    }
    
    // 处理文件夹
    processor := archiver.NewProcessor(globalConfig)
    processedCount, err := processor.ProcessFolder(tempDir, brarchiveDir)
    if err != nil {
        return fmt.Errorf("failed to process folder: %v", err)
    }
    
    logger.Info("Processed %d folders into brarchive files", processedCount)
    
    // 重新打包
    logger.Info("Repacking archive...")
    
    // 确定最终输出路径
    var finalOutputPath string
    
    // 获取原文件名（不含路径和扩展名）
    originalBaseName := filepath.Base(inputPath)
    originalExt := filepath.Ext(originalBaseName)
    originalName := strings.TrimSuffix(originalBaseName, originalExt)
    
    if outputPath == "" || outputPath == originalName+".brarchive" {
        // 如果没有指定输出路径或输出路径是默认的 .brarchive，则使用原压缩包名
        finalOutputPath = inputPath
    } else {
        // 使用用户指定的输出路径
        finalOutputPath = outputPath
    }
    
    // 确保输出目录存在
    outputDir := filepath.Dir(finalOutputPath)
    if outputDir != "" && outputDir != "." {
        if err := os.MkdirAll(outputDir, 0755); err != nil {
            return fmt.Errorf("failed to create output directory: %v", err)
        }
    }
    
    // 处理文件冲突 - 使用 Pack.OnConflict 配置
    if _, err := os.Stat(finalOutputPath); err == nil {
        switch globalConfig.Pack.OnConflict {
        case "rename":
            finalOutputPath = getRenamedPath(finalOutputPath, globalConfig.Pack.RenamePattern)
            logger.Warn("Output file exists, using renamed: %s", finalOutputPath)
        case "replace":
            logger.Warn("Overwriting existing file: %s", finalOutputPath)
            // 删除原文件
            if err := os.Remove(finalOutputPath); err != nil {
                return fmt.Errorf("failed to remove existing file: %v", err)
            }
        default:
            return fmt.Errorf("output file already exists: %s", finalOutputPath)
        }
    }
    
    // 重新打包
    ext := filepath.Ext(inputPath)
    if err := archiverInstance.Pack(tempDir, finalOutputPath, ext); err != nil {
        return fmt.Errorf("failed to pack archive: %v", err)
    }
    
    // 根据配置决定是否删除输入文件
    // 注意：DeleteInput 控制是否删除原压缩包
    if globalConfig.Pack.DeleteInput {
        // 如果输出路径就是输入路径（即覆盖原文件），则不需要额外删除
        if finalOutputPath != inputPath {
            if err := os.Remove(inputPath); err != nil {
                logger.Warn("Failed to delete input file: %v", err)
            } else {
                logger.Info("Deleted input file: %s", inputPath)
            }
        }
    }
    
    fileInfo, _ := os.Stat(finalOutputPath)
    logger.Info("Successfully processed archive: %s (%d bytes) in %v!", 
        finalOutputPath, fileInfo.Size(), time.Since(startTime))
    
    return nil
}

func decodeAction(c *cli.Context) error {
    startTime := time.Now()
    
    if c.NArg() < 1 {
        return fmt.Errorf("input file is required")
    }
    
    inputPath := c.Args().Get(0)
    
    // 获取输出路径
    outputPath := c.String("output")
    
    // 如果命令行没有指定，手动解析参数
    if outputPath == "" {
        for i, arg := range os.Args {
            if arg == "-o" || arg == "--out" || arg == "--output" {
                if i+1 < len(os.Args) {
                    outputPath = os.Args[i+1]
                    break
                }
            } else if strings.HasPrefix(arg, "-o=") || strings.HasPrefix(arg, "--out=") || strings.HasPrefix(arg, "--output=") {
                parts := strings.SplitN(arg, "=", 2)
                if len(parts) == 2 {
                    outputPath = parts[1]
                    break
                }
            }
        }
    }
    
    logger.Debug("Input path: %s", inputPath)
    logger.Debug("Output path: '%s'", outputPath)
    
    logger.Info("Beginning to decode archive \"%s\" into \"%s\"", inputPath, outputPath)
    
    // Check input file exists
    if _, err := os.Stat(inputPath); os.IsNotExist(err) {
        return fmt.Errorf("input file \"%s\" does not exist", inputPath)
    }
    
    // 检查输入是否为支持的压缩包格式
    if globalConfig.IsSupportedArchive(inputPath) {
        logger.Info("Detected supported archive format, processing archive decode...")
        return decodeArchiveAction(inputPath, outputPath, startTime)
    }
    
    // 处理输出路径
    if outputPath == "" {
        outputPath = extractFileName(inputPath)
        if outputPath == "" {
            outputPath = "brarchive_output"
        }
        logger.Debug("No output specified, using default: %s", outputPath)
    }
    
    // 原有逻辑：处理单个 .brarchive 文件
    data, err := os.ReadFile(inputPath)
    if err != nil {
        return fmt.Errorf("failed to read archive file: %v", err)
    }
    
    logger.Info("Read %d bytes from archive file", len(data))
    
    archive, err := brarchive.Deserialize(data)
    if err != nil {
        logger.Error("Failed to decode: %v", err)
        if len(data) > 0 {
            logger.Debug("First 50 bytes of file: %x", data[:min(50, len(data))])
        }
        return fmt.Errorf("failed to decode archive file: %v", err)
    }
    
    logger.Debug("Decoded %d files from archive", len(archive))
    
    // Check if output directory exists
    if _, err := os.Stat(outputPath); err == nil {
        files, err := os.ReadDir(outputPath)
        if err != nil {
            return fmt.Errorf("failed to read output directory: %v", err)
        }
        if len(files) > 0 {
            return fmt.Errorf("output directory \"%s\" already exists and is not empty", outputPath)
        }
    } else if os.IsNotExist(err) {
        if err := os.MkdirAll(outputPath, 0755); err != nil {
            return fmt.Errorf("failed to create output directory: %v", err)
        }
        logger.Info("Successfully created output directory \"%s\"", outputPath)
    } else {
        return fmt.Errorf("failed to check output directory: %v", err)
    }
    
    for fileName, content := range archive {
        filePath := filepath.Join(outputPath, fileName)
        if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
            return fmt.Errorf("failed to write output file %s: %v", filePath, err)
        }
        logger.Info("Successfully decoded %s (%d bytes)", fileName, len(content))
    }
    
    logger.Info("Successfully decoded %d files to %s in %v!", len(archive), outputPath, time.Since(startTime))
    return nil
}

// decodeArchiveAction 处理压缩包的解码
func decodeArchiveAction(inputPath, outputPath string, startTime time.Time) error {
    // 创建临时目录
    tempDir, err := os.MkdirTemp("", "brarchive_decode_*")
    if err != nil {
        return fmt.Errorf("failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)
    
    logger.Info("Extracting archive to temporary directory: %s", tempDir)
    
    // 解压压缩包
    archiverInstance := archiver.NewArchiver(globalConfig.Archive.SupportedArchives)
    if err := archiverInstance.Extract(inputPath, tempDir); err != nil {
        return fmt.Errorf("failed to extract archive: %v", err)
    }
    
    // 查找所有 __brarchive 文件夹（包括 subpacks 下的）
    brarchiveDirs, err := findBrarchiveDirs(tempDir)
    if err != nil {
        return fmt.Errorf("failed to find brarchive directories: %v", err)
    }
    
    if len(brarchiveDirs) == 0 {
        logger.Warn("No __brarchive folders found in archive, nothing to decode")
        // 如果没有 __brarchive 文件夹，直接复制整个解压内容到输出目录
        finalOutputPath := getDecodeOutputPath(inputPath, outputPath)
        if err := copyDir(tempDir, finalOutputPath); err != nil {
            return fmt.Errorf("failed to copy extracted content: %v", err)
        }
        logger.Info("Copied extracted content to %s", finalOutputPath)
        return nil
    }
    
    logger.Info("Found %d __brarchive folder(s), decoding brarchive files...", len(brarchiveDirs))
    
    // 解码所有 __brarchive 文件夹中的 .brarchive 文件
    for _, brarchiveDir := range brarchiveDirs {
        if err := decodeBrarchiveFiles(brarchiveDir, tempDir); err != nil {
            return fmt.Errorf("failed to decode brarchive files in %s: %v", brarchiveDir, err)
        }
        
        // 删除 __brarchive 文件夹
        if err := os.RemoveAll(brarchiveDir); err != nil {
            logger.Warn("Failed to remove __brarchive folder %s: %v", brarchiveDir, err)
        } else {
            logger.Info("Removed __brarchive folder: %s", brarchiveDir)
        }
    }
    
    // 确定最终输出路径
    finalOutputPath := getDecodeOutputPath(inputPath, outputPath)
    
    // 确保输出目录存在
    outputDir := filepath.Dir(finalOutputPath)
    if outputDir != "" && outputDir != "." {
        if err := os.MkdirAll(outputDir, 0755); err != nil {
            return fmt.Errorf("failed to create output directory: %v", err)
        }
    }
    
    // 处理文件冲突
    if _, err := os.Stat(finalOutputPath); err == nil {
        switch globalConfig.Pack.OnConflict {
        case "rename":
            finalOutputPath = getRenamedPath(finalOutputPath, globalConfig.Pack.RenamePattern)
            logger.Warn("Output file exists, using renamed: %s", finalOutputPath)
        case "replace":
            logger.Warn("Overwriting existing file: %s", finalOutputPath)
            if err := os.Remove(finalOutputPath); err != nil {
                return fmt.Errorf("failed to remove existing file: %v", err)
            }
        default:
            return fmt.Errorf("output file already exists: %s", finalOutputPath)
        }
    }
    
    // 重新打包
    ext := filepath.Ext(inputPath)
    if err := archiverInstance.Pack(tempDir, finalOutputPath, ext); err != nil {
        return fmt.Errorf("failed to pack archive: %v", err)
    }
    
    // 如果配置了删除输入文件
    if globalConfig.Pack.DeleteInput && finalOutputPath != inputPath {
        if err := os.Remove(inputPath); err != nil {
            logger.Warn("Failed to delete input file: %v", err)
        } else {
            logger.Info("Deleted input file: %s", inputPath)
        }
    }
    
    fileInfo, _ := os.Stat(finalOutputPath)
    logger.Info("Successfully decoded archive: %s (%d bytes) in %v!", 
        finalOutputPath, fileInfo.Size(), time.Since(startTime))
    
    return nil
}

// decodeBrarchiveFiles 解码 __brarchive 文件夹中的所有 .brarchive 文件
func decodeBrarchiveFiles(brarchiveDir, targetRoot string) error {
    // 获取 __brarchive 文件夹的父目录路径
    parentDir := filepath.Dir(brarchiveDir)
    
    // 遍历 __brarchive 文件夹
    err := filepath.Walk(brarchiveDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if info.IsDir() {
            return nil
        }
        
        // 检查是否是 .brarchive 文件
        if strings.HasSuffix(info.Name(), ".brarchive") {
            logger.Debug("Decoding brarchive file: %s", path)
            
            // 读取 .brarchive 文件
            data, err := os.ReadFile(path)
            if err != nil {
                return fmt.Errorf("failed to read brarchive file %s: %v", path, err)
            }
            
            // 解码
            entries, err := brarchive.Deserialize(data)
            if err != nil {
                return fmt.Errorf("failed to deserialize %s: %v", path, err)
            }
            
            // 确定目标目录
            // 获取相对于 __brarchive 文件夹的路径
            relPath, err := filepath.Rel(brarchiveDir, path)
            if err != nil {
                return err
            }
            
            // 移除 .brarchive 扩展名
            targetRelPath := strings.TrimSuffix(relPath, ".brarchive")
            
            // 目标目录是 __brarchive 的父目录 + 相对路径
            // 例如: subpacks/a/__brarchive/ui.brarchive -> 恢复到 subpacks/a/ui/
            targetDir := filepath.Join(parentDir, targetRelPath)
            
            // 创建目标目录
            if err := os.MkdirAll(targetDir, 0755); err != nil {
                return fmt.Errorf("failed to create target directory %s: %v", targetDir, err)
            }
            
            // 写入文件
            for fileName, content := range entries {
                filePath := filepath.Join(targetDir, fileName)
                if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
                    return fmt.Errorf("failed to write file %s: %v", filePath, err)
                }
                logger.Debug("Restored file: %s", filePath)
            }
            
            logger.Info("Restored %d files from %s to %s", len(entries), info.Name(), targetDir)
        }
        
        return nil
    })
    
    return err
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
    return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        relPath, err := filepath.Rel(src, path)
        if err != nil {
            return err
        }
        
        dstPath := filepath.Join(dst, relPath)
        
        if info.IsDir() {
            return os.MkdirAll(dstPath, info.Mode())
        }
        
        // 复制文件
        data, err := os.ReadFile(path)
        if err != nil {
            return err
        }
        
        if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
            return err
        }
        
        return os.WriteFile(dstPath, data, info.Mode())
    })
}

// findBrarchiveDirs 递归查找所有 __brarchive 文件夹
func findBrarchiveDirs(rootDir string) ([]string, error) {
    var brarchiveDirs []string
    
    err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if info.IsDir() && info.Name() == globalConfig.Archive.OutputDir {
            brarchiveDirs = append(brarchiveDirs, path)
            return filepath.SkipDir // 找到后跳过子目录，避免重复
        }
        
        return nil
    })
    
    return brarchiveDirs, err
}

// getDecodeOutputPath 获取解码后的输出路径
func getDecodeOutputPath(inputPath, outputPath string) string {
    if outputPath != "" {
        return outputPath
    }

    return getRenamedPath(inputPath, globalConfig.Pack.RenamePattern)
}

// getRenamedPath 根据重命名模板生成新路径
func getRenamedPath(originalPath, pattern string) string {
    dir := filepath.Dir(originalPath)
    base := filepath.Base(originalPath)
    ext := filepath.Ext(base)
    name := strings.TrimSuffix(base, ext)
    
    for i := 1; ; i++ {
        var newName string
        // 替换模板中的占位符
        if strings.Contains(pattern, "{number}") {
            newName = strings.ReplaceAll(pattern, "{filename}", name)
            newName = strings.ReplaceAll(newName, "{number}", fmt.Sprintf("%d", i))
            newName = strings.ReplaceAll(newName, "{ext}", ext)
        } else {
            // 如果模板没有包含 {number}，使用默认格式
            newName = fmt.Sprintf("%s (%d)%s", name, i, ext)
        }
        
        newPath := filepath.Join(dir, newName)
        if _, err := os.Stat(newPath); os.IsNotExist(err) {
            return newPath
        }
    }
}

func extractFileName(path string) string {
    base := filepath.Base(path)
    ext := filepath.Ext(base)
    if ext == "" {
        return base
    }
    return base[:len(base)-len(ext)]
}

func addExtensionIfMissing(path, extension string) string {
    if filepath.Ext(path) == "" {
        return path + "." + extension
    }
    return path
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
