package main

import (
	"fmt"
	"path/filepath"
	"netmonitor/pkg/config"
	"netmonitor/pkg/logger"
	"netmonitor/pkg/monitor"
	"time"
)

func main() {
	// 初始化配置
	cfgPath := filepath.Join("config", "config.toml")
	if err := config.InitConfig(cfgPath); err != nil {
		panic(fmt.Sprintf("初始化配置失败: %v", err))
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}

	// 初始化日志
	if err := logger.InitLogger(cfg.Log.ListenerDir, cfg.Log.EstablishedDir); err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}

	fmt.Println("启动端口监控...")
	fmt.Printf("配置检测间隔: %d秒\n", cfg.Monitor.Interval)

	// 初始化监控器
	listenerMon := monitor.NewListenerMonitor()
	if err := listenerMon.Initialize(); err != nil {
		panic(err)
	}

	establishedMon := monitor.NewEstablishedMonitor()
	if err := establishedMon.Initialize(); err != nil {
		panic(err)
	}

	// 启动定时检测
	ticker := time.NewTicker(cfg.Monitor.GetInterval())
	defer ticker.Stop()

	for range ticker.C {
		// 监听端口检测
		newListeners, err := listenerMon.CheckChanges()
		if err != nil {
			fmt.Printf("监听端口检测错误: %v\n", err)
			continue
		}
		if len(newListeners) > 0 {
			listenerMon.LogNewListeners(newListeners)
		}

		// 已建立连接检测
		newEstablished, err := establishedMon.CheckChanges()
		if err != nil {
			fmt.Printf("已建立连接检测错误: %v\n", err)
			continue
		}
		if len(newEstablished) > 0 {
			establishedMon.LogNewConnections(newEstablished)
		}
	}
}