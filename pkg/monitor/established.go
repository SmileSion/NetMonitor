package monitor

import (
	"fmt"
	"netmonitor/pkg/logger"
	"netmonitor/pkg/netinfo"
)

type EstablishedMonitor struct {
	initialState map[string]netinfo.Connection
	filter       *netinfo.ConnectionFilter
}

func NewEstablishedMonitor(filter *netinfo.ConnectionFilter) *EstablishedMonitor {
	return &EstablishedMonitor{
		initialState: make(map[string]netinfo.Connection),
		filter:       filter,
	}
}

func (m *EstablishedMonitor) getKey(c netinfo.Connection) string {
	return fmt.Sprintf("%s|%s|%s", c.Protocol, c.LocalAddr, c.RemoteAddr)
}

func (m *EstablishedMonitor) Initialize() error {
	conns, err := netinfo.GetConnections()
	if err != nil {
		return err
	}

	for _, c := range conns {
		if c.Status == "ESTABLISHED" && !m.filter.ShouldFilter(c) {
			m.initialState[m.getKey(c)] = c
		}
	}
	return nil
}

func (m *EstablishedMonitor) CheckChanges() ([]netinfo.Connection, []netinfo.Connection, error) {
	currentConns, err := netinfo.GetConnections()
	if err != nil {
		return nil, nil, err
	}

	var newConnections []netinfo.Connection
	var closedConnections []netinfo.Connection
	currentState := make(map[string]netinfo.Connection)

	for _, c := range currentConns {
		if c.Status == "ESTABLISHED" {
			key := m.getKey(c)
			currentState[key] = c

			// 检查是否被过滤器过滤
			shouldFilter := m.filter.ShouldFilter(c)

			// 检查新连接
			if _, exists := m.initialState[key]; !exists && !shouldFilter {
				newConnections = append(newConnections, c)
			}
		}
	}

	// 检查关闭的连接
	for key, oldConn := range m.initialState {
		if _, exists := currentState[key]; !exists {
			closedConnections = append(closedConnections, oldConn)
		}
	}

	m.initialState = currentState
	return newConnections, closedConnections, nil
}

func (m *EstablishedMonitor) LogNewConnections(conns []netinfo.Connection) {
	for _, c := range conns {
		logger.LogConnection(logger.EstablishedWriter, "", c.Protocol,
			c.LocalAddr, c.RemoteAddr, c.PID, c.ProcessName, true)
	}
}

func (m *EstablishedMonitor) LogClosedConnections(conns []netinfo.Connection) {
	for _, c := range conns {
		logger.LogConnection(logger.EstablishedWriter, "", c.Protocol,
			c.LocalAddr, c.RemoteAddr, c.PID, c.ProcessName, false)
	}
}