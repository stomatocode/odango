package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WRDashboardHandler handles the Web Responder dashboard
type WRDashboardHandler struct {
	// Store active calls and events
	activeCalls map[string]*CallSession
	events      []CallEvent
	mu          sync.RWMutex

	// WebSocket upgrader
	upgrader websocket.Upgrader

	// Connected clients
	clients   map[*websocket.Conn]bool
	broadcast chan CallEvent
}

// CallSession represents an active IVR call
type CallSession struct {
	SessionID    string      `json:"session_id"`
	CallerNumber string      `json:"caller_number"`
	AreaCode     string      `json:"area_code"`
	Location     string      `json:"location"`
	StartTime    time.Time   `json:"start_time"`
	LastAction   string      `json:"last_action"`
	State        string      `json:"state"` // "active", "menu", "ended"
	Events       []CallEvent `json:"events"`
}

// CallEvent represents an event in the call flow
type CallEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	SessionID    string    `json:"session_id"`
	EventType    string    `json:"event_type"` // "call_start", "dtmf", "response", "hangup"
	Description  string    `json:"description"`
	Data         string    `json:"data"`
	CallerNumber string    `json:"caller_number"`
}

// NewWRDashboardHandler creates a new dashboard handler
func NewWRDashboardHandler() *WRDashboardHandler {
	handler := &WRDashboardHandler{
		activeCalls: make(map[string]*CallSession),
		events:      make([]CallEvent, 0, 1000), // Keep last 1000 events
		clients:     make(map[*websocket.Conn]bool),
		broadcast:   make(chan CallEvent, 100),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	// Start WebSocket broadcast handler
	go handler.handleBroadcast()

	return handler
}

// ShowDashboard displays the Web Responder dashboard
func (wrd *WRDashboardHandler) ShowDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "wr_dashboard.html", gin.H{
		"title": "Web Responder Live Dashboard",
	})
}

// GetActiveCalls returns current active calls as JSON
func (wrd *WRDashboardHandler) GetActiveCalls(c *gin.Context) {
	wrd.mu.RLock()
	defer wrd.mu.RUnlock()

	calls := make([]*CallSession, 0, len(wrd.activeCalls))
	for _, call := range wrd.activeCalls {
		calls = append(calls, call)
	}

	c.JSON(http.StatusOK, gin.H{
		"active_calls": calls,
		"total_active": len(calls),
		"timestamp":    time.Now(),
	})
}

// GetRecentEvents returns recent call events
func (wrd *WRDashboardHandler) GetRecentEvents(c *gin.Context) {
	wrd.mu.RLock()
	defer wrd.mu.RUnlock()

	// Return last 50 events
	start := 0
	if len(wrd.events) > 50 {
		start = len(wrd.events) - 50
	}

	c.JSON(http.StatusOK, gin.H{
		"events": wrd.events[start:],
		"total":  len(wrd.events),
	})
}

// HandleWebSocket handles WebSocket connections for real-time updates
func (wrd *WRDashboardHandler) HandleWebSocket(c *gin.Context) {
	conn, err := wrd.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Register client
	wrd.mu.Lock()
	wrd.clients[conn] = true
	wrd.mu.Unlock()

	// Send initial state
	wrd.sendInitialState(conn)

	// Keep connection alive and handle messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			wrd.mu.Lock()
			delete(wrd.clients, conn)
			wrd.mu.Unlock()
			break
		}
	}
}

// LogCallStart logs a new call starting
func (wrd *WRDashboardHandler) LogCallStart(sessionID, callerNumber, areaCode, location string) {
	wrd.mu.Lock()
	defer wrd.mu.Unlock()

	session := &CallSession{
		SessionID:    sessionID,
		CallerNumber: callerNumber,
		AreaCode:     areaCode,
		Location:     location,
		StartTime:    time.Now(),
		State:        "active",
		LastAction:   "Call initiated",
		Events:       []CallEvent{},
	}

	wrd.activeCalls[sessionID] = session

	event := CallEvent{
		Timestamp:    time.Now(),
		SessionID:    sessionID,
		EventType:    "call_start",
		Description:  "New call from " + areaCode + " (" + location + ")",
		CallerNumber: callerNumber,
	}

	wrd.addEvent(event)
	wrd.broadcast <- event
}

