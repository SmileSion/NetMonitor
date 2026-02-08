package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"netmonitor/pkg/monitor"
	"netmonitor/pkg/netinfo"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	port        int
	stats       *monitor.Stats
	filter      *netinfo.ConnectionFilter
	clients     map[*websocket.Conn]bool
	clientsMu   sync.RWMutex
	broadcast   chan []byte
	lastConns   []netinfo.Connection
	lastConnsMu sync.RWMutex
}

type ConnectionEvent struct {
	Type        string    `json:"type"`
	Protocol    string    `json:"protocol"`
	LocalAddr   string    `json:"local_addr"`
	RemoteAddr  string    `json:"remote_addr"`
	PID         int32     `json:"pid"`
	ProcessName string    `json:"process_name"`
	Timestamp   time.Time `json:"timestamp"`
}

type StatsData struct {
	TotalConnections  int               `json:"total_connections"`
	TotalListeners    int               `json:"total_listeners"`
	NewConnections    int               `json:"new_connections"`
	ClosedConnections int               `json:"closed_connections"`
	ByProtocol        map[string]int    `json:"by_protocol"`
	ByPID             map[int32]int     `json:"by_pid"`
	LastUpdate        time.Time         `json:"last_update"`
}

type ConnectionResponse struct {
	LocalAddr   string `json:"local_addr"`
	RemoteAddr  string `json:"remote_addr"`
	Protocol    string `json:"protocol"`
	Status      string `json:"status"`
	PID         int32  `json:"pid"`
	ProcessName string `json:"process_name"`
}

func NewServer(port int) *Server {
	return &Server{
		port:      port,
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte, 100),
		filter:    &netinfo.ConnectionFilter{},
	}
}

func (s *Server) SetStats(stats *monitor.Stats) {
	s.stats = stats
}

func (s *Server) SetFilter(filter *netinfo.ConnectionFilter) {
	if filter != nil {
		s.filter = filter
	}
}

func (s *Server) Start() error {
	go s.handleBroadcast()

	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/stats", s.handleStats)
	http.HandleFunc("/api/connections", s.handleConnections)
	http.HandleFunc("/ws", s.handleWebSocket)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("Web界面已启动: http://localhost:%d\n", s.port)

	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if s.stats == nil {
		http.Error(w, "Stats not initialized", http.StatusInternalServerError)
		return
	}

	statsData := StatsData{
		TotalConnections:  0,
		TotalListeners:    0,
		NewConnections:    s.stats.GetRecentNewCount(),
		ClosedConnections: s.stats.GetRecentClosedCount(),
		ByProtocol:        make(map[string]int),
		ByPID:             make(map[int32]int),
		LastUpdate:        time.Now(),
	}

	s.lastConnsMu.RLock()
	conns := s.lastConns
	s.lastConnsMu.RUnlock()

	for _, conn := range conns {
		if conn.Status == "ESTABLISHED" {
			statsData.TotalConnections++
		} else if conn.Status == "LISTEN" {
			statsData.TotalListeners++
		}
		statsData.ByProtocol[conn.Protocol]++
		if conn.PID > 0 {
			statsData.ByPID[conn.PID]++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statsData)
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	s.lastConnsMu.RLock()
	allConns := s.lastConns
	s.lastConnsMu.RUnlock()

	// 应用过滤
	var filteredConns []ConnectionResponse
	for _, conn := range allConns {
		if s.filter == nil || !s.filter.ShouldFilter(conn) {
			filteredConns = append(filteredConns, ConnectionResponse{
				LocalAddr:   conn.LocalAddr,
				RemoteAddr:  conn.RemoteAddr,
				Protocol:    conn.Protocol,
				Status:      conn.Status,
				PID:         conn.PID,
				ProcessName: conn.ProcessName,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredConns)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	// 发送当前连接列表
	s.lastConnsMu.RLock()
	conns := s.lastConns
	s.lastConnsMu.RUnlock()

	if len(conns) > 0 {
		data := s.buildConnectionsMessage(conns)
		conn.WriteMessage(websocket.TextMessage, data)
	}

	// 保持连接
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	s.clientsMu.Lock()
	delete(s.clients, conn)
	s.clientsMu.Unlock()
}

func (s *Server) handleBroadcast() {
	for {
		msg := <-s.broadcast

		s.clientsMu.RLock()
		for client := range s.clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				client.Close()
				s.clientsMu.Lock()
				delete(s.clients, client)
				s.clientsMu.Unlock()
			}
		}
		s.clientsMu.RUnlock()
	}
}

func (s *Server) BroadcastNewConnection(conn netinfo.Connection) {
	event := ConnectionEvent{
		Type:        "new",
		Protocol:    conn.Protocol,
		LocalAddr:   conn.LocalAddr,
		RemoteAddr:  conn.RemoteAddr,
		PID:         conn.PID,
		ProcessName: conn.ProcessName,
		Timestamp:   time.Now(),
	}

	data, _ := json.Marshal(map[string]interface{}{
		"type": "event",
		"data": event,
	})

	select {
	case s.broadcast <- data:
	default:
	}
}

func (s *Server) BroadcastClosedConnection(conn netinfo.Connection) {
	event := ConnectionEvent{
		Type:        "closed",
		Protocol:    conn.Protocol,
		LocalAddr:   conn.LocalAddr,
		RemoteAddr:  conn.RemoteAddr,
		PID:         conn.PID,
		ProcessName: conn.ProcessName,
		Timestamp:   time.Now(),
	}

	data, _ := json.Marshal(map[string]interface{}{
		"type": "event",
		"data": event,
	})

	select {
	case s.broadcast <- data:
	default:
	}
}

func (s *Server) UpdateConnections(conns []netinfo.Connection) {
	s.lastConnsMu.Lock()
	s.lastConns = conns
	s.lastConnsMu.Unlock()

	// 广播完整连接列表
	data := s.buildConnectionsMessage(conns)

	select {
	case s.broadcast <- data:
	default:
	}
}

func (s *Server) buildConnectionsMessage(conns []netinfo.Connection) []byte {
	// 应用过滤
	var filteredConns []ConnectionResponse
	for _, conn := range conns {
		if s.filter == nil || !s.filter.ShouldFilter(conn) {
			filteredConns = append(filteredConns, ConnectionResponse{
				LocalAddr:   conn.LocalAddr,
				RemoteAddr:  conn.RemoteAddr,
				Protocol:    conn.Protocol,
				Status:      conn.Status,
				PID:         conn.PID,
				ProcessName: conn.ProcessName,
			})
		}
	}

	data, _ := json.Marshal(map[string]interface{}{
		"type": "connections",
		"data": filteredConns,
	})

	return data
}
