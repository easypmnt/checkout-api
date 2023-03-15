package sse

import (
	"fmt"
	"time"

	"github.com/gin-contrib/sse"
)

type (
	// Event struct
	Event struct {
		ID        int64
		Data      EventData
		TTL       int64 `json:"-"`
		Timestamp int64 `json:"-"`
	}

	// EventData struct
	EventData struct {
		Name    string      `json:"name"`
		Payload interface{} `json:"payload"`
	}
)

// MapToSseEvent func to convert struct Event to sse.Event
func (e Event) MapToSseEvent() sse.Event {
	t := time.Unix(0, e.ID)
	id := fmt.Sprintf("%d:%d", t.Unix(), t.Nanosecond())
	return sse.Event{
		Id:    id,
		Event: "message",
		Data:  e.Data,
		Retry: 500,
	}
}

// IsExpired func
func (e Event) IsExpired() bool {
	if e.TTL > 0 {
		return time.Now().After(time.Unix(0, e.Timestamp).Add(time.Duration(e.TTL) * time.Second))
	}
	return false
}
