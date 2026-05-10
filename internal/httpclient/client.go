// Package httpclient provides an HTTP client abstraction used by all
// music platform providers. It supports dependency injection for testing
// via replaceable *http.Client.
package httpclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client wraps an *http.Client with convenience methods for common
// request patterns (GET, POST form, POST JSON).
type Client struct {
	hc     *http.Client
	common map[string]string // common headers added to every request
}

// New creates a Client backed by the given *http.Client.
// If hc is nil, http.DefaultClient is used.
func New(hc *http.Client) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{hc: hc, common: make(map[string]string)}
}

// SetCommonHeader sets a header that will be included with every request.
func (c *Client) SetCommonHeader(key, value string) {
	c.common[key] = value
}

// Do performs an HTTP request with common headers applied.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	for k, v := range c.common {
		req.Header.Set(k, v)
	}
	return c.hc.Do(req)
}

// Get performs a GET request and returns the response body as a string.
func (c *Client) Get(rawURL string, params map[string]string, headers map[string]string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// PostForm performs a POST request with URL-encoded form data.
func (c *Client) PostForm(rawURL string, params map[string]string, headers map[string]string) (string, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// PostJSON performs a POST request with a JSON body.
func (c *Client) PostJSON(rawURL string, body any, headers map[string]string) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
}

// RawPost performs a POST request with a raw byte body.
func (c *Client) RawPost(rawURL string, body []byte, headers map[string]string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
}

// GetBytes performs a GET request and returns the raw bytes.
func (c *Client) GetBytes(rawURL string, params map[string]string, headers map[string]string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
