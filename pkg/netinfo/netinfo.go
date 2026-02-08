package netinfo

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"strings"
	"syscall"
)

type Connection struct {
	LocalAddr   string // 本地地址(IP:Port)
	RemoteAddr  string // 远程地址(IP:Port)
	Protocol    string // 协议类型(TCP/UDP)
	Status      string // 连接状态
	PID         int32  // 进程ID
	ProcessName string // 进程名称
}

type ConnectionFilter struct {
	ProcessName string
	PIDs        []int32
	Protocols   []string
	RemoteIP    string
}

// 精准协议判断（与Python逻辑一致）
func getProtocol(c net.ConnectionStat) string {
	switch c.Type {
	case syscall.SOCK_STREAM:
		return "TCP"
	case syscall.SOCK_DGRAM:
		return "UDP"
	default:
		return fmt.Sprintf("UNKNOWN-%d", c.Type)
	}
}

func (f *ConnectionFilter) ShouldFilter(conn Connection) bool {
	// 检查协议
	if len(f.Protocols) > 0 {
		found := false
		for _, p := range f.Protocols {
			if strings.EqualFold(p, conn.Protocol) {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}

	// 检查进程名
	if f.ProcessName != "" && !strings.EqualFold(conn.ProcessName, f.ProcessName) {
		return true
	}

	// 检查PID列表
	if len(f.PIDs) > 0 {
		found := false
		for _, pid := range f.PIDs {
			if pid == conn.PID {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}

	// 检查远程IP
	if f.RemoteIP != "" && !strings.Contains(conn.RemoteAddr, f.RemoteIP) {
		return true
	}

	return false
}

func GetConnections() ([]Connection, error) {
	conns, err := net.Connections("all")
	if err != nil {
		return nil, err
	}

	var result []Connection
	for _, c := range conns {
		protocol := getProtocol(c)
		localAddr := fmt.Sprintf("%s:%d", c.Laddr.IP, c.Laddr.Port)
		remoteAddr := fmt.Sprintf("%s:%d", c.Raddr.IP, c.Raddr.Port)

		conn := Connection{
			LocalAddr:   localAddr,
			RemoteAddr:  remoteAddr,
			Protocol:    protocol,
			Status:      c.Status,
			PID:         c.Pid,
			ProcessName: "",
		}

		if c.Pid > 0 {
			p, _ := process.NewProcess(c.Pid)
			if name, err := p.Name(); err == nil {
				conn.ProcessName = name
			}
		}

		result = append(result, conn)
	}
	return result, nil
}