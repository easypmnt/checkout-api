package websocketrpc

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type (
	// WebSocket client for Solana RPC.
	WSClient struct {
		config       *Config
		conn         *websocket.Conn
		eventHandler func(*Event)
		errorHandler func(error)
		done         chan struct{}
	}

	Config struct {
		URL            string
		ReconnectDelay time.Duration
	}

	JSONRPCRequest struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      int         `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}

	JSONRPCResponse struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}
)

func NewWSClient(config *Config, errorHandler func(error)) (*WSClient, error) {
	u, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}

	c := &WSClient{
		config:       config,
		eventHandler: nil,
		errorHandler: errorHandler,
		done:         make(chan struct{}),
	}

	if err := c.connect(u); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *WSClient) Subscribe(eventHandler func(*Event)) error {
	if c.conn == nil {
		return errors.New("websocket connection not initialized")
	}

	c.eventHandler = eventHandler

	return nil
}

func (c *WSClient) Close() error {
	if c.conn == nil {
		return errors.New("websocket connection not initialized")
	}

	close(c.done)
	return c.conn.Close()
}

func (c *WSClient) connect(u *url.URL) error {
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		c.errorHandler(err)

		// Try to reconnect
		go func() {
			log.Printf("Reconnecting in %v...", c.config.ReconnectDelay)
			time.Sleep(c.config.ReconnectDelay)
			if err := c.connect(u); err != nil {
				log.Println("Reconnect failed:", err)
			}
		}()

		return err
	}

	c.conn = conn

	go c.readEvents()

	return nil
}

func (c *WSClient) readEvents() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.errorHandler(err)
			return
		}

		var event Event
		err = json.Unmarshal(message, &event)
		if err != nil {
			c.errorHandler(err)
			continue
		}

		if c.eventHandler != nil {
			c.eventHandler(&event)
		}
	}
}
