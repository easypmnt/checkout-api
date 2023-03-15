package websocketrpc

import (
	"sync"
)

// responseCallbacks is a map of request ID to response callback.
type responseCallbacks struct {
	sync.RWMutex
	m map[uint64]ResponseCallback
}

// newResponseCallbacks returns a new responseCallbacks.
func newResponseCallbacks() *responseCallbacks {
	return &responseCallbacks{
		m: make(map[uint64]ResponseCallback),
	}
}

// Set sets the response callback for the given request ID.
func (rc *responseCallbacks) Set(id uint64, cb ResponseCallback) {
	rc.Lock()
	defer rc.Unlock()
	rc.m[id] = cb
}

// Get gets the response callback for the given request ID.
func (rc *responseCallbacks) Get(id uint64) (ResponseCallback, bool) {
	rc.RLock()
	defer rc.RUnlock()

	cb, ok := rc.m[id]
	if ok && cb != nil {
		return cb, true
	}
	return nil, false
}

// Delete deletes the response callback for the given request ID.
func (rc *responseCallbacks) Delete(id uint64) {
	rc.Lock()
	defer rc.Unlock()
	delete(rc.m, id)
}

// subscriptions is a map of subscription ID to event name.
type subscriptions struct {
	sync.RWMutex
	m map[float64]string
}

// newSubscriptions returns a new subscriptions.
func newSubscriptions() *subscriptions {
	return &subscriptions{
		m: make(map[float64]string),
	}
}

// Set sets the event name for the given subscription ID.
func (s *subscriptions) Set(id float64, name string) {
	s.Lock()
	defer s.Unlock()
	s.m[id] = name
}

// Get gets the event name for the given subscription ID.
func (s *subscriptions) Get(id float64) (string, bool) {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.m[id]
	if ok && v != "" {
		return v, true
	}
	return "", false
}

// Delete deletes the event name for the given subscription ID.
func (s *subscriptions) Delete(id float64) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, id)
}

// GetAll gets all subscriptions.
func (s *subscriptions) GetAll() map[float64]string {
	s.RLock()
	defer s.RUnlock()
	return s.m
}

// Len returns the number of subscriptions.
func (s *subscriptions) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.m)
}

// GetKeyByValue gets the key for the given value.
func (s *subscriptions) GetKeyByValue(value string) (float64, bool) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.m {
		if v == value {
			return k, true
		}
	}
	return 0, false
}
