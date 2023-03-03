package websocketrpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket JSON-RPC client
type Client struct {
	conn          *websocket.Conn
	lock          sync.RWMutex
	done          chan struct{}
	nextReqID     int
	eventHandlers map[string][]func(json.RawMessage)
	errorHandler  func(error)
}

// NewClient creates a new WebSocket JSON-RPC client
func NewClient(endpoint string, errorHandler func(error)) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing endpoint URL: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error dialing endpoint: %v", err)
	}

	c := &Client{
		conn:          conn,
		eventHandlers: make(map[string][]func(json.RawMessage)),
		done:          make(chan struct{}),
		nextReqID:     1,
		errorHandler: func(err error) {
			log.Printf("websocket client error: %v", err)
		},
	}

	if errorHandler != nil {
		c.errorHandler = errorHandler
	}

	return c, nil
}

// Stop gracefully stops the client by closing the WebSocket connection
func (c *Client) Stop() error {
	close(c.done)
	return c.conn.Close()
}

// SendEvent sends a JSON-RPC event to the server
func (c *Client) SendEvent(method string, params []interface{}) error {
	event := &Request{
		Version: "2.0",
		Method:  method,
		Params:  params,
	}

	if err := c.conn.WriteJSON(event); err != nil {
		return fmt.Errorf("error sending JSON-RPC event: %v", err)
	}

	return nil
}

// SendRequest sends a JSON-RPC request to the server and returns the response
func (c *Client) SendRequest(method string, params []interface{}) (*Response, error) {
	req := &Request{
		Version: "2.0",
		ID:      c.nextReqID,
		Method:  method,
		Params:  params,
	}

	if err := c.conn.WriteJSON(req); err != nil {
		return nil, fmt.Errorf("error sending JSON-RPC request: %v", err)
	}

	c.nextReqID++

	var res Response
	if err := c.conn.ReadJSON(&res); err != nil {
		return nil, fmt.Errorf("error reading JSON-RPC response: %v", err)
	}

	return &res, nil
}

// RegisterEventHandler registers an event handler for the given event name.
// The handler will be called when an event with the given name is received.
func (c *Client) RegisterEventHandler(name string, handler func(json.RawMessage)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.eventHandlers[name] = append(c.eventHandlers[name], handler)
}

// Run starts the client and listens for incoming events and responses
func (c *Client) Run() {
	go c.listen()

	<-c.done
}

func (c *Client) listen() {
	for {
		var event Event
		if err := c.conn.ReadJSON(&event); err != nil {
			c.errorHandler(fmt.Errorf("error reading JSON-RPC event: %v", err))
			return
		}

		if event.Params == nil {
			continue
		}

		// Event
		c.lock.RLock()
		handlers, ok := c.eventHandlers[event.Method]
		c.lock.RUnlock()
		if !ok {
			continue
		}

		for _, handler := range handlers {
			handler(event.Params)
		}
	}
}

// Subscribe subscribes to the given event name and returns a subscription id or an error.
func (c *Client) Subscribe(name RequestType, params []interface{}) (int, error) {
	res, err := c.SendRequest(string(name), params)
	if err != nil {
		return 0, err
	}

	if res.Error != nil {
		return 0, fmt.Errorf("error subscribing to event: %v", res.Error)
	}

	return res.Result.(int), nil
}

// Unsubscribe unsubscribes from the given event name and subscription id or returns an error.
func (c *Client) Unsubscribe(name RequestType, id int) error {
	res, err := c.SendRequest(string(name), []interface{}{id})
	if err != nil {
		return err
	}

	if res.Error != nil {
		return fmt.Errorf("error unsubscribing from event: %v", res.Error)
	}

	return nil
}
