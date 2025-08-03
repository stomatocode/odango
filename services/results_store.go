// services/results_store.go
package services

import (
	"sync"
	"time"
)

// ResultsStore provides temporary in-memory storage for CDR results
// This can be easily replaced with Redis, database, or other storage in the future
type ResultsStore struct {
	mu      sync.RWMutex
	results map[string]*CDRDiscoveryResult
	ttl     time.Duration // Time to live for stored results
}

// GlobalResultsStore is the singleton instance used throughout the application
var GlobalResultsStore = NewResultsStore(1 * time.Hour)

// NewResultsStore creates a new results store with specified TTL
func NewResultsStore(ttl time.Duration) *ResultsStore {
	return &ResultsStore{
		results: make(map[string]*CDRDiscoveryResult),
		ttl:     ttl,
	}
}

// Store saves a CDR discovery result with automatic expiration
func (rs *ResultsStore) Store(sessionID string, result *CDRDiscoveryResult) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Store the result
	rs.results[sessionID] = result

	// Schedule cleanup after TTL
	go func() {
		time.Sleep(rs.ttl)
		rs.Delete(sessionID)
	}()
}

// Get retrieves a CDR discovery result by session ID
func (rs *ResultsStore) Get(sessionID string) (*CDRDiscoveryResult, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	result, exists := rs.results[sessionID]
	return result, exists
}

// Delete removes a result from storage
func (rs *ResultsStore) Delete(sessionID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	delete(rs.results, sessionID)
}

// GetAll returns all stored results (useful for admin/debugging)
func (rs *ResultsStore) GetAll() map[string]*CDRDiscoveryResult {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	// Create a copy to avoid race conditions
	resultsCopy := make(map[string]*CDRDiscoveryResult)
	for k, v := range rs.results {
		resultsCopy[k] = v
	}

	return resultsCopy
}

// Count returns the number of stored results
func (rs *ResultsStore) Count() int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	return len(rs.results)
}

// Clear removes all stored results
func (rs *ResultsStore) Clear() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.results = make(map[string]*CDRDiscoveryResult)
}

// UpdateTTL updates the time-to-live for new results
func (rs *ResultsStore) UpdateTTL(ttl time.Duration) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.ttl = ttl
}
