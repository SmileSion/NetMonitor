package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ListenerWriter   io.Writer
	EstablishedWriter io.Writer
	ColorEnabled      bool
	LogToConsole      bool
)

// ANSI颜色代码
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
)

func InitLogger(listenerDir, establishedDir string, colorEnabled, logToConsole bool) error {
	ColorEnabled = colorEnabled
	LogToConsole = logToConsole

	if err := createLogWriter(listenerDir, &ListenerWriter); err != nil {
		return err
	}
	return createLogWriter(establishedDir, &EstablishedWriter)
}

func createLogWriter(dir string, writer *io.Writer) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(dir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// 总是写入文件
	writers := []io.Writer{f}

	// 根据配置决定是否输出到控制台
	if LogToConsole {
		writers = append(writers, os.Stdout)
	}

	*writer = io.MultiWriter(writers...)
	return nil
}

func LogMessage(writer io.Writer, message string) {
	entry := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
	writer.Write([]byte(entry))
}

// 带颜色的日志输出
func LogConnection(writer io.Writer, connType, protocol, localAddr, remoteAddr string, pid int32, processName string, isNew bool) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	var symbol, color string
	if isNew {
		symbol = "[+]"
		color = ColorGreen
	} else {
		symbol = "[-]"
		color = ColorRed
	}

	var message string
	if connType == "LISTEN" {
		message = fmt.Sprintf("%s %s %s %s PID:%d %s",
			symbol, connType, protocol, localAddr, pid, processName)
	} else {
		message = fmt.Sprintf("%s %s %s → %s PID:%d %s",
			symbol, protocol, localAddr, remoteAddr, pid, processName)
	}

	// Windows终端可能不支持ANSI颜色,需要检查
	if ColorEnabled && isColorSupported() {
		entry := fmt.Sprintf("[%s] %s%s%s\n", timestamp, color, message, ColorReset)
		writer.Write([]byte(entry))
	} else {
		entry := fmt.Sprintf("[%s] %s\n", timestamp, message)
		writer.Write([]byte(entry))
	}
}

// 检查是否支持颜色输出
func isColorSupported() bool {
	// Windows 10及以上支持ANSI颜色, Linux终端也支持
	term := os.Getenv("TERM")
	isWindows := strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") || os.Getenv("WT_SESSION") != ""

	// Windows 10+ 和大多数Linux终端都支持ANSI颜色
	return strings.Contains(term, "xterm") || strings.Contains(term, "color") ||
		strings.Contains(term, "ansi") || isWindows
}

// 统计信息输出
func LogStats(writer io.Writer, establishedCount, listenerCount, newConnections, closedConnections, newListeners, closedListeners int) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if ColorEnabled && isColorSupported() {
		stats := fmt.Sprintf("[%s] %s=== 统计: 活跃连接=%d 监听端口=%d 新建=%d 关闭=%d ===%s\n",
			timestamp, ColorCyan, establishedCount, listenerCount, newConnections, closedConnections, ColorReset)
		writer.Write([]byte(stats))
	} else {
		stats := fmt.Sprintf("[%s] === 统计: 活跃连接=%d 监听端口=%d 新建=%d 关闭=%d ===\n",
			timestamp, establishedCount, listenerCount, newConnections, closedConnections)
		writer.Write([]byte(stats))
	}
}

// 信息输出
func LogInfo(writer io.Writer, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if ColorEnabled && isColorSupported() {
		entry := fmt.Sprintf("[%s] %s[INFO]%s %s\n", timestamp, ColorBlue, ColorReset, message)
		writer.Write([]byte(entry))
	} else {
		entry := fmt.Sprintf("[%s] [INFO] %s\n", timestamp, message)
		writer.Write([]byte(entry))
	}
}

// 警告输出
func LogWarning(writer io.Writer, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if ColorEnabled && isColorSupported() {
		entry := fmt.Sprintf("[%s] %s[WARN]%s %s\n", timestamp, ColorYellow, ColorReset, message)
		writer.Write([]byte(entry))
	} else {
		entry := fmt.Sprintf("[%s] [WARN] %s\n", timestamp, message)
		writer.Write([]byte(entry))
	}
}