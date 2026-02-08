package monitor

import (
	"net"
	"netmonitor/pkg/logger"
	"netmonitor/pkg/netinfo"
	"strconv"
)

type ListenerMonitor struct {
	initialState map[uint32]netinfo.Connection
	filter       *netinfo.ConnectionFilter
}

func NewListenerMonitor(filter *netinfo.ConnectionFilter) *ListenerMonitor {
	return &ListenerMonitor{
		initialState: make(map[uint32]netinfo.Connection),
		filter:       filter,
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
		if isListeningPort(c) && !m.filter.ShouldFilter(c) {
			port := extractPort(c.LocalAddr)
			m.initialState[port] = c
		}
	}
	return nil
}

func (m *ListenerMonitor) CheckChanges() ([]netinfo.Connection, []netinfo.Connection, error) {
	currentConns, err := netinfo.GetConnections()
	if err != nil {
		return nil, nil, err
	}

	var newListeners []netinfo.Connection
	var closedListeners []netinfo.Connection
	currentState := make(map[uint32]netinfo.Connection)

	for _, c := range currentConns {
		if isListeningPort(c) {
			port := extractPort(c.LocalAddr)
			currentState[port] = c

			// 检查是否被过滤器过滤
			shouldFilter := m.filter.ShouldFilter(c)

			// 检查新监听端口
			if _, exists := m.initialState[port]; !exists && !shouldFilter {
				newListeners = append(newListeners, c)
			}
		}
	}

	// 检查关闭的监听端口
	for port, oldConn := range m.initialState {
		if _, exists := currentState[port]; !exists {
			closedListeners = append(closedListeners, oldConn)
		}
	}

	m.initialState = currentState
	return newListeners, closedListeners, nil
}

func (m *ListenerMonitor) LogNewListeners(listeners []netinfo.Connection) {
	for _, l := range listeners {
		logger.LogConnection(logger.ListenerWriter, "LISTEN", l.Protocol,
			l.LocalAddr, "", l.PID, l.ProcessName, true)
	}
}

func (m *ListenerMonitor) LogClosedListeners(listeners []netinfo.Connection) {
	for _, l := range listeners {
		logger.LogConnection(logger.ListenerWriter, "LISTEN", l.Protocol,
			l.LocalAddr, "", l.PID, l.ProcessName, false)
	}
}