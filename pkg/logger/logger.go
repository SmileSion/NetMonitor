package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

var (
	ListenerWriter   io.Writer
	EstablishedWriter io.Writer
)

func InitLogger(listenerDir, establishedDir string) error {
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

	// 同时输出到文件和控制台
	*writer = io.MultiWriter(f, os.Stdout)
	return nil
}

func LogMessage(writer io.Writer, message string) {
	entry := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), message)
	writer.Write([]byte(entry))
}