package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/BurntSushi/toml"
)

// LogLevel 日志级别
type LogLevel string

const (
    LogLevelSilent LogLevel = "silent"  // 完全不显示日志
    LogLevelError  LogLevel = "error"   // 只显示错误
    LogLevelWarn   LogLevel = "warn"    // 显示警告和错误
    LogLevelInfo   LogLevel = "info"    // 显示信息、警告和错误（默认）
    LogLevelDebug  LogLevel = "debug"   // 显示所有日志
)

// Config 主配置结构
type Config struct {
    Log      LogConfig      `toml:"log"`
    Archive  ArchiveConfig  `toml:"archive"`
    Pack     PackConfig     `toml:"pack"`
}

// LogConfig 日志配置
type LogConfig struct {
    Level     LogLevel `toml:"level"`      // 日志级别: silent, error, warn, info, debug
    Color     bool     `toml:"color"`      // 是否使用彩色输出
    Timestamp bool     `toml:"timestamp"`  // 是否显示时间戳
    File      string   `toml:"file"`       // 日志文件路径（可选）
}

// ArchiveConfig 压缩包处理配置
type ArchiveConfig struct {
    IncludeExts          []string `toml:"include_exts"`           // 需要打包的文件后缀
    ExcludeDirs          []string `toml:"exclude_dirs"`           // 排除的文件夹
    SpecialFolders       []string `toml:"special_folders"`        // 特殊处理的文件夹（如 subpacks）
    ExcludeFiles         []string `toml:"exclude_files"`          // 排除的文件
    OutputDir            string   `toml:"output_dir"`             // 输出目录名，默认 "__brarchive"
    SupportedArchives    []string `toml:"supported_archives"`     // 支持的压缩包格式
    RemoveProcessedFiles bool     `toml:"remove_processed_files"` // 是否删除已处理的文件
}

// PackConfig 打包配置
type PackConfig struct {
    OutputPath    string `toml:"output_path"`     // 输出路径
    KeepOriginal  bool   `toml:"keep_original"`   // 是否保留原始压缩包（已废弃，使用 DeleteInput）
    RenamePattern string `toml:"rename_pattern"`  // 重名文件的命名模板
    OnConflict    string `toml:"on_conflict"`     // 冲突处理: "rename" 或 "replace"
    DeleteInput   bool   `toml:"delete_input"`    // 是否删除输入文件（原压缩包）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
    return &Config{
        Log: LogConfig{
            Level:     LogLevelInfo,
            Color:     true,
            Timestamp: true,
            File:      "",
        },
        Archive: ArchiveConfig{
            IncludeExts:          []string{".json", ".lang", ".properties"},
            ExcludeDirs:          []string{"textures", "materials", "texts", "sounds"},
            SpecialFolders:       []string{"subpacks"}, // 特殊处理的文件夹
            ExcludeFiles:         []string{"ui/_global_variables.json"},
            OutputDir:            "__brarchive",
            SupportedArchives:    []string{".zip", ".7z", ".tar", ".tar.xz", ".tar.gz", ".tar.zst", ".tar.bz2", ".tar.lz4", ".mcpack"},
            RemoveProcessedFiles: true, // 默认删除已处理的文件
        },
        Pack: PackConfig{
            OutputPath:    "./output",
            KeepOriginal:  true,  // 默认保留原始压缩包
            RenamePattern: "{filename} ({number}){ext}",
            OnConflict:    "rename", // 默认重命名
            DeleteInput:   false,    // 默认不删除输入文件
        },
    }
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
    config := DefaultConfig()

    // 如果文件不存在，返回默认配置
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return config, nil
    }

    // 解析配置文件
    if _, err := toml.DecodeFile(path, config); err != nil {
        return nil, fmt.Errorf("failed to parse config file %s: %v", path, err)
    }

    // 验证日志级别
    if err := validateLogLevel(config.Log.Level); err != nil {
        return nil, err
    }

    // 验证冲突处理方式
    if config.Pack.OnConflict != "rename" && config.Pack.OnConflict != "replace" {
        config.Pack.OnConflict = "rename"
    }

    // 验证输出目录
    if config.Archive.OutputDir == "" {
        config.Archive.OutputDir = "__brarchive"
    }

    return config, nil
}

// validateLogLevel 验证日志级别
func validateLogLevel(level LogLevel) error {
    validLevels := []LogLevel{LogLevelSilent, LogLevelError, LogLevelWarn, LogLevelInfo, LogLevelDebug}
    for _, l := range validLevels {
        if l == level {
            return nil
        }
    }
    return fmt.Errorf("invalid log level: %s (must be one of: silent, error, warn, info, debug)", level)
}

// ShouldLog 判断是否应该记录指定级别的日志
func (c *Config) ShouldLog(level LogLevel) bool {
    if c.Log.Level == LogLevelSilent {
        return false
    }

    levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}
    configLevelIndex := indexOf(levels, c.Log.Level)
    targetLevelIndex := indexOf(levels, level)

    if configLevelIndex == -1 || targetLevelIndex == -1 {
        return false
    }

    return targetLevelIndex >= configLevelIndex
}

// IsTargetExtension 判断是否为需要处理的后缀
func (c *Config) IsTargetExtension(ext string) bool {
    ext = strings.ToLower(ext)
    for _, target := range c.Archive.IncludeExts {
        if strings.EqualFold(ext, target) {
            return true
        }
    }
    return false
}

// IsExcludedFolder 判断是否为排除的文件夹
func (c *Config) IsExcludedFolder(folderName string) bool {
    for _, excluded := range c.Archive.ExcludeDirs {
        if strings.EqualFold(folderName, excluded) {
            return true
        }
    }
    return false
}

// IsSpecialFolder 判断是否为特殊处理的文件夹
func (c *Config) IsSpecialFolder(folderName string) bool {
    for _, special := range c.Archive.SpecialFolders {
        if strings.EqualFold(folderName, special) {
            return true
        }
    }
    return false
}

// IsExcludedFile 判断是否为排除的文件
func (c *Config) IsExcludedFile(filePath string) bool {
    for _, excluded := range c.Archive.ExcludeFiles {
        // 支持完整路径或文件名匹配
        if strings.EqualFold(filePath, excluded) || strings.EqualFold(filepath.Base(filePath), excluded) {
            return true
        }
    }
    return false
}

// IsSupportedArchive 判断是否为支持的压缩包格式
func (c *Config) IsSupportedArchive(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    for _, supported := range c.Archive.SupportedArchives {
        if strings.EqualFold(ext, supported) {
            return true
        }
    }
    return false
}

func indexOf(slice []LogLevel, item LogLevel) int {
    for i, v := range slice {
        if v == item {
            return i
        }
    }
    return -1
}

// SaveDefaultConfig 保存默认配置文件
func SaveDefaultConfig(path string) error {
    config := DefaultConfig()
    
    // 确保目录存在
    dir := filepath.Dir(path)
    if dir != "" && dir != "." {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create config directory: %v", err)
        }
    }

    file, err := os.Create(path)
    if err != nil {
        return fmt.Errorf("failed to create config file: %v", err)
    }
    defer file.Close()

    encoder := toml.NewEncoder(file)
    if err := encoder.Encode(config); err != nil {
        return fmt.Errorf("failed to write config file: %v", err)
    }

    return nil
}
