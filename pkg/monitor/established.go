package monitor

import (
	"fmt"
	"netmonitor/pkg/logger"
	"netmonitor/pkg/netinfo"
)

type EstablishedMonitor struct {
	initialState map[string]netinfo.Connection
}

func NewEstablishedMonitor() *EstablishedMonitor {
	return &EstablishedMonitor{
		initialState: make(map[string]netinfo.Connection),
	}
}

func (m *EstablishedMonitor) getKey(c netinfo.Connection) string {
    // 使用完整的本地和远程地址作为唯一键
    return fmt.Sprintf("%s|%s|%s", c.Protocol, c.LocalAddr, c.RemoteAddr)
}

func (m *EstablishedMonitor) Initialize() error {
	conns, err := netinfo.GetConnections()
	if err != nil {
		return err
	}

	for _, c := range conns {
		if c.Status == "ESTABLISHED" {
			m.initialState[m.getKey(c)] = c
		}
	}
	return nil
}

func (m *EstablishedMonitor) CheckChanges() ([]netinfo.Connection, error) {
	currentConns, err := netinfo.GetConnections()
	if err != nil {
		return nil, err
	}

	var newConnections []netinfo.Connection
	currentState := make(map[string]netinfo.Connection)

	for _, c := range currentConns {
		if c.Status == "ESTABLISHED" {
			key := m.getKey(c)
			currentState[key] = c
			if _, exists := m.initialState[key]; !exists {
				newConnections = append(newConnections, c)
			}
		}
	}

	m.initialState = currentState
	return newConnections, nil
}

func (m *EstablishedMonitor) LogNewConnections(conns []netinfo.Connection) {
    for _, c := range conns {
        msg := fmt.Sprintf("[ESTABLISHED] 协议: %s, 本地地址: %s → 远程地址: %s, PID: %d, 进程: %s",
            c.Protocol, c.LocalAddr, c.RemoteAddr, c.PID, c.ProcessName)
        logger.LogMessage(logger.EstablishedWriter, msg)
    }
}