package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// 日志级别常量
const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// LevelNames 日志级别名称
var LevelNames = map[int]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

// 日志级别颜色（控制台输出使用）
var levelColors = map[int]string{
	LevelDebug: "\033[37m", // 灰色
	LevelInfo:  "\033[32m", // 绿色
	LevelWarn:  "\033[33m", // 黄色
	LevelError: "\033[31m", // 红色
	LevelFatal: "\033[35m", // 紫色
}

const (
	colorReset = "\033[0m"
)

// Logger 结构体定义
type Logger struct {
	file       *os.File
	console    *log.Logger
	fileLogger *log.Logger
	minLevel   int
	useColor   bool
}

var (
	// 全局日志实例
	defaultLogger *Logger
	// 兼容旧接口
	InfoLogger  = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime)
	ErrorLogger = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime)
)

// InitLogger 初始化日志系统
func InitLogger(logDir string) error {
	// 创建日志目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 生成日志文件名（按日期）
	logFileName := fmt.Sprintf("audio_converter_%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, logFileName)

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 创建日志记录器
	logger := &Logger{
		file:       file,
		console:    log.New(os.Stdout, "", 0),
		fileLogger: log.New(file, "", 0),
		minLevel:   LevelDebug, // 默认记录所有级别
		useColor:   true,       // 默认使用彩色输出
	}

	// 设置全局日志实例
	defaultLogger = logger

	// 兼容旧接口
	consoleWriter := io.MultiWriter(os.Stdout)
	errorWriter := io.MultiWriter(os.Stderr, file)
	InfoLogger = log.New(consoleWriter, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(errorWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)

	// 输出启动信息
	logger.Info("日志系统初始化完成，日志文件: %s", logPath)
	return nil
}

// SetLevel 设置日志记录的最小级别
func SetLevel(level int) {
	if defaultLogger != nil {
		defaultLogger.minLevel = level
	}
}

// EnableColor 启用或禁用彩色输出
func EnableColor(enable bool) {
	if defaultLogger != nil {
		defaultLogger.useColor = enable
	}
}

// 获取调用者文件名和行号
func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown:0"
	}
	// 仅使用文件名
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' || file[i] == '\\' {
			short = file[i+1:]
			break
		}
	}
	return fmt.Sprintf("%s:%d", short, line)
}

// 格式化日志消息
func (l *Logger) formatLog(level int, format string, args ...interface{}) string {
	// 获取时间
	now := time.Now().Format("2006-01-02 15:04:05.000")

	// 获取调用位置
	callInfo := caller(2)

	// 格式化消息内容
	var msg string
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = format
	}

	// 控制台带颜色输出
	if l.useColor {
		color := levelColors[level]
		return fmt.Sprintf("%s [%s%s%s] [%s] %s",
			now, color, LevelNames[level], colorReset, callInfo, msg)
	}

	// 文件日志（无颜色）
	return fmt.Sprintf("%s [%s] [%s] %s",
		now, LevelNames[level], callInfo, msg)
}

// 记录普通日志
func (l *Logger) log(level int, format string, args ...interface{}) {
	if level < l.minLevel {
		return
	}

	logMsg := l.formatLog(level, format, args...)
	l.console.Println(logMsg)
	l.fileLogger.Println(logMsg)

	// FATAL级别日志输出后强制退出
	if level == LevelFatal {
		l.Close()
		os.Exit(1)
	}
}

// Debug 输出调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info 输出信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn 输出警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error 输出错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal 输出致命错误日志并退出程序
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
}

// Close 关闭日志文件
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
}

// 全局方法 - 兼容现有代码，同时提供新接口

// Debug 输出调试日志
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(format, args...)
	}
}

// Info 输出信息日志
func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(format, args...)
	} else {
		InfoLogger.Printf(format, args...)
	}
}

// Warn 输出警告日志
func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(format, args...)
	}
}

// Error 输出错误日志
func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(format, args...)
	} else {
		ErrorLogger.Printf(format, args...)
	}
}

// Fatal 输出致命错误日志并退出程序
func Fatal(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatal(format, args...)
	} else {
		ErrorLogger.Printf(format, args...)
		os.Exit(1)
	}
}

// CloseLogger 关闭日志文件
func CloseLogger() {
	if defaultLogger != nil {
		defaultLogger.Info("日志系统关闭")
		defaultLogger.Close()
	}
}

// CleanOldLogs 清理旧日志文件（保留最近7天）
func CleanOldLogs(logDir string) error {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return fmt.Errorf("读取日志目录失败: %v", err)
	}

	cutoff := time.Now().AddDate(0, 0, -7)
	var cleanedCount int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理匹配的日志文件
		if !strings.HasPrefix(entry.Name(), "audio_converter_") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(logDir, info.Name())
			if err := os.Remove(path); err != nil {
				if defaultLogger != nil {
					defaultLogger.Error("删除旧日志文件失败 %s: %v", path, err)
				} else {
					ErrorLogger.Printf("删除旧日志文件失败 %s: %v", path, err)
				}
			} else {
				cleanedCount++
			}
		}
	}

	if cleanedCount > 0 && defaultLogger != nil {
		defaultLogger.Info("已清理%d个过期日志文件", cleanedCount)
	}

	return nil
}
