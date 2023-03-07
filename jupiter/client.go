package jupiter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	// ContentTypeJSON is the content type for JSON.
	ContentTypeJSON = "application/json"
)

type (
	// Client is a Jupiter client that can be used to make requests to the Jupiter API.
	Client struct {
		client *http.Client

		apiURL            string
		endpointQuote     string
		endpointSwap      string
		endpointPrice     string
		endpointRoutesMap string
	}

	// ClientOption is a function that can be used to configure a Jupiter client.
	ClientOption func(*Client)

	// Response is a generic response structure.
	Response struct {
		Data        json.RawMessage `json:"data"`
		TimeTaken   int64           `json:"timeTaken"`
		ContextSlot int64           `json:"contextSlot"`
	}
)

// NewClient returns a new Jupiter client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},

		apiURL:            "https://quote-api.jup.ag/v4",
		endpointQuote:     "/quote",
		endpointSwap:      "/swap",
		endpointPrice:     "/price",
		endpointRoutesMap: "/indexed-route-map",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// get makes a GET request to the specified endpoint with the given parameters.
// It returns the response and any error encountered.
// The caller is responsible for closing the response body.
func (c *Client) get(endpoint string, params url.Values) (json.RawMessage, error) {
	parsedURL, err := url.Parse(c.apiURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if len(params) > 0 {
		parsedURL.RawQuery = params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	req.Header.Set("Accept", ContentTypeJSON)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}

	return c.parseResponse(resp)
}

// post makes a POST request to the specified URL with the given parameters.
// It returns the response and any error encountered.
// The caller is responsible for closing the response body.
func (c *Client) post(url string, params interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal POST params: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", ContentTypeJSON)
	req.Header.Set("Accept", ContentTypeJSON)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make POST request: %w", err)
	}

	return c.parseResponse(resp)
}

// parseResponse parses the response body into the given response structure.
func (c *Client) parseResponse(resp *http.Response) (json.RawMessage, error) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data, nil
}

// Quote returns a quote for the given parameters.
func (c *Client) Quote(params QuoteParams) ([]QuoteResponse, error) {
	resp, err := c.post(c.apiURL+c.endpointQuote, params)
	if err != nil {
		return nil, fmt.Errorf("failed to make quote request: %w", err)
	}

	var quotes []QuoteResponse
	if err := json.Unmarshal(resp, &quotes); err != nil {
		return nil, fmt.Errorf("failed to parse quote response: %w", err)
	}

	if len(quotes) == 0 {
		return nil, fmt.Errorf("no quotes returned")
	}

	return quotes, nil
}
