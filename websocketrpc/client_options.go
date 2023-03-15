package websocketrpc

// WithLogger sets the logger for the client.
func WithLogger(l logger) ClientOption {
	return func(c *Client) {
		c.log = l
	}
}

// WithEventsEmitter sets the events emitter for the client.
func WithEventsEmitter(e eventsEmitter) ClientOption {
	return func(c *Client) {
		c.emitter = e
	}
}