// LogDTMF logs DTMF input
func (wrd *WRDashboardHandler) LogDTMF(sessionID, digit string) {
	wrd.mu.Lock()
	defer wrd.mu.Unlock()

	if session, exists := wrd.activeCalls[sessionID]; exists {
		session.LastAction = "DTMF: " + digit
		session.State = "menu"

		description := ""
		switch digit {
		case "1":
			description = "User selected: Local Time"
		case "2":
			description = "User selected: Temperature"
		case "3":
			description = "User selected: Air Quality"
		default:
			description = "User pressed: " + digit
		}

		event := CallEvent{
			Timestamp:    time.Now(),
			SessionID:    sessionID,
			EventType:    "dtmf",
			Description:  description,
			Data:         digit,
			CallerNumber: session.CallerNumber,
		}

		session.Events = append(session.Events, event)
		wrd.addEvent(event)
		wrd.broadcast <- event
	}
}

// LogResponse logs a response sent to caller
func (wrd *WRDashboardHandler) LogResponse(sessionID, response string) {
	wrd.mu.Lock()
	defer wrd.mu.Unlock()

	if session, exists := wrd.activeCalls[sessionID]; exists {
		session.LastAction = "Response sent"

		event := CallEvent{
			Timestamp:    time.Now(),
			SessionID:    sessionID,
			EventType:    "response",
			Description:  "Response: " + response,
			CallerNumber: session.CallerNumber,
		}

		session.Events = append(session.Events, event)
		wrd.addEvent(event)
		wrd.broadcast <- event
	}
}

// LogCallEnd logs call ending
func (wrd *WRDashboardHandler) LogCallEnd(sessionID string) {
	wrd.mu.Lock()
	defer wrd.mu.Unlock()

	if session, exists := wrd.activeCalls[sessionID]; exists {
		session.State = "ended"

		event := CallEvent{
			Timestamp:    time.Now(),
			SessionID:    sessionID,
			EventType:    "hangup",
			Description:  "Call ended",
			CallerNumber: session.CallerNumber,
		}

		wrd.addEvent(event)
		wrd.broadcast <- event

		// Remove from active calls after a delay
		go func() {
			time.Sleep(5 * time.Second)
			wrd.mu.Lock()
			delete(wrd.activeCalls, sessionID)
			wrd.mu.Unlock()
		}()
	}
}

// Helper methods

func (wrd *WRDashboardHandler) addEvent(event CallEvent) {
	wrd.events = append(wrd.events, event)

	// Keep only last 1000 events
	if len(wrd.events) > 1000 {
		wrd.events = wrd.events[1:]
	}
}

func (wrd *WRDashboardHandler) sendInitialState(conn *websocket.Conn) {
	wrd.mu.RLock()
	defer wrd.mu.RUnlock()

	// Send current active calls
	state := gin.H{
		"type":          "initial",
		"active_calls":  wrd.activeCalls,
		"recent_events": wrd.events,
	}

	conn.WriteJSON(state)
}

func (wrd *WRDashboardHandler) handleBroadcast() {
	for {
		event := <-wrd.broadcast

		wrd.mu.RLock()
		clients := make([]*websocket.Conn, 0, len(wrd.clients))
		for client := range wrd.clients {
			clients = append(clients, client)
		}
		wrd.mu.RUnlock()

		message := gin.H{
			"type":  "event",
			"event": event,
		}

		for _, client := range clients {
			err := client.WriteJSON(message)
			if err != nil {
				client.Close()
				wrd.mu.Lock()
				delete(wrd.clients, client)
				wrd.mu.Unlock()
			}
		}
	}
}

// TestCall simulates a call for testing
func (wrd *WRDashboardHandler) TestCall(c *gin.Context) {
	var req struct {
		CallerNumber string `json:"caller_number"`
		Digit        string `json:"digit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simulate call flow
	sessionID := "test_" + time.Now().Format("20060102150405")

	// Start call
	wrd.LogCallStart(sessionID, req.CallerNumber, "415", "San Francisco, CA")

	// Simulate menu response after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		wrd.LogResponse(sessionID, "Welcome! Press 1 for time, 2 for temperature...")

		if req.Digit != "" {
			time.Sleep(2 * time.Second)
			wrd.LogDTMF(sessionID, req.Digit)

			time.Sleep(1 * time.Second)
			response := ""
			switch req.Digit {
			case "1":
				response = "The current time is 2:30 PM"
			case "2":
				response = "The temperature is 68°F"
			case "3":
				response = "Air quality index is 45 (Good)"
			}
			wrd.LogResponse(sessionID, response)

			time.Sleep(2 * time.Second)
			wrd.LogCallEnd(sessionID)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"status":     "test_initiated",
	})
}
