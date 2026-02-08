package monitor

import (
	"fmt"
	"netmonitor/pkg/netinfo"
	"sync"
	"time"
)

type Stats struct {
	TotalEstablished  int
	TotalListeners    int
	NewConnections    int
	ClosedConnections int
	NewListeners      int
	ClosedListeners   int
	ByProtocol        map[string]int
	ByPID             map[int32]int
	LastUpdate        time.Time
	RecentNew         []time.Time
	RecentClosed      []time.Time
	mu                sync.RWMutex
}

func NewStats() *Stats {
	return &Stats{
		ByProtocol:  make(map[string]int),
		ByPID:      make(map[int32]int),
		RecentNew:   make([]time.Time, 0),
		RecentClosed: make([]time.Time, 0),
	}
}

func (s *Stats) Update(currentConns []netinfo.Connection) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalEstablished = 0
	s.TotalListeners = 0
	s.ByProtocol = make(map[string]int)
	s.ByPID = make(map[int32]int)

	for _, conn := range currentConns {
		if conn.Status == "ESTABLISHED" {
			s.TotalEstablished++
		} else if conn.Status == "LISTEN" {
			s.TotalListeners++
		}

		s.ByProtocol[conn.Protocol]++
		if conn.PID > 0 {
			s.ByPID[conn.PID]++
		}
	}

	s.LastUpdate = time.Now()

	// 清理60秒之前的记录
	s.cleanupOldEvents()
}

func (s *Stats) RecordNewConnection(protocol string, pid int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.NewConnections++
	s.ByProtocol[protocol]++
	if pid > 0 {
		s.ByPID[pid]++
	}

	// 记录时间戳
	s.RecentNew = append(s.RecentNew, time.Now())
	s.cleanupOldEvents()
}

func (s *Stats) RecordClosedConnection(protocol string, pid int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ClosedConnections++

	// 记录时间戳
	s.RecentClosed = append(s.RecentClosed, time.Now())
	s.cleanupOldEvents()
}

func (s *Stats) RecordNewListener(protocol string, pid int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.NewListeners++
}

func (s *Stats) RecordClosedListener(protocol string, pid int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ClosedListeners++
}

func (s *Stats) GetRecentNewCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.RecentNew)
}

func (s *Stats) GetRecentClosedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.RecentClosed)
}

func (s *Stats) cleanupOldEvents() {
	now := time.Now()
	cutoff := now.Add(-60 * time.Second)

	// 清理60秒之前的新连接
	for i := len(s.RecentNew) - 1; i >= 0; i-- {
		if s.RecentNew[i].Before(cutoff) {
			s.RecentNew = append(s.RecentNew[:0], s.RecentNew[i+1:]...)
			break
		}
	}

	// 清理60秒之前的关闭连接
	for i := len(s.RecentClosed) - 1; i >= 0; i-- {
		if s.RecentClosed[i].Before(cutoff) {
			s.RecentClosed = append(s.RecentClosed[:0], s.RecentClosed[i+1:]...)
			break
		}
	}
}

func (s *Stats) GetDisplay() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recentNew := len(s.RecentNew)
	recentClosed := len(s.RecentClosed)

	var result string
	result += fmt.Sprintf("\n=== 网络连接统计 [%s] ===\n", s.LastUpdate.Format("15:04:05"))
	result += fmt.Sprintf("活跃连接: %d  监听端口: %d\n", s.TotalEstablished, s.TotalListeners)
	result += fmt.Sprintf("最近60秒新建: %d  最近60秒关闭: %d\n", recentNew, recentClosed)

	if len(s.ByProtocol) > 0 {
		result += "\n按协议分布:\n"
		for protocol, count := range s.ByProtocol {
			result += fmt.Sprintf("  %s: %d\n", protocol, count)
		}
	}

	// 显示前5个最活跃的进程
	if len(s.ByPID) > 0 {
		type PIDCount struct {
			PID   int32
			Count int
		}

		var topPIDs []PIDCount
		for pid, count := range s.ByPID {
			topPIDs = append(topPIDs, PIDCount{PID: pid, Count: count})
		}

		// 简单排序(取前5)
		if len(topPIDs) > 5 {
			topPIDs = topPIDs[:5]
		}

		result += "\n活跃进程:\n"
		for _, pc := range topPIDs {
			result += fmt.Sprintf("  PID %d: %d 连接\n", pc.PID, pc.Count)
		}
	}

	result += "================================\n"

	return result
}

func (s *Stats) ResetCurrentSession() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.NewConnections = 0
	s.ClosedConnections = 0
	s.NewListeners = 0
	s.ClosedListeners = 0
	s.RecentNew = make([]time.Time, 0)
	s.RecentClosed = make([]time.Time, 0)
}

