package monitor

import (
	"fmt"
	"net"
	"netmonitor/pkg/logger"
	"netmonitor/pkg/netinfo"
	"strconv"
)

type ListenerMonitor struct {
	initialState map[uint32]netinfo.Connection
}

func NewListenerMonitor() *ListenerMonitor {
	return &ListenerMonitor{
		initialState: make(map[uint32]netinfo.Connection),
	}
}

// 从地址字符串提取端口号
func extractPort(addr string) uint32 {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	port, _ := strconv.ParseUint(portStr, 10, 32)
	return uint32(port)
}

// 判断是否为监听端口（完全对齐Python逻辑）
func isListeningPort(c netinfo.Connection) bool {
	if c.Protocol == "TCP" && c.Status == "LISTEN" {
		return true
	}
	if c.Protocol == "UDP" && c.LocalAddr != ":0" { // UDP 只需本地端口非零
		return true
	}
	return false
}

func (m *ListenerMonitor) Initialize() error {
	conns, err := netinfo.GetConnections()
	if err != nil {
		return err
	}

	for _, c := range conns {
		if isListeningPort(c) {
			port := extractPort(c.LocalAddr)
			m.initialState[port] = c
		}
	}
	return nil
}

func (m *ListenerMonitor) CheckChanges() ([]netinfo.Connection, error) {
	currentConns, err := netinfo.GetConnections()
	if err != nil {
		return nil, err
	}

	var newListeners []netinfo.Connection
	currentState := make(map[uint32]netinfo.Connection)

	for _, c := range currentConns {
		if isListeningPort(c) {
			port := extractPort(c.LocalAddr)
			currentState[port] = c
			if _, exists := m.initialState[port]; !exists {
				newListeners = append(newListeners, c)
			}
		}
	}

	m.initialState = currentState
	return newListeners, nil
}

func (m *ListenerMonitor) LogNewListeners(listeners []netinfo.Connection) {
	for _, l := range listeners {
		msg := fmt.Sprintf("[LISTEN] 协议: %s, 地址: %s, PID: %d, 进程: %s",
			l.Protocol, l.LocalAddr, l.PID, l.ProcessName)
		logger.LogMessage(logger.ListenerWriter, msg)
	}
}