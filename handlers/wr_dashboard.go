package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"o-dan-go/events"
	"o-dan-go/services"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WRDashboardHandler handles the Web Responder dashboard
type WRDashboardHandler struct {
	clients   map[*websocket.Conn]bool
	broadcast chan events.CallEvent
	upgrader  websocket.Upgrader
}

// NewWRDashboardHandler creates a new dashboard handler
func NewWRDashboardHandler() *WRDashboardHandler {
	handler := &WRDashboardHandler{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan events.CallEvent),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				// TODO: Restrict in production
				return true
			},
		},
	}

	// Start broadcasting events
	go handler.broadcastEvents()

	return handler
}

// ShowDashboard displays the dashboard HTML
func (h *WRDashboardHandler) ShowDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "wr_dashboard.html", gin.H{
		"title": "Web Responder Dashboard",
	})
}

// GetActiveCalls returns current active calls as JSON
func (h *WRDashboardHandler) GetActiveCalls(c *gin.Context) {
	calls := events.Manager.GetActiveCalls()
	c.JSON(http.StatusOK, gin.H{
		"calls": calls,
		"count": len(calls),
	})
}

// GetRecentEvents returns recent events (mock data for now)
func (h *WRDashboardHandler) GetRecentEvents(c *gin.Context) {
	// TODO: Implement actual event history storage
	events := []gin.H{
		{
			"timestamp": time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			"type":      "call_started",
			"details":   "Call from 415-555-1234",
		},
		{
			"timestamp": time.Now().Add(-3 * time.Minute).Format(time.RFC3339),
			"type":      "dtmf_received",
			"details":   "Pressed 2 for temperature",
		},
		{
			"timestamp": time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
			"type":      "response_sent",
			"details":   "Temperature: 72°F",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// HandleWebSocket manages WebSocket connections for real-time updates
func (h *WRDashboardHandler) HandleWebSocket(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Register new client
	h.clients[conn] = true
	log.Printf("WebSocket client connected. Total clients: %d", len(h.clients))

	// Subscribe to events
	eventListener := events.Manager.Subscribe()
	defer events.Manager.Unsubscribe(eventListener)

	// Send initial state
	activeCalls := events.Manager.GetActiveCalls()
	err = conn.WriteJSON(gin.H{
		"type":  "initial",
		"calls": activeCalls,
	})
	if err != nil {
		log.Printf("Error sending initial state: %v", err)
		delete(h.clients, conn)
		return
	}

	// Create channels for coordinating goroutines
	done := make(chan struct{})

	// Handle incoming messages from client (ping/pong)
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				return
			}
		}
	}()

	// Send events to this client
	for {
		select {
		case event := <-eventListener:
			// Send event to client
			err := conn.WriteJSON(gin.H{
				"type":  "event",
				"event": event,
			})
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				delete(h.clients, conn)
				return
			}

			// Also send updated active calls
			if event.EventType == "call_started" || event.EventType == "call_ended" {
				activeCalls := events.Manager.GetActiveCalls()
				err = conn.WriteJSON(gin.H{
					"type":  "update",
					"calls": activeCalls,
				})
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					delete(h.clients, conn)
					return
				}
			}

		case <-done:
			// Client disconnected
			delete(h.clients, conn)
			log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))
			return
		}
	}
}

// broadcastEvents sends events to all connected clients
func (h *WRDashboardHandler) broadcastEvents() {
	for event := range h.broadcast {
		for client := range h.clients {
			err := client.WriteJSON(gin.H{
				"type":  "event",
				"event": event,
			})
			if err != nil {
				log.Printf("Broadcast error: %v", err)
				client.Close()
				delete(h.clients, client)
			}
		}
	}
}

// TestCall simulates an incoming call for testing
func (h *WRDashboardHandler) TestCall(c *gin.Context) {
	// Test phone numbers from different cities
	testNumbers := []string{
		"4155551234", // San Francisco
		"2125551234", // New York
		"3125551234", // Chicago
		"5125551234", // Austin
		"7025551234", // Las Vegas
		"3055551234", // Miami
		"2065551234", // Seattle
		"6175551234", // Boston
	}

	// Pick a random number
	randomNum := testNumbers[rand.Intn(len(testNumbers))]
	areaCode := randomNum[:3]

	// Look up location
	location, exists := services.CompleteAreaCodes[areaCode]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid test number",
		})
		return
	}

	// Generate IDs
	sessionID := fmt.Sprintf("test_%s_%d", areaCode, time.Now().Unix())
	callID := fmt.Sprintf("call_%d", time.Now().Unix())

	// Send call started event
	startEvent := events.CallEvent{
		SessionID: sessionID,
		CallID:    callID,
		CallerNum: randomNum,
		AreaCode:  areaCode,
		Location:  fmt.Sprintf("%s, %s", location.City, location.State),
		EventType: "call_started",
		Details:   "Test call initiated",
		Timestamp: time.Now(),
	}
	events.SendEvent(startEvent)

	// Simulate DTMF after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)

		// Random button press
		digits := []string{"1", "2", "3"}
		digit := digits[rand.Intn(len(digits))]

		dtmfEvent := events.CallEvent{
			SessionID: sessionID,
			CallID:    callID,
			CallerNum: randomNum,
			AreaCode:  areaCode,
			Location:  fmt.Sprintf("%s, %s", location.City, location.State),
			EventType: "dtmf_received",
			Details:   fmt.Sprintf("Pressed %s", digit),
			Timestamp: time.Now(),
		}
		events.SendEvent(dtmfEvent)

		// Simulate response after 1 second
		time.Sleep(1 * time.Second)

		var responseDetail string
		switch digit {
		case "1":
			responseDetail = "Local time: 3:45 PM"
		case "2":
			responseDetail = fmt.Sprintf("Temperature: %d°F", rand.Intn(30)+50)
		case "3":
			responseDetail = fmt.Sprintf("AQI: %d (Good)", rand.Intn(50)+20)
		}

		responseEvent := events.CallEvent{
			SessionID: sessionID,
			CallID:    callID,
			CallerNum: randomNum,
			AreaCode:  areaCode,
			Location:  fmt.Sprintf("%s, %s", location.City, location.State),
			EventType: "response_sent",
			Details:   responseDetail,
			Timestamp: time.Now(),
		}
		events.SendEvent(responseEvent)

		// End call after 2 seconds
		time.Sleep(2 * time.Second)

		endEvent := events.CallEvent{
			SessionID: sessionID,
			CallID:    callID,
			CallerNum: randomNum,
			AreaCode:  areaCode,
			Location:  fmt.Sprintf("%s, %s", location.City, location.State),
			EventType: "call_ended",
			Details:   "Test call completed",
			Timestamp: time.Now(),
		}
		events.SendEvent(endEvent)
	}()

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "Test call initiated",
		"call_id":  callID,
		"caller":   randomNum,
		"location": fmt.Sprintf("%s, %s", location.City, location.State),
	})
}

// SimulateCall is an alias for TestCall for compatibility
func (h *WRDashboardHandler) SimulateCall(c *gin.Context) {
	h.TestCall(c)
}
