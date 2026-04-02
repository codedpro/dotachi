package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func nodeURL(host string, port int, path string) string {
	return fmt.Sprintf("http://%s:%d%s", host, port, path)
}

func doNodeRequest(host string, port int, secret string, path string, body interface{}) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode body: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPost, nodeURL(host, port, path), &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Secret", secret)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("node request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return fmt.Errorf("node error: %s", errResp.Error)
		}
		return fmt.Errorf("node returned status %d", resp.StatusCode)
	}
	return nil
}

func CreateHub(host string, port int, secret string, hubName string, maxSessions int, subnet string) error {
	return doNodeRequest(host, port, secret, "/hub/create", map[string]interface{}{
		"hub_name":     hubName,
		"max_sessions": maxSessions,
		"subnet":       subnet,
	})
}

func DeleteHub(host string, port int, secret string, hubName string) error {
	return doNodeRequest(host, port, secret, "/hub/delete", map[string]interface{}{
		"hub_name": hubName,
	})
}

func CreateVPNUser(host string, port int, secret string, hubName string, username string, password string) error {
	return doNodeRequest(host, port, secret, "/user/create", map[string]interface{}{
		"hub_name": hubName,
		"username": username,
		"password": password,
	})
}

func DeleteVPNUser(host string, port int, secret string, hubName string, username string) error {
	return doNodeRequest(host, port, secret, "/user/delete", map[string]interface{}{
		"hub_name": hubName,
		"username": username,
	})
}

func DisconnectUser(host string, port int, secret string, hubName string, username string) error {
	return doNodeRequest(host, port, secret, "/user/disconnect", map[string]interface{}{
		"hub_name": hubName,
		"username": username,
	})
}

func PingNode(host string, port int, secret string) error {
	req, err := http.NewRequest(http.MethodGet, nodeURL(host, port, "/health"), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Api-Secret", secret)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("node request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("node returned status %d", resp.StatusCode)
	}
	return nil
}
