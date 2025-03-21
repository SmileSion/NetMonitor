package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Log     LogConfig
	Monitor MonitorConfig
}

type LogConfig struct {
	ListenerDir   string `toml:"listener_dir"`
	EstablishedDir string `toml:"established_dir"`
}

type MonitorConfig struct {
	Interval int `toml:"interval"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Log: LogConfig{
			ListenerDir:   "logs/listener_logs",
			EstablishedDir: "logs/established_logs",
		},
		Monitor: MonitorConfig{
			Interval: 5,
		},
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // 返回默认配置
		}
		return nil, err
	}
	return cfg, nil
}

// 获取间隔时间（转换为Duration）
func (m *MonitorConfig) GetInterval() time.Duration {
	return time.Duration(m.Interval) * time.Second
}

// 初始化配置文件（如果不存在则创建）
func InitConfig(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		defaultCfg := `[log]
listener_dir = "logs/listener_logs"
established_dir = "logs/established_logs"

[monitor]
interval = 5
`

		return os.WriteFile(path, []byte(defaultCfg), 0644)
	}
	return nil
}