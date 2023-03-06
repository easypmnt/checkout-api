package websocketrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type (
	Client struct {
		conn *websocket.Conn
		log  logger

		nextReqID uint64

		subscriptions     *subscriptions
		eventHandlers     *eventHandlers
		responseCallbacks *responseCallbacks

		reqChan   chan *Request
		respChan  chan *Response
		eventChan chan *Event
		done      chan bool
	}

	ClientOption     func(*Client)
	EventHandler     func(json.RawMessage) error
	ResponseCallback func(json.RawMessage, error) error

	logger interface {
		Infof(format string, args ...interface{})
		Errorf(format string, args ...interface{})
	}
)

// NewClient creates a new websocket rpc client.
// It accepts a websocket connection and optional client options.
func NewClient(conn *websocket.Conn, opts ...ClientOption) *Client {
	c := &Client{
		conn:      conn,
		nextReqID: 1,

		subscriptions:     newSubscriptions(),
		eventHandlers:     newEventHandlers(),
		responseCallbacks: newResponseCallbacks(),

		reqChan:   make(chan *Request, 1000),
		respChan:  make(chan *Response, 1000),
		eventChan: make(chan *Event, 1000),
		done:      make(chan bool),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.log == nil {
		c.log = logrus.New()
	}

	return c
}

// SetEventHandler sets the event handler for the given event name.
func (c *Client) SetEventHandler(eventName string, handler EventHandler) {
	c.eventHandlers.Set(eventName, handler)
}

// RemoveEventHandler removes the event handler for the given event name.
func (c *Client) RemoveEventHandler(eventName string) {
	c.eventHandlers.Delete(eventName)
}

// Subscribe subscribes for account notifications to the given wallet address.
func (c *Client) Subscribe(base58Addr string) error {
	c.log.Infof("websocketrpc: subscribing to account %s", base58Addr)
	err := c.sendRequest(&Request{
		Version: "2.0",
		ID:      c.nextReqID,
		Method:  SubscribeAccountRequest,
		Params:  AccountSubscribeRequestPayload(base58Addr),
	}, func(resp json.RawMessage, err error) error {
		if err != nil {
			return fmt.Errorf("websocketrpc: subscribe: %w", err)
		}

		var subID int64
		if err := json.Unmarshal(resp, &subID); err != nil {
			return fmt.Errorf("websocketrpc: subscribe: %w", err)
		}

		if subID == 0 {
			return fmt.Errorf("websocketrpc: subscribe: failed to subscribe")
		}

		c.subscriptions.Set(subID, base58Addr)
		c.log.Infof("websocketrpc: subscribed to account %s with subscription ID %d", base58Addr, subID)

		return nil
	})
	if err != nil {
		return fmt.Errorf("websocketrpc: subscribe: %w", err)
	}

	return nil
}

// Unsubscribe unsubscribes from account notifications for the given subscription ID.
func (c *Client) Unsubscribe(subID int64) error {
	c.log.Infof("websocketrpc: unsubscribing from account with subscription ID %d", subID)
	err := c.sendRequest(&Request{
		Version: "2.0",
		ID:      c.nextReqID,
		Method:  UnsubscribeAccountRequest,
		Params:  AccountUnsubscribeRequestPayload(subID),
	}, func(resp json.RawMessage, err error) error {
		if err != nil {
			return fmt.Errorf("websocketrpc: unsubscribe: %w", err)
		}

		var result bool
		if err := json.Unmarshal(resp, &result); err != nil {
			return fmt.Errorf("websocketrpc: unsubscribe: %w", err)
		}

		if !result {
			return fmt.Errorf("websocketrpc: unsubscribe: failed to unsubscribe")
		}

		c.subscriptions.Delete(subID)
		c.log.Infof("websocketrpc: unsubscribed from account with subscription ID %d", subID)

		return nil
	})
	if err != nil {
		return fmt.Errorf("websocketrpc: unsubscribe: %w", err)
	}

	return nil
}

// unsubscribeAll unsubscribes from all account notifications.
func (c *Client) unsubscribeAll() error {
	c.log.Infof("websocketrpc: unsubscribing from all accounts")

	subscriptions := c.subscriptions.GetAll()
	for subID := range subscriptions {
		if err := c.Unsubscribe(subID); err != nil {
			return fmt.Errorf("websocketrpc: unsubscribe all: %w", err)
		}
	}

	// wait for all subscriptions to be removed
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if c.subscriptions.Len() == 0 {
				c.log.Infof("websocketrpc: unsubscribed from all accounts")
				c.done <- true
				return nil
			}
		case <-c.done:
			return nil
		}
	}
}

// sendRequest sends a JSON-RPC v2 request to the websocket server.
// The response is returned as a json.RawMessage or an error.
func (c *Client) sendRequest(req *Request, callback ResponseCallback) error {
	c.log.Infof("websocketrpc: sending request: %s", req)
	if c.conn == nil {
		return ErrConnectionClosed
	}

	if req.ID != nil && callback != nil {
		c.responseCallbacks.Set(req.ID, callback)
	}

	c.reqChan <- req
	atomic.AddUint64(&c.nextReqID, 1)

	c.log.Infof("websocketrpc: sent request: %s", req)
	return nil
}

// listen function listens for incoming JSON-RPC v2 events and notifications.
// It calls the appropriate callback function.
func (c *Client) listen() error {
	c.log.Infof("websocketrpc: listening for events")

	for {
		var msg json.RawMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			c.log.Errorf("websocketrpc: listen: error reading message: %v", err)
			continue
		}

		var parsedMsg messagePayload
		if err := json.Unmarshal(msg, &parsedMsg); err != nil {
			c.log.Errorf("websocketrpc: listen: error unmarshaling event: %v", err)
			continue
		}

		c.log.Infof("websocketrpc: received message: %v", parsedMsg)

		if parsedMsg.IsEvent() {
			c.eventChan <- &Event{
				Method: parsedMsg.Method,
				Params: parsedMsg.Params,
			}

			continue
		}

		if parsedMsg.IsResponse() {
			c.respChan <- &Response{
				Version: parsedMsg.Version,
				ID:      parsedMsg.ID,
				Result:  parsedMsg.Result,
				Error:   parsedMsg.Error,
			}

			continue
		}
	}
}

// run function runs the websocket rpc service.
func (c *Client) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			c.log.Infof("websocketrpc: run: context done")
			if err := c.unsubscribeAll(); err != nil {
				c.log.Errorf("websocketrpc: run: %v", err)
			}
			return nil
		case req := <-c.reqChan:
			c.log.Infof("websocketrpc: run: sending request: %s", req)
			if err := c.conn.WriteJSON(req); err != nil {
				c.log.Errorf("websocketrpc: run: error writing request: %v", err)
			}
		case event := <-c.eventChan:
			c.log.Infof("websocketrpc: run: received event: %s", event)
			if h, ok := c.eventHandlers.Get(event.Method); ok && h != nil {
				if err := h(event.Params); err != nil {
					c.log.Errorf("websocketrpc: run: error handling event: %v", err)
				}
			}
		case resp := <-c.respChan:
			c.log.Infof("websocketrpc: run: received response: %s", resp)
			if callback, ok := c.responseCallbacks.Get(resp.ID); ok {
				c.responseCallbacks.Delete(resp.ID)
				if err := callback(resp.Result, resp.Error); err != nil {
					c.log.Errorf("websocketrpc: run: error handling response: %v", err)
				}
			}
		}
	}
}

// Run websocket rpc service.
func (c *Client) Run(ctx context.Context) {
	go c.listen()
	go c.run(ctx)

	// Wait for the run function to finish.
	<-c.done
}
