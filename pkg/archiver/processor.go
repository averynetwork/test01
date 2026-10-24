package archiver

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/wisebreeze/brarchive-go/internal/logger"
    "github.com/wisebreeze/brarchive-go/pkg/brarchive"
    "github.com/wisebreeze/brarchive-go/pkg/config"
)

// Processor 处理压缩包内的文件
type Processor struct {
    Config *config.Config
}

// NewProcessor 创建新的处理器
func NewProcessor(cfg *config.Config) *Processor {
    return &Processor{
        Config: cfg,
    }
}

// ProcessFolder 处理文件夹，找到符合条件的子文件夹并打包成.brarchive
func (p *Processor) ProcessFolder(rootDir, outputDir string) (int, error) {
    var processedCount int
    var processedDirs []string // 记录处理过的目录，用于后续清理
    var createdOutputDirs []string // 记录创建的输出目录
    
    err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            // 如果文件/目录已经不存在，跳过
            if os.IsNotExist(err) {
                return nil
            }
            return err
        }
        
        if !info.IsDir() {
            return nil
        }
        
        // 检查是否应该排除此目录
        relPath, err := filepath.Rel(rootDir, path)
        if err != nil {
            return err
        }
        
        if relPath == "." {
            return nil
        }
        
        // 跳过 __brarchive 目录本身
        if filepath.Base(path) == p.Config.Archive.OutputDir {
            return filepath.SkipDir
        }
        
        // 检查是否是特殊文件夹（如 subpacks）
        isSpecial := p.Config.IsSpecialFolder(filepath.Base(path))
        
        // 如果不在特殊文件夹中，检查是否在排除列表中
        if !isSpecial && p.Config.IsExcludedFolder(filepath.Base(path)) {
            logger.Debug("Skipping excluded folder: %s", path)
            return filepath.SkipDir
        }
        
        // 对于特殊文件夹，需要处理其子目录，而不是跳过
        if isSpecial {
            logger.Debug("Processing special folder: %s", path)
            // 继续遍历子目录，不跳过
            return nil
        }
        
        // 检查此目录是否包含目标文件
        hasTargetFiles, targetFiles := p.hasTargetFiles(path)
        if hasTargetFiles {
            // 确定输出目录
            var targetOutputDir string
            var relativePathForLog string
            
            // 检查当前目录是否在特殊文件夹内
            isInSpecial, specialRoot, subpackRoot := p.isInSpecialFolder(rootDir, path)
            
            if isInSpecial && specialRoot != "" {
                // 在特殊文件夹内，输出到子包的 __brarchive 目录
                // specialRoot 是 subpacks 文件夹的路径
                // subpackRoot 是子包的路径，例如 subpacks/light
                // 相对路径: 相对于子包的路径
                relToSubpack, err := filepath.Rel(subpackRoot, path)
                if err != nil {
                    return err
                }
                
                // 输出目录是子包下的 __brarchive
                targetOutputDir = filepath.Join(subpackRoot, p.Config.Archive.OutputDir)
                relativePathForLog = relToSubpack
                
                // 确保输出目录存在
                if err := os.MkdirAll(targetOutputDir, 0755); err != nil {
                    return fmt.Errorf("failed to create output directory: %v", err)
                }
                createdOutputDirs = append(createdOutputDirs, targetOutputDir)
            } else {
                // 普通目录，输出到根目录的 __brarchive
                targetOutputDir = outputDir
                relativePathForLog = relPath
            }
            
            // 处理此目录
            if err := p.processDirectory(path, targetOutputDir, relativePathForLog, targetFiles); err != nil {
                logger.Error("Failed to process directory %s: %v", path, err)
                return nil // 继续处理其他目录
            }
            processedCount++
            processedDirs = append(processedDirs, path)
        }
        
        return nil
    })
    
    // 清理空目录
    if p.Config.Archive.RemoveProcessedFiles && len(processedDirs) > 0 {
        logger.Info("Cleaning up processed directories...")
        // 从最深的目录开始清理，避免重复检查
        for i := len(processedDirs) - 1; i >= 0; i-- {
            p.cleanupDirectory(processedDirs[i])
        }
    }
    
    // 清理空的 __brarchive 目录
    p.cleanupEmptyOutputDirs(rootDir, createdOutputDirs)
    
    return processedCount, err
}

