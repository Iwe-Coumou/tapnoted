package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// serverError means the Worker actually responded, just with a 4xx/5xx —
// a real answer, not a network problem, so request() won't retry it.
type serverError struct {
	status string
	body   string
}

func (e *serverError) Error() string {
	return fmt.Sprintf("server returned %s: %s", e.status, e.body)
}

// request sends an authenticated request to the tapnote Worker and returns
// the raw response body. body is JSON-marshaled if non-nil. A network-level
// failure (timeout, connection refused, DNS, ...) is retried once in case
// it was a one-off blip; an actual error response from the server is not.
func request(method, path string, body any) ([]byte, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	respBody, err := doRequest(cfg, method, path, bodyBytes)
	if err == nil {
		return respBody, nil
	}

	var srvErr *serverError
	if errors.As(err, &srvErr) {
		return nil, err
	}

	return doRequest(cfg, method, path, bodyBytes)
}

func doRequest(cfg config, method, path string, bodyBytes []byte) ([]byte, error) {
	var reqBody io.Reader
	if bodyBytes != nil {
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, cfg.URL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Secret)
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, &serverError{status: resp.Status, body: string(respBody)}
	}

	return respBody, nil
}
