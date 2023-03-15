package sse

import (
	"net/http"

	"github.com/gin-contrib/sse"
	"github.com/go-chi/chi/v5"
)

type (
	logger interface {
		Debugf(format string, args ...interface{})
		Infof(format string, args ...interface{})
		Warnf(format string, args ...interface{})
		Errorf(format string, args ...interface{})
	}

	sseServer interface {
		SubscribeToChannel(channelID, lastEventID string) (chan interface{}, []Event, error)
		Unsubscribe(channelID string, listener chan interface{}) error
	}
)

// MakeHTTPHandler returns a handler that makes a set of endpoints available on
// predefined paths.
func MakeHTTPHandler(s sseServer, log logger) http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("/channel/{channel}", subscribeToChannel(s, log))

	return r
}

func subscribeToChannel(s sseServer, log logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := chi.URLParam(r, "channel")
		if channelID == "" {
			log.Debugf("missed channel id")
			http.Error(w, "Missed channel id!", http.StatusBadRequest)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			log.Warnf("streaming unsupported: channel %s", channelID)
			http.Error(w, "Streaming unsupported!", http.StatusNotImplemented)
			return
		}

		listener, history, err := s.SubscribeToChannel(channelID, getLastEventID(r))
		if err != nil {
			log.Errorf("subscribe to channel %s with last event id %s", channelID, getLastEventID(r))
			http.Error(w, "Could not subscribe to events channel", http.StatusInternalServerError)
			return
		}
		defer s.Unsubscribe(channelID, listener)

		// Set the headers related to event streaming.
		if err := openHTTPConnection(w, r); err != nil {
			log.Errorf("open http connection: %s", err.Error())
			return
		}
		flusher.Flush()

		log.Debugf("[client_connected] client connected: %s", channelID)

		// send historical events
		for _, event := range history {
			if !event.IsExpired() {
				err := sse.Encode(w, event.MapToSseEvent())
				if err != nil {
					log.Errorf("sse encoding: %s (channel id: %s, event: %#v)", err.Error(), channelID, event)
					return
				}
				flusher.Flush()
				log.Debugf("[channel_received_events_history] channel %s: received event: %+v", channelID, event)
			}
		}

		for {
			select {
			case <-r.Context().Done():
				log.Debugf("[client_disconnected] client closed connection: %s", channelID)
				return
			case event := <-listener:
				if e, ok := event.(Event); ok {
					err := sse.Encode(w, e.MapToSseEvent())
					if err != nil {
						log.Errorf("sse encoding: %s (channel id: %s, event: %#v)", err.Error(), channelID, event)
						return
					}
					flusher.Flush()
					log.Debugf("[channel_received_event] channel %s: received event: %+v", channelID, event)
				} else {
					log.Errorf("event is not Event type: %#v", event)
				}
			}
		}
	}
}

func getLastEventID(r *http.Request) string {
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID == "" {
		lastEventID = r.URL.Query().Get("last_event_id")
	}
	return lastEventID
}

// Set the headers related to event streaming.
func openHTTPConnection(w http.ResponseWriter, r *http.Request) error {
	origin := r.Header.Get("Origin")
	if origin == "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	return sse.Encode(w, sse.Event{
		Event: "notification",
		Data:  "SSE connection successfully established",
	})
}