// isInSpecialFolder 检查路径是否在特殊文件夹内
// 返回: (是否在特殊文件夹内, 特殊文件夹根路径, 子包根路径)
func (p *Processor) isInSpecialFolder(rootDir, path string) (bool, string, string) {
    relPath, err := filepath.Rel(rootDir, path)
    if err != nil {
        return false, "", ""
    }
    
    parts := strings.Split(relPath, string(filepath.Separator))
    for i, part := range parts {
        if p.Config.IsSpecialFolder(part) {
            // 找到特殊文件夹（如 subpacks）
            specialPath := filepath.Join(rootDir, strings.Join(parts[:i+1], string(filepath.Separator)))
            
            // 子包根路径：特殊文件夹的直接子目录
            // 例如: subpacks/light
            if i+1 < len(parts) {
                subpackPath := filepath.Join(rootDir, strings.Join(parts[:i+2], string(filepath.Separator)))
                return true, specialPath, subpackPath
            }
            
            // 如果路径本身就是特殊文件夹，返回 false（不处理特殊文件夹本身）
            return false, "", ""
        }
    }
    
    return false, "", ""
}

// cleanupEmptyOutputDirs 清理空的 __brarchive 目录
func (p *Processor) cleanupEmptyOutputDirs(rootDir string, createdDirs []string) {
    // 去重
    dirMap := make(map[string]bool)
    for _, dir := range createdDirs {
        dirMap[dir] = true
    }
    
    for dir := range dirMap {
        // 检查目录是否存在
        if _, err := os.Stat(dir); os.IsNotExist(err) {
            continue
        }
        
        // 检查目录是否为空
        entries, err := os.ReadDir(dir)
        if err != nil {
            continue
        }
        
        // 如果目录为空，删除它
        if len(entries) == 0 {
            if err := os.Remove(dir); err != nil {
                logger.Warn("Failed to remove empty output directory %s: %v", dir, err)
            } else {
                logger.Debug("Removed empty output directory: %s", dir)
            }
        }
    }
}

// hasTargetFiles 检查目录是否包含目标文件
func (p *Processor) hasTargetFiles(dir string) (bool, []string) {
    var targetFiles []string
    
    // 检查目录是否存在
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        return false, nil
    }
    
    entries, err := os.ReadDir(dir)
    if err != nil {
        return false, nil
    }
    
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        
        fileName := entry.Name()
        fullPath := filepath.Join(dir, fileName)
        
        // 获取相对路径用于排除检查
        relPath, _ := filepath.Rel(filepath.Dir(dir), fullPath)
        
        // 检查是否排除此文件
        if p.Config.IsExcludedFile(relPath) || p.Config.IsExcludedFile(fileName) {
            logger.Debug("Skipping excluded file: %s", fullPath)
            continue
        }
        
        ext := strings.ToLower(filepath.Ext(fileName))
        if p.Config.IsTargetExtension(ext) {
            targetFiles = append(targetFiles, fileName)
        }
    }
    
    return len(targetFiles) > 0, targetFiles
}

// processDirectory 处理单个目录
func (p *Processor) processDirectory(dirPath, outputDir, relPath string, targetFiles []string) error {
    // 构建输出路径
    archiveName := filepath.Base(dirPath) + ".brarchive"
    
    // 构建目标路径：outputDir + relPath的父目录
    var destDir string
    if relPath == filepath.Base(dirPath) {
        // 直接子目录
        destDir = outputDir
    } else {
        // 有父路径的情况，需要保持目录结构
        // 例如: relPath = "ui/folder"，我们想要 outputDir/ui
        parentRelPath := filepath.Dir(relPath)
        if parentRelPath == "." {
            destDir = outputDir
        } else {
            destDir = filepath.Join(outputDir, parentRelPath)
        }
    }
    
    // 确保输出目录存在
    if err := os.MkdirAll(destDir, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %v", err)
    }
    
    outputPath := filepath.Join(destDir, archiveName)
    
    // 检查输出文件是否已存在（brarchive 文件冲突）
    if _, err := os.Stat(outputPath); err == nil {
        // 使用 Pack.OnConflict 配置
        switch p.Config.Pack.OnConflict {
        case "replace":
            logger.Warn("Overwriting existing brarchive file: %s", outputPath)
        case "rename":
            outputPath = p.getUniqueFilename(outputPath)
            logger.Warn("Brarchive file exists, using renamed: %s", outputPath)
        default:
            return fmt.Errorf("output brarchive file already exists: %s", outputPath)
        }
    }
    
    // 收集目录中所有目标文件的内容
    entries := make(map[string]string)
    for _, fileName := range targetFiles {
        filePath := filepath.Join(dirPath, fileName)
        
        // 检查文件是否存在
        fileInfo, err := os.Stat(filePath)
        if err != nil {
            if os.IsNotExist(err) {
                logger.Debug("File no longer exists, skipping: %s", filePath)
            } else {
                logger.Warn("Failed to stat file %s: %v", filePath, err)
            }
            continue
        }
        
        // 确保是普通文件
        if fileInfo.IsDir() {
            continue
        }
        
        content, err := os.ReadFile(filePath)
        if err != nil {
            logger.Warn("Failed to read file %s: %v", filePath, err)
            continue
        }
        entries[fileName] = string(content)
        logger.Debug("Added file %s (%d bytes)", fileName, len(content))
    }
    
    if len(entries) == 0 {
        return fmt.Errorf("no valid files to archive in %s", dirPath)
    }
    
    // 序列化为 brarchive 格式
    archiveData, err := brarchive.Serialize(entries)
    if err != nil {
        return fmt.Errorf("failed to serialize: %v", err)
    }
    
    // 写入文件
    if err := os.WriteFile(outputPath, archiveData, 0644); err != nil {
        return fmt.Errorf("failed to write archive: %v", err)
    }
    
    logger.Info("Created archive: %s (%d files)", outputPath, len(entries))
    
    // 如果配置了删除已处理的文件，则删除原始文件
    if p.Config.Archive.RemoveProcessedFiles {
        for _, fileName := range targetFiles {
            filePath := filepath.Join(dirPath, fileName)
            if err := os.Remove(filePath); err != nil {
                if !os.IsNotExist(err) {
                    logger.Warn("Failed to remove file %s: %v", filePath, err)
                }
            } else {
                logger.Debug("Removed file: %s", filePath)
            }
        }
    }
    
    return nil
}

