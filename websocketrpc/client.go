package websocketrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
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
	}

	ClientOption     func(*Client)
	EventHandler     func(base58Addr string, event json.RawMessage) error
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
	err := c.sendRequest(&Request{
		Version: "2.0",
		ID:      c.nextReqID,
		Method:  SubscribeAccountRequest,
		Params:  AccountSubscribeRequestPayload(base58Addr),
	}, func(resp json.RawMessage, err error) error {
		if err.Error() != "" {
			return fmt.Errorf("websocketrpc: subscribe: %w", err)
		}

		var jsonN json.Number
		if err := json.Unmarshal(resp, &jsonN); err != nil {
			return fmt.Errorf("websocketrpc: subscribe: %w", err)
		}

		subID, err := jsonN.Float64()
		if err != nil {
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
func (c *Client) Unsubscribe(subID float64) error {
	err := c.sendRequest(&Request{
		Version: "2.0",
		ID:      c.nextReqID,
		Method:  UnsubscribeAccountRequest,
		Params:  AccountUnsubscribeRequestPayload(subID),
	}, func(resp json.RawMessage, err error) error {
		if err.Error() != "" {
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

// UnsubscribeByAddress unsubscribes from account notifications for the given wallet address.
func (c *Client) UnsubscribeByAddress(base58Addr string) error {
	subID, ok := c.subscriptions.GetKeyByValue(base58Addr)
	if !ok {
		return fmt.Errorf("websocketrpc: unsubscribe by address: no subscription found for address %s", base58Addr)
	}

	return c.Unsubscribe(subID)
}

// unsubscribeAll unsubscribes from all account notifications.
func (c *Client) unsubscribeAll() error {
	subscriptions := c.subscriptions.GetAll()
	for subID := range subscriptions {
		if err := c.Unsubscribe(subID); err != nil {
			c.log.Errorf("websocketrpc: unsubscribing all: %v", err)
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
				return nil // all subscriptions removed
			}
		case <-time.After(15 * time.Second):
			return fmt.Errorf("websocketrpc: unsubscribing all: timed out")
		}
	}
}

// sendRequest sends a JSON-RPC v2 request to the websocket server.
// The response is returned as a json.RawMessage or an error.
func (c *Client) sendRequest(req *Request, callback ResponseCallback) error {
	if c.conn == nil {
		return ErrConnectionClosed
	}

	if req.ID > 0 && callback != nil {
		c.responseCallbacks.Set(req.ID, callback)
	}

	c.reqChan <- req
	atomic.AddUint64(&c.nextReqID, 1)

	return nil
}

// listener function listens for incoming JSON-RPC v2 events and notifications.
// It calls the appropriate callback function.
func (c *Client) listener() error {
	for {
		if c.conn == nil {
			return ErrConnectionClosed
		}

		var msg json.RawMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			if e, ok := err.(*websocket.CloseError); ok {
				return fmt.Errorf("websocketrpc: listen: connection closed with code %d (%s)", e.Code, e.Text)
			}
			continue
		}

		var parsedMsg messagePayload
		if err := json.Unmarshal(msg, &parsedMsg); err != nil {
			c.log.Errorf("websocketrpc: listen: error unmarshaling event: %v", err)
			continue
		}

		if parsedMsg.IsEvent() {
			event := parsedMsg.GetEvent()
			c.eventChan <- event
		} else if parsedMsg.IsResponse() {
			resp := parsedMsg.GetResponse()
			c.respChan <- resp
		}
	}
}

// runner function runs the websocket rpc service.
func (c *Client) runner() error {
	for {
		select {
		case req, open := <-c.reqChan:
			if open {
				if err := c.conn.WriteJSON(req); err != nil {
					c.log.Errorf("websocketrpc: run: error writing request: %v", err)
				}
			}
		case event, open := <-c.eventChan:
			if open {
				if h, ok := c.eventHandlers.Get(event.Method); ok {
					if sid, err := event.Params.Subscription.Float64(); err == nil && sid > 0 {
						base58Addr, ok := c.subscriptions.Get(sid)
						if !ok {
							c.log.Errorf("websocketrpc: run: error handling event: subscription ID %d not found", sid)
							continue
						}
						if err := h(base58Addr, event.Params.Result); err != nil {
							c.log.Errorf("websocketrpc: run: error handling event: %v", err)
						}
					}
				}
			}
		case resp, open := <-c.respChan:
			if open {
				if callback, ok := c.responseCallbacks.Get(resp.ID); ok {
					c.responseCallbacks.Delete(resp.ID)
					if err := callback(resp.Result, resp.Error); err != nil {
						c.log.Errorf("websocketrpc: run: error handling response: %v", err)
					}
				}
			}
		}
	}
}

// Run websocket rpc service.
func (c *Client) Run(ctx context.Context) error {
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(c.listener)
	eg.Go(c.runner)

	c.log.Infof("websocketrpc: running...")

	<-ctx.Done()
	c.log.Infof("websocketrpc: run: context done, stopping...")
	eg.Go(c.unsubscribeAll)

	if err := eg.Wait(); err != nil {
		c.log.Errorf("websocketrpc: run: error: %v", err)
	}

	c.conn = nil

	// Close all channels.
	close(c.reqChan)
	close(c.respChan)
	close(c.eventChan)

	c.log.Infof("websocketrpc: stopped")

	return nil
}
