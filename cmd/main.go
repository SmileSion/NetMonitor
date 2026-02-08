package main

import (
	"fmt"
	"netmonitor/pkg/config"
	"netmonitor/pkg/logger"
	"netmonitor/pkg/monitor"
	"netmonitor/pkg/netinfo"
	"netmonitor/pkg/web"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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
	if err := logger.InitLogger(cfg.Log.ListenerDir, cfg.Log.EstablishedDir,
		cfg.Log.ColorEnabled, cfg.Monitor.LogToConsole); err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}

	// 创建过滤器
	filter := &netinfo.ConnectionFilter{
		ProcessName: cfg.Filter.ProcessName,
		PIDs:        cfg.Filter.PIDs,
		Protocols:   cfg.Filter.Protocols,
		RemoteIP:    cfg.Filter.RemoteIP,
	}

	// 打印启动信息
	printStartupInfo(cfg, filter)

	// 初始化监控器
	listenerMon := monitor.NewListenerMonitor(filter)
	if err := listenerMon.Initialize(); err != nil {
		panic(err)
	}

	establishedMon := monitor.NewEstablishedMonitor(filter)
	if err := establishedMon.Initialize(); err != nil {
		panic(err)
	}

	// 初始化统计
	stats := monitor.NewStats()

	// 初始化Web服务器(如果启用)
	var webServer *web.Server
	if cfg.Web.Enabled {
		webServer = web.NewServer(cfg.Web.Port)
		webServer.SetStats(stats)
		webServer.SetFilter(filter)

		// 预加载连接数据
		allConns, _ := netinfo.GetConnections()
		webServer.UpdateConnections(allConns)

		go func() {
			if err := webServer.Start(); err != nil {
				logger.LogWarning(os.Stdout, fmt.Sprintf("Web服务器启动失败: %v", err))
			}
		}()
		time.Sleep(100 * time.Millisecond) // 等待Web服务器启动
	}

	// 设置优雅退出
	setupExitHandler()

	// 启动定时检测
	ticker := time.NewTicker(cfg.Monitor.GetInterval())
	defer ticker.Stop()

	// 统计显示定时器
	var statsTicker *time.Ticker
	if cfg.Monitor.ShowStats {
		statsTicker = time.NewTicker(10 * time.Second)
		defer statsTicker.Stop()
	}

	for {
		select {
		case <-ticker.C:
			// 监听端口检测
			newListeners, closedListeners, err := listenerMon.CheckChanges()
			if err != nil {
				logger.LogWarning(logger.ListenerWriter, fmt.Sprintf("监听端口检测错误: %v", err))
				continue
			}

			if len(newListeners) > 0 {
				listenerMon.LogNewListeners(newListeners)
				for _, l := range newListeners {
					stats.RecordNewListener(l.Protocol, l.PID)
					if webServer != nil {
						webServer.BroadcastNewConnection(l)
					}
				}
			}

			if len(closedListeners) > 0 {
				listenerMon.LogClosedListeners(closedListeners)
				for _, l := range closedListeners {
					stats.RecordClosedListener(l.Protocol, l.PID)
					if webServer != nil {
						webServer.BroadcastClosedConnection(l)
					}
				}
			}

			// 已建立连接检测
			newEstablished, closedEstablished, err := establishedMon.CheckChanges()
			if err != nil {
				logger.LogWarning(logger.EstablishedWriter, fmt.Sprintf("已建立连接检测错误: %v", err))
				continue
			}

			if len(newEstablished) > 0 {
				establishedMon.LogNewConnections(newEstablished)
				for _, c := range newEstablished {
					stats.RecordNewConnection(c.Protocol, c.PID)
					if webServer != nil {
						webServer.BroadcastNewConnection(c)
					}
				}
			}

			if len(closedEstablished) > 0 {
				establishedMon.LogClosedConnections(closedEstablished)
				for _, c := range closedEstablished {
					stats.RecordClosedConnection(c.Protocol, c.PID)
					if webServer != nil {
						webServer.BroadcastClosedConnection(c)
					}
				}
			}

			// 更新统计信息
			allConns, _ := netinfo.GetConnections()
			stats.Update(allConns)

			// 更新Web服务器的连接列表
			if webServer != nil {
				webServer.UpdateConnections(allConns)
			}

		case <-statsTicker.C:
			// 显示统计信息
			logger.LogInfo(os.Stdout, stats.GetDisplay())
		}
	}
}

func printStartupInfo(cfg *config.Config, filter *netinfo.ConnectionFilter) {
	fmt.Println("========================================")
	fmt.Println("       网络连接监控器已启动")
	fmt.Println("========================================")
	fmt.Printf("检测间隔: %d 秒\n", cfg.Monitor.Interval)
	fmt.Printf("统计显示: %s\n", getBoolString(cfg.Monitor.ShowStats))
	fmt.Printf("彩色输出: %s\n", getBoolString(cfg.Log.ColorEnabled))
	fmt.Println("\n过滤配置:")
	fmt.Printf("  进程名称: %s\n", getStringOrDefault(filter.ProcessName, "全部"))
	fmt.Printf("  PID过滤: %s\n", getPIDsString(filter.PIDs))
	fmt.Printf("  协议类型: %s\n", getProtocolsString(filter.Protocols))
	fmt.Printf("  远程IP: %s\n", getStringOrDefault(filter.RemoteIP, "全部"))
	fmt.Println("========================================\n")
	logger.LogInfo(os.Stdout, "开始监控网络连接...")
}

func getBoolString(b bool) string {
	if b {
		return "启用"
	}
	return "禁用"
}

func getStringOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func getPIDsString(pids []int32) string {
	if len(pids) == 0 {
		return "全部"
	}
	result := ""
	for i, pid := range pids {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%d", pid)
	}
	return result
}

func getProtocolsString(protocols []string) string {
	if len(protocols) == 0 {
		return "全部"
	}
	result := ""
	for i, p := range protocols {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func setupExitHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n收到信号 %v, 正在退出...\n", sig)
		logger.LogInfo(os.Stdout, "监控器已停止")
		os.Exit(0)
	}()
}