// cleanupDirectory 递归删除空目录
func (p *Processor) cleanupDirectory(dirPath string) {
    // 先检查目录是否存在
    info, err := os.Stat(dirPath)
    if err != nil {
        if os.IsNotExist(err) {
            // 目录已经不存在，无需清理
            return
        }
        logger.Warn("Failed to stat directory %s: %v", dirPath, err)
        return
    }
    
    // 确保是目录
    if !info.IsDir() {
        return
    }
    
    // 检查目录是否为空
    entries, err := os.ReadDir(dirPath)
    if err != nil {
        logger.Warn("Failed to read directory %s: %v", dirPath, err)
        return
    }
    
    // 如果目录不为空，不删除
    if len(entries) > 0 {
        // 检查是否只有 __brarchive 目录
        onlyBrarchive := true
        for _, entry := range entries {
            if entry.Name() != p.Config.Archive.OutputDir {
                onlyBrarchive = false
                break
            }
        }
        
        if onlyBrarchive {
            // 检查 __brarchive 目录是否为空
            subDir := filepath.Join(dirPath, p.Config.Archive.OutputDir)
            subEntries, err := os.ReadDir(subDir)
            if err == nil && len(subEntries) == 0 {
                // __brarchive 目录为空，删除它
                os.Remove(subDir)
                // 重新检查当前目录
                entries, _ = os.ReadDir(dirPath)
                if len(entries) == 0 {
                    // 目录现在为空，继续删除
                } else {
                    return
                }
            } else {
                return
            }
        } else {
            return
        }
    }
    
    // 删除空目录
    if err := os.Remove(dirPath); err != nil {
        if !os.IsNotExist(err) {
            logger.Warn("Failed to remove directory %s: %v", dirPath, err)
        }
        return
    }
    
    logger.Info("Removed empty directory: %s", dirPath)
    
    // 递归检查父目录
    parentDir := filepath.Dir(dirPath)
    
    // 如果父目录是根目录或输出目录，停止递归
    if parentDir == "." || parentDir == "/" {
        return
    }
    
    // 检查父目录是否在排除列表中
    if p.Config.IsExcludedFolder(filepath.Base(parentDir)) {
        return
    }
    
    // 继续清理父目录
    p.cleanupDirectory(parentDir)
}

// getUniqueFilename 获取唯一的文件名（用于 brarchive 文件）
func (p *Processor) getUniqueFilename(filePath string) string {
    dir := filepath.Dir(filePath)
    base := filepath.Base(filePath)
    ext := filepath.Ext(base)
    name := strings.TrimSuffix(base, ext)
    
    for i := 1; ; i++ {
        newName := fmt.Sprintf("%s (%d)%s", name, i, ext)
        newPath := filepath.Join(dir, newName)
        if _, err := os.Stat(newPath); os.IsNotExist(err) {
            return newPath
        }
    }
}
