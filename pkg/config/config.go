package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Log     LogConfig
	Monitor MonitorConfig
	Filter  FilterConfig
	Web     WebConfig
}

type LogConfig struct {
	ListenerDir     string `toml:"listener_dir"`
	EstablishedDir  string `toml:"established_dir"`
	ColorEnabled    bool   `toml:"color_enabled"`
	RetentionDays   int    `toml:"retention_days"`    // 日志保留天数
	AutoCompress    bool   `toml:"auto_compress"`    // 是否自动压缩日志
}

type MonitorConfig struct {
	Interval      int  `toml:"interval"`
	ShowStats     bool `toml:"show_stats"`
	LogToConsole  bool `toml:"log_to_console"`
}

type FilterConfig struct {
	ProcessName string   `toml:"process_name"` // 进程名称过滤(留空表示不过滤)
	PIDs         []int32 `toml:"pids"`         // PID过滤(留空表示不过滤)
	Protocols    []string `toml:"protocols"`   // 协议过滤: tcp, udp
	RemoteIP     string   `toml:"remote_ip"`   // 远程IP过滤(留空表示不过滤)
}

type WebConfig struct {
	Enabled bool `toml:"enabled"` // 是否启用Web界面
	Port    int  `toml:"port"`    // Web服务端口
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Log: LogConfig{
			ListenerDir:     "logs/listener_logs",
			EstablishedDir:  "logs/established_logs",
			ColorEnabled:     true,
			RetentionDays:   7,
			AutoCompress:    true,
		},
		Monitor: MonitorConfig{
			Interval:      1,
			ShowStats:     true,
			LogToConsole:  true,
		},
		Filter: FilterConfig{
			ProcessName: "",
			PIDs:        []int32{},
			Protocols:   []string{"tcp", "udp"},
			RemoteIP:    "",
		},
		Web: WebConfig{
			Enabled: false,
			Port:    8080,
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

// 检查协议是否在过滤列表中
func (f *FilterConfig) ShouldFilterProtocol(protocol string) bool {
	if len(f.Protocols) == 0 {
		return false // 不过滤
	}
	for _, p := range f.Protocols {
		if strings.EqualFold(p, protocol) {
			return false // 匹配,不过滤
		}
	}
	return true // 不匹配,过滤
}

// 检查进程是否应该被过滤
func (f *FilterConfig) ShouldFilterProcess(pid int32, processName string) bool {
	// 检查进程名
	if f.ProcessName != "" {
		if !strings.EqualFold(processName, f.ProcessName) {
			return true // 过滤
		}
	}

	// 检查PID列表
	if len(f.PIDs) > 0 {
		found := false
		for _, p := range f.PIDs {
			if p == pid {
				found = true
				break
			}
		}
		if !found {
			return true // 过滤
		}
	}

	// 检查远程IP(需要在连接信息中检查)
	return false // 不过滤
}

// 检查远程地址是否应该被过滤
func (f *FilterConfig) ShouldFilterRemoteAddr(remoteAddr string) bool {
	if f.RemoteIP == "" {
		return false
	}
	return !strings.Contains(remoteAddr, f.RemoteIP)
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
color_enabled = true

[monitor]
interval = 1  # 单位：秒
show_stats = true
log_to_console = true

[filter]
# 留空表示不过滤
process_name = ""  # 要监控的进程名称,例如 "chrome.exe"
pids = []          # 要监控的PID列表,例如 [1234, 5678]
protocols = ["tcp", "udp"]  # 监控的协议类型
remote_ip = ""      # 过滤特定远程IP
`

		return os.WriteFile(path, []byte(defaultCfg), 0644)
	}
	return nil
}