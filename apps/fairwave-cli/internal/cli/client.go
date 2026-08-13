package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// get performs a GET returning decoded JSON or a typed error.
func (c *client) get(path string, out interface{}) error {
	return c.do(http.MethodGet, path, nil, out)
}

// post performs a POST with JSON body.
func (c *client) post(path string, in, out interface{}) error {
	return c.do(http.MethodPost, path, in, out)
}

// raw performs a request and returns the response bytes (for binary
// payloads: backups, CSV exports, QR PNGs). extra headers are added to
// the request.
func (c *client) raw(method, path string, body []byte, extra map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("control plane unreachable at %s: %w", c.base, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var eb api.ErrorBody
		_ = json.Unmarshal(data, &eb)
		if eb.Error.Code != "" {
			return nil, fmt.Errorf("%s: %s", eb.Error.Code, eb.Error.Message)
		}
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return data, nil
}

func (c *client) do(method, path string, in, out interface{}) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("control plane unreachable at %s: %w", c.base, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var eb api.ErrorBody
		_ = json.NewDecoder(resp.Body).Decode(&eb)
		if eb.Error.Code != "" {
			return fmt.Errorf("%s: %s", eb.Error.Code, eb.Error.Message)
		}
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
