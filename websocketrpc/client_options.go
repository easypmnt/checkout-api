package websocketrpc

// WithLogger sets the logger for the client.
func WithLogger(l logger) ClientOption {
	return func(c *Client) {
		c.log = l
	}
}

// WithEventHandler sets an event handler for the client.
func WithEventHandler(eventName string, handler EventHandler) ClientOption {
	return func(c *Client) {
		c.eventHandlers.Set(eventName, handler)
	}
}
