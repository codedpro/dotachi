package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	defaultAPIBase = "http://localhost:8080"
	configDir      = ".dotachi"
	configFile     = "config"
	tokenFile      = "token"
)

// Client handles HTTP communication with the control-plane API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a Client by reading config and token from disk / env.
func NewClient() *Client {
	c := &Client{
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}

	// API base URL: env > config file > default
	if v := os.Getenv("DOTACHI_API"); v != "" {
		c.BaseURL = strings.TrimRight(v, "/")
	} else if url, err := readDotachiFile(configFile); err == nil && url != "" {
		c.BaseURL = strings.TrimRight(url, "/")
	} else {
		c.BaseURL = defaultAPIBase
	}

	// JWT token
	if tok, err := readDotachiFile(tokenFile); err == nil {
		c.Token = strings.TrimSpace(tok)
	}

	return c
}

// ----- helpers for ~/.dotachi/ files -----

func dotachiDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configDir)
}

func readDotachiFile(name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dotachiDir(), name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeDotachiFile(name, content string) error {
	dir := dotachiDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content+"\n"), 0600)
}

// SaveToken persists a JWT token to ~/.dotachi/token.
func SaveToken(token string) error {
	return writeDotachiFile(tokenFile, token)
}

// SaveConfig persists the API base URL to ~/.dotachi/config.
func SaveConfig(url string) error {
	return writeDotachiFile(configFile, url)
}

// ----- HTTP verbs -----

func (c *Client) do(method, path string, body interface{}) (map[string]interface{}, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			// If response is not JSON, wrap it.
			return map[string]interface{}{"raw": string(respBody)}, nil
		}
	}
	return result, nil
}

// doList is like do but expects the top-level response to be a JSON array.
func (c *Client) doList(method, path string, body interface{}) ([]map[string]interface{}, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Try to parse as array first.
	var list []map[string]interface{}
	if err := json.Unmarshal(respBody, &list); err == nil {
		return list, nil
	}

	// Fallback: might be wrapped in {"data": [...]}.
	var wrapper map[string]interface{}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Try common wrapper keys.
	for _, key := range []string{"data", "items", "results", "nodes", "rooms", "users"} {
		if raw, ok := wrapper[key]; ok {
			if items, ok := raw.([]interface{}); ok {
				result := make([]map[string]interface{}, 0, len(items))
				for _, item := range items {
					if m, ok := item.(map[string]interface{}); ok {
						result = append(result, m)
					}
				}
				return result, nil
			}
		}
	}

	return nil, fmt.Errorf("unexpected response format")
}

func (c *Client) Get(path string) (map[string]interface{}, error) {
	return c.do(http.MethodGet, path, nil)
}

func (c *Client) GetList(path string) ([]map[string]interface{}, error) {
	return c.doList(http.MethodGet, path, nil)
}

func (c *Client) Post(path string, body interface{}) (map[string]interface{}, error) {
	return c.do(http.MethodPost, path, body)
}

func (c *Client) Delete(path string) (map[string]interface{}, error) {
	return c.do(http.MethodDelete, path, nil)
}

// ----- table printing helpers -----

// PrintTable renders rows as a tab-aligned table to stdout.
// columns defines the header names and the corresponding map keys.
func PrintTable(columns []Column, rows []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	// Header
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	// Rows
	for _, row := range rows {
		vals := make([]string, len(columns))
		for i, col := range columns {
			vals[i] = fmtVal(row[col.Key])
		}
		fmt.Fprintln(w, strings.Join(vals, "\t"))
	}

	w.Flush()
}

// Column maps a table header to a JSON key.
type Column struct {
	Header string
	Key    string
}

// PrintObject prints a single key-value map vertically.
func PrintObject(obj map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for k, v := range obj {
		fmt.Fprintf(w, "%s:\t%s\n", k, fmtVal(v))
	}
	w.Flush()
}

func fmtVal(v interface{}) string {
	if v == nil {
		return "-"
	}
	switch val := v.(type) {
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "yes"
		}
		return "no"
	default:
		return fmt.Sprintf("%v", val)
	}
}
