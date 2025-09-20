package events

import (
	"sync"
	"time"
)

// CallEvent represents a Web Responder call event for the dashboard
type CallEvent struct {
	SessionID string    `json:"session_id"`
	CallID    string    `json:"call_id"`
	CallerNum string    `json:"caller_number"`
	AreaCode  string    `json:"area_code"`
	Location  string    `json:"location"`
	EventType string    `json:"event_type"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

// ActiveCall represents an ongoing call in the system
type ActiveCall struct {
	CallID     string    `json:"call_id"`
	SessionID  string    `json:"session_id"`
	CallerNum  string    `json:"caller_number"`
	AreaCode   string    `json:"area_code"`
	Location   string    `json:"location"`
	StartTime  time.Time `json:"start_time"`
	LastAction string    `json:"last_action"`
	Status     string    `json:"status"`
	Duration   string    `json:"duration"`
}

// EventManager handles event broadcasting and active call tracking
type EventManager struct {
	mu           sync.RWMutex
	activeCalls  map[string]*ActiveCall
	EventChannel chan CallEvent
	listeners    []chan CallEvent
}

// Global event manager instance
var Manager = &EventManager{
	activeCalls:  make(map[string]*ActiveCall),
	EventChannel: make(chan CallEvent, 100),
	listeners:    make([]chan CallEvent, 0),
}

// Start begins processing events
func (em *EventManager) Start() {
	go func() {
		for event := range em.EventChannel {
			em.processEvent(event)
			em.broadcast(event)
		}
	}()
}

// Subscribe adds a new listener for events
func (em *EventManager) Subscribe() chan CallEvent {
	em.mu.Lock()
	defer em.mu.Unlock()

	listener := make(chan CallEvent, 50)
	em.listeners = append(em.listeners, listener)
	return listener
}

// Unsubscribe removes a listener
func (em *EventManager) Unsubscribe(listener chan CallEvent) {
	em.mu.Lock()
	defer em.mu.Unlock()

	for i, l := range em.listeners {
		if l == listener {
			em.listeners = append(em.listeners[:i], em.listeners[i+1:]...)
			close(listener)
			break
		}
	}
}

// processEvent updates active calls based on event type
func (em *EventManager) processEvent(event CallEvent) {
	em.mu.Lock()
	defer em.mu.Unlock()

	switch event.EventType {
	case "call_started":
		em.activeCalls[event.CallID] = &ActiveCall{
			CallID:     event.CallID,
			SessionID:  event.SessionID,
			CallerNum:  event.CallerNum,
			AreaCode:   event.AreaCode,
			Location:   event.Location,
			StartTime:  event.Timestamp,
			LastAction: "Started",
			Status:     "active",
		}

	case "dtmf_received":
		if call, exists := em.activeCalls[event.CallID]; exists {
			call.LastAction = event.Details
		}

	case "response_sent":
		if call, exists := em.activeCalls[event.CallID]; exists {
			call.LastAction = event.Details
		}

	case "call_ended":
		delete(em.activeCalls, event.CallID)
	}

	// Update duration for all active calls
	for _, call := range em.activeCalls {
		call.Duration = time.Since(call.StartTime).Round(time.Second).String()
	}
}

// broadcast sends event to all listeners
func (em *EventManager) broadcast(event CallEvent) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	for _, listener := range em.listeners {
		select {
		case listener <- event:
		default:
			// Don't block if listener is full
		}
	}
}

// GetActiveCalls returns current active calls
func (em *EventManager) GetActiveCalls() []ActiveCall {
	em.mu.RLock()
	defer em.mu.RUnlock()

	calls := make([]ActiveCall, 0, len(em.activeCalls))
	for _, call := range em.activeCalls {
		calls = append(calls, *call)
	}
	return calls
}

// SendEvent is a helper to send events to the manager
func SendEvent(event CallEvent) {
	select {
	case Manager.EventChannel <- event:
	default:
		// Channel full, drop event
	}
}
