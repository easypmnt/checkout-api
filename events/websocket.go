package events

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

type (
	Event struct {
		Channel string      `json:"-"`
		Name    string      `json:"name"`
		Data    interface{} `json:"data"`
	}

	EventBroadcaster struct {
		clients   *channelHub
		broadcast chan Event
		upgrader  websocket.Upgrader
		emitter   Emitter
		log       Logger
	}
)

func NewEventBroadcaster(emitter Emitter, log Logger) *EventBroadcaster {
	b := &EventBroadcaster{
		clients:   newChannelHub(),
		broadcast: make(chan Event, 100),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		emitter: emitter,
		log:     log,
	}

	b.emitter.ListenEvents(b.RetranslateEvents, AllEvents...)

	return b
}

// RetranslateEvents retranslates events from the emitter to the event Broadcaster.
func (b *EventBroadcaster) RetranslateEvents(event EventName, payload interface{}) error {
	data, ok := payload.(PaymentIDGetter)
	if !ok {
		return fmt.Errorf("event Broadcaster: retranslate events: invalid payload: %T", payload)
	}

	b.broadcast <- Event{
		Channel: data.GetPaymentID(),
		Name:    string(event),
		Data:    payload,
	}

	return nil
}

// Run starts the event Broadcaster.
func (b *EventBroadcaster) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				b.log.Infof("event Broadcaster: stopped")
				if ctx.Err() == context.Canceled {
					return nil
				} else if ctx.Err() == context.DeadlineExceeded {
					return ctx.Err()
				}
				return nil
			case event := <-b.broadcast:
				clients := b.clients.Get(event.Channel)
				for _, client := range clients {
					if err := client.WriteJSON(event); err != nil {
						client.Close()
						b.clients.Remove(event.Channel, client)
					}
				}
			}
		}
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("event Broadcaster: run: %w", err)
	}

	return nil
}

func (b *EventBroadcaster) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "channel")
	if channelID == "" {
		http.Error(w, "channel id is required", http.StatusBadRequest)
		return
	}

	conn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection to websocket: %v", err)
		return
	}

	b.clients.Add(channelID, conn)
	defer func() {
		b.clients.Remove(channelID, conn)
		conn.Close()
	}()

	<-r.Context().Done()
	b.log.Infof("event Broadcaster: websocket connection closed by client")

	// for {
	// 	select {
	// 	case <-r.Context().Done():
	// 		b.log.Infof("event Broadcaster: websocket connection closed by client")
	// 		return
	// 	default:
	// 		if _, _, err := conn.ReadMessage(); err != nil {
	// 			log.Printf("error reading message from websocket: %v", err)
	// 			break
	// 		}
	// 	}
	// }
}

func newChannelHub() *channelHub {
	return &channelHub{
		clients: make(map[string][]*websocket.Conn),
	}
}

type channelHub struct {
	sync.RWMutex
	clients map[string][]*websocket.Conn
}

func (h *channelHub) Add(channel string, conn *websocket.Conn) {
	h.Lock()
	defer h.Unlock()

	h.clients[channel] = append(h.clients[channel], conn)
}

func (h *channelHub) Remove(channel string, conn *websocket.Conn) {
	h.Lock()
	defer h.Unlock()

	clients := h.clients[channel]
	for i, c := range clients {
		if c == conn {
			clients = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}

func (h *channelHub) Get(channel string) []*websocket.Conn {
	h.RLock()
	defer h.RUnlock()

	return h.clients[channel]
}

// MakeHTTPHandler returns a handler that makes a set of endpoints available on
// predefined paths.
func MakeHTTPHandler(b *EventBroadcaster) http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("/channel/{channel}", b.handleWebSocket)

	return r
}
