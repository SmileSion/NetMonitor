package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CleanupConfig 日志清理配置
type CleanupConfig struct {
	Enabled          bool
	RetentionDays   int
	CompressEnabled bool
}

// CleanupOldLogs 清理过期日志并压缩旧日志
func CleanupOldLogs(dir string, config CleanupConfig) error {
	if !config.Enabled {
		return nil
	}

	// 遍历目录中的所有文件
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取日志目录失败: %w", err)
	}

	now := time.Now()
	cutoffDate := now.AddDate(0, 0, -config.RetentionDays)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dir, file.Name())

		// 获取文件信息
		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		// 跳过已压缩的文件
		if strings.HasSuffix(file.Name(), ".gz") {
			// 检查是否需要删除
			if fileInfo.ModTime().Before(cutoffDate) {
				os.Remove(filePath)
			}
			continue
		}

		// 检查是否为今天或昨天的日志文件
		if !isLogFile(file.Name()) {
			continue
		}

		// 获取日志日期
		logDate, err := parseLogDate(file.Name())
		if err != nil {
			continue
		}

		// 如果是昨天的日志,进行压缩
		yesterday := now.AddDate(0, 0, -1)
		if logDate.Before(yesterday.Truncate(24 * time.Hour)) {
			if config.CompressEnabled {
				err := compressLog(filePath)
				if err == nil {
					fmt.Printf("已压缩日志: %s\n", file.Name())
				}
			}
		}

		// 删除过期的日志
		if logDate.Before(cutoffDate) {
			os.Remove(filePath)
			fmt.Printf("已删除过期日志: %s\n", file.Name())
		}
	}

	return nil
}

// isLogFile 检查是否为日志文件
func isLogFile(filename string) bool {
	return strings.HasSuffix(filename, ".log")
}

// parseLogDate 从日志文件名中解析日期
func parseLogDate(filename string) (time.Time, error) {
	// 去掉.log后缀
	dateStr := strings.TrimSuffix(filename, ".log")

	// 解析日期(格式: 2006-01-02)
	return time.Parse("2006-01-02", dateStr)
}

// compressLog 压缩日志文件
func compressLog(filePath string) error {
	// 打开源文件
	sourceFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	// 创建压缩文件
	gzPath := filePath + ".gz"
	gzFile, err := os.Create(gzPath)
	if err != nil {
		return fmt.Errorf("创建压缩文件失败: %w", err)
	}
	defer gzFile.Close()

	// 创建gzip写入器
	gzWriter := gzip.NewWriter(gzFile)
	defer gzWriter.Close()

	// 复制数据
	_, err = io.Copy(gzWriter, sourceFile)
	if err != nil {
		os.Remove(gzPath)
		return fmt.Errorf("压缩文件失败: %w", err)
	}

	// 压缩成功后删除原文件
	os.Remove(filePath)

	return nil
}

// StartCleanupTask 启动定时清理任务
func StartCleanupTask(listenerDir, establishedDir string, config CleanupConfig) {
	if !config.Enabled {
		return
	}

	// 立即执行一次清理
	go func() {
		CleanupOldLogs(listenerDir, config)
		CleanupOldLogs(establishedDir, config)
	}()

	// 每天执行一次清理
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			CleanupOldLogs(listenerDir, config)
			CleanupOldLogs(establishedDir, config)
		}
	}()
}
