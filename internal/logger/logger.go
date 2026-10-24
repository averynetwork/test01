package logger

import (
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/fatih/color"
    "github.com/wisebreeze/brarchive-go/pkg/config"
)

var (
    infoColor   = color.New(color.FgGreen)
    warnColor   = color.New(color.FgYellow)
    errorColor  = color.New(color.FgRed)
    debugColor  = color.New(color.FgCyan)
    
    logFile     *os.File
    appConfig   *config.Config
    colorEnabled bool = true
    timestampEnabled bool = true
)

// InitWithConfig 使用配置初始化日志器
func InitWithConfig(cfg *config.Config, logPath string) error {
    appConfig = cfg
    
    // 设置颜色和时间戳
    colorEnabled = cfg.Log.Color
    timestampEnabled = cfg.Log.Timestamp
    
    // 如果指定了日志文件
    if logPath != "" {
        if err := initLogFile(logPath); err != nil {
            return err
        }
    } else if cfg.Log.File != "" {
        if err := initLogFile(cfg.Log.File); err != nil {
            return err
        }
    }
    
    return nil
}

// Init 向后兼容的初始化函数
func Init(logPath string) error {
    return InitWithConfig(config.DefaultConfig(), logPath)
}

func initLogFile(logPath string) error {
    // Create log directory if it doesn't exist
    dir := filepath.Dir(logPath)
    if dir != "" && dir != "." {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create log directory: %v", err)
        }
    }
    
    file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("failed to create log file: %v", err)
    }
    
    logFile = file
    return nil
}

func Info(format string, args ...interface{}) {
    if appConfig != nil && !appConfig.ShouldLog(config.LogLevelInfo) {
        return
    }
    log("INFO", infoColor, format, args...)
}

func Warn(format string, args ...interface{}) {
    if appConfig != nil && !appConfig.ShouldLog(config.LogLevelWarn) {
        return
    }
    log("WARN", warnColor, format, args...)
}

func Error(format string, args ...interface{}) {
    if appConfig != nil && !appConfig.ShouldLog(config.LogLevelError) {
        return
    }
    log("ERROR", errorColor, format, args...)
}

func Debug(format string, args ...interface{}) {
    if appConfig != nil && !appConfig.ShouldLog(config.LogLevelDebug) {
        return
    }
    log("DEBUG", debugColor, format, args...)
}

func log(level string, c *color.Color, format string, args ...interface{}) {
    msg := fmt.Sprintf(format, args...)
    
    // Console output
    if colorEnabled {
        if timestampEnabled {
            timestamp := time.Now().Format("2006-01-02 15:04:05.000")
            c.Printf("[%s %s] %s\n", timestamp, level, msg)
        } else {
            c.Printf("[%s] %s\n", level, msg)
        }
    } else {
        if timestampEnabled {
            timestamp := time.Now().Format("2006-01-02 15:04:05.000")
            fmt.Printf("[%s %s] %s\n", timestamp, level, msg)
        } else {
            fmt.Printf("[%s] %s\n", level, msg)
        }
    }
    
    // File output without colors
    if logFile != nil {
        timestamp := time.Now().Format("2006-01-02 15:04:05.000")
        fmt.Fprintf(logFile, "[%s %s] %s\n", timestamp, level, msg)
    }
}

func Close() {
    if logFile != nil {
        logFile.Close()
        logFile = nil
    }
}
