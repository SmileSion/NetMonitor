package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ListenerLogDir   = "logs/listener_logs"
	EstablishedLogDir = "logs/established_logs"
)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func LogMessage(logDir, message string) error {
	if err := ensureDir(logDir); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), message)
	_, err = f.WriteString(entry)
	return err
}