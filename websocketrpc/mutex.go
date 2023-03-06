package websocketrpc

import "sync"

// responseCallbacks is a map of request ID to response callback.
type responseCallbacks struct {
	sync.RWMutex
	m map[interface{}]ResponseCallback
}

// newResponseCallbacks returns a new responseCallbacks.
func newResponseCallbacks() *responseCallbacks {
	return &responseCallbacks{
		m: make(map[interface{}]ResponseCallback),
	}
}

// Set sets the response callback for the given request ID.
func (rc *responseCallbacks) Set(id interface{}, cb ResponseCallback) {
	rc.Lock()
	defer rc.Unlock()
	rc.m[id] = cb
}

// Get gets the response callback for the given request ID.
func (rc *responseCallbacks) Get(id interface{}) (ResponseCallback, bool) {
	rc.RLock()
	defer rc.RUnlock()
	cb, ok := rc.m[id]
	return cb, ok
}

// Delete deletes the response callback for the given request ID.
func (rc *responseCallbacks) Delete(id interface{}) {
	rc.Lock()
	defer rc.Unlock()
	delete(rc.m, id)
}

// eventHandlers is a map of event name to event handler.
type eventHandlers struct {
	sync.RWMutex
	m map[string]EventHandler
}

// newEventHandlers returns a new eventHandlers.
func newEventHandlers() *eventHandlers {
	return &eventHandlers{
		m: make(map[string]EventHandler),
	}
}

// Set sets the event handler for the given event name.
func (eh *eventHandlers) Set(name string, h EventHandler) {
	eh.Lock()
	defer eh.Unlock()
	eh.m[name] = h
}

// Get gets the event handler for the given event name.
func (eh *eventHandlers) Get(name string) (EventHandler, bool) {
	eh.RLock()
	defer eh.RUnlock()
	h, ok := eh.m[name]
	return h, ok
}

// Delete deletes the event handler for the given event name.
func (eh *eventHandlers) Delete(name string) {
	eh.Lock()
	defer eh.Unlock()
	delete(eh.m, name)
}

// subscriptions is a map of subscription ID to event name.
type subscriptions struct {
	sync.RWMutex
	m map[int64]string
}

// newSubscriptions returns a new subscriptions.
func newSubscriptions() *subscriptions {
	return &subscriptions{
		m: make(map[int64]string),
	}
}

// Set sets the event name for the given subscription ID.
func (s *subscriptions) Set(id int64, name string) {
	s.Lock()
	defer s.Unlock()
	s.m[id] = name
}

// Get gets the event name for the given subscription ID.
func (s *subscriptions) Get(id int64) (string, bool) {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.m[id]
	return v, ok
}

// Delete deletes the event name for the given subscription ID.
func (s *subscriptions) Delete(id int64) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, id)
}

// GetAll gets all subscriptions.
func (s *subscriptions) GetAll() map[int64]string {
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
