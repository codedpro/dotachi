package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dotachi/node-agent/handler"
	"github.com/dotachi/node-agent/softether"
)

// heartbeatPayload is sent to the control plane every 30 seconds.
type heartbeatPayload struct {
	NodeName      string   `json:"node_name"`
	Host          string   `json:"host"`
	APIPort       int      `json:"api_port"`
	HubCount      int      `json:"hub_count"`
	TotalSessions int      `json:"total_sessions"`
	CPUUsage      float64  `json:"cpu_usage"`
	MemoryMB      int64    `json:"memory_mb"`
	ActiveHubs    []string `json:"active_hubs"`
}

// heartbeatResponse is received from the control plane.
type heartbeatResponse struct {
	OK           bool          `json:"ok"`
	ExpectedHubs []expectedHub `json:"expected_hubs"`
}

type expectedHub struct {
	HubName     string `json:"hub_name"`
	MaxSessions int    `json:"max_sessions"`
	Subnet      string `json:"subnet"`
}

var heartbeatClient = &http.Client{Timeout: 15 * time.Second}

// StartHeartbeat sends periodic status reports to the control plane.
// If the control plane is unreachable, it keeps trying -- the node
// operates independently regardless.
func StartHeartbeat(controlPlaneURL, apiSecret, nodeName, nodeHost string, nodePort int, se *softether.Client) {
	go func() {
		log.Printf("[heartbeat] started -- reporting to %s every 30s as %q", controlPlaneURL, nodeName)
		for {
			sendHeartbeat(controlPlaneURL, apiSecret, nodeName, nodeHost, nodePort, se)
			time.Sleep(30 * time.Second)
		}
	}()
}

func sendHeartbeat(controlPlaneURL, apiSecret, nodeName, nodeHost string, nodePort int, se *softether.Client) {
	// Gather node stats
	hubCount := 0
	totalSessions := 0

	out, err := se.ServerCmd("ServerStatusGet")
	if err == nil {
		hubCount = handler.ParseServerStatusInt(out, "Number of Virtual Hubs")
		totalSessions = handler.ParseServerStatusInt(out, "Number of Sessions")
		if totalSessions == 0 {
			totalSessions = handler.ParseServerStatusInt(out, "Num Sessions")
		}
	} else {
		log.Printf("[heartbeat] ServerStatusGet failed: %v", err)
	}

	cpuUsage := handler.ReadCPUUsage()
	memoryMB := handler.ReadMemoryUsageMB()

	// Get list of active hubs
	activeHubs := listHubs(se)

	payload := heartbeatPayload{
		NodeName:      nodeName,
		Host:          nodeHost,
		APIPort:       nodePort,
		HubCount:      hubCount,
		TotalSessions: totalSessions,
		CPUUsage:      cpuUsage,
		MemoryMB:      memoryMB,
		ActiveHubs:    activeHubs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[heartbeat] failed to marshal payload: %v", err)
		return
	}

	url := strings.TrimRight(controlPlaneURL, "/") + "/internal/heartbeat"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[heartbeat] failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Secret", apiSecret)

	resp, err := heartbeatClient.Do(req)
	if err != nil {
		log.Printf("[heartbeat] control plane unreachable: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[heartbeat] control plane returned status %d", resp.StatusCode)
		return
	}

	var hbResp heartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		log.Printf("[heartbeat] failed to decode response: %v", err)
		return
	}

	if !hbResp.OK {
		log.Printf("[heartbeat] control plane returned ok=false")
		return
	}

	log.Printf("[heartbeat] ok -- hubs=%d sessions=%d cpu=%.1f%% mem=%dMB",
		hubCount, totalSessions, cpuUsage, memoryMB)

	// Reconcile hubs
	reconcileHubs(se, activeHubs, hbResp.ExpectedHubs)
}

// listHubs runs vpncmd HubList and returns the hub names (excluding DEFAULT).
func listHubs(se *softether.Client) []string {
	out, err := se.ServerCmd("HubList")
	if err != nil {
		log.Printf("[heartbeat] HubList failed: %v", err)
		return nil
	}
	return parseHubList(out)
}

// parseHubList extracts hub names from vpncmd HubList output.
// The output format has lines like:
//
//	Virtual Hub Name |Status|...
//	-----------------+------+---
//	hub_1            |Online|...
func parseHubList(output string) []string {
	var hubs []string
	lines := strings.Split(output, "\n")

	// Find the data lines after the header separator (----)
	dataStarted := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Detect the separator line that comes after column headers
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") {
			dataStarted = true
			continue
		}

		if !dataStarted {
			continue
		}

		// Each data line has fields separated by |
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 1 {
			continue
		}

		hubName := strings.TrimSpace(parts[0])
		if hubName == "" || strings.EqualFold(hubName, "DEFAULT") {
			continue
		}

		// Skip if it looks like a header or footer
		if strings.Contains(hubName, "Virtual Hub Name") {
			continue
		}
		// Stop if we hit the command result line
		if strings.HasPrefix(hubName, "The command completed") {
			break
		}

		hubs = append(hubs, hubName)
	}
	return hubs
}

// reconcileHubs compares local hubs with expected hubs from control plane
// and performs recovery or cleanup as needed.
func reconcileHubs(se *softether.Client, localHubs []string, expectedHubs []expectedHub) {
	if expectedHubs == nil {
		return
	}

	// Build sets for comparison
	localSet := make(map[string]bool, len(localHubs))
	for _, h := range localHubs {
		localSet[h] = true
	}

	expectedSet := make(map[string]expectedHub, len(expectedHubs))
	for _, eh := range expectedHubs {
		expectedSet[eh.HubName] = eh
	}

	// Recovery: expected but not local -- recreate
	for _, eh := range expectedHubs {
		if !localSet[eh.HubName] {
			log.Printf("[heartbeat/recovery] recreating missing hub %s (max=%d, subnet=%s)",
				eh.HubName, eh.MaxSessions, eh.Subnet)
			recreateHub(se, eh)
		}
	}

	// Cleanup: local but not expected -- delete orphans.
	// Only delete hubs that look like Dotachi-managed hubs (hub_*).
	for _, hubName := range localHubs {
		if _, expected := expectedSet[hubName]; !expected {
			if !strings.HasPrefix(hubName, "hub_") {
				continue
			}
			log.Printf("[heartbeat/cleanup] deleting orphan hub %s", hubName)
			if _, err := se.ServerCmd("HubDelete", hubName); err != nil {
				log.Printf("[heartbeat/cleanup] failed to delete hub %s: %v", hubName, err)
			}
		}
	}
}

// recreateHub creates a hub with the same pipeline as handler.HubCreate.
func recreateHub(se *softether.Client, hub expectedHub) {
	maxSess := hub.MaxSessions
	if maxSess <= 0 {
		maxSess = 100
	}

	start, end, mask, gw, err := handler.SubnetParams(hub.Subnet)
	if err != nil {
		log.Printf("[heartbeat/recovery] invalid subnet %s for hub %s: %v", hub.Subnet, hub.HubName, err)
		return
	}

	// Step 1: Create the hub
	if _, err := se.ServerCmd("HubCreate", hub.HubName, `/PASSWORD:""`); err != nil {
		log.Printf("[heartbeat/recovery] HubCreate failed for %s: %v", hub.HubName, err)
		return
	}

	// Step 2: Set max sessions
	if _, err := se.HubCmd(hub.HubName, "SetMaxSession", strconv.Itoa(maxSess)); err != nil {
		log.Printf("[heartbeat/recovery] SetMaxSession failed for %s: %v", hub.HubName, err)
		return
	}

	// Step 3: Enable SecureNAT
	if _, err := se.HubCmd(hub.HubName, "SecureNatEnable"); err != nil {
		log.Printf("[heartbeat/recovery] SecureNatEnable failed for %s: %v", hub.HubName, err)
		return
	}

	// Step 4: DHCP config -- optimized for LAN gaming
	dhcpArgs := fmt.Sprintf(
		"/START:%s /END:%s /MASK:%s /EXPIRE:86400 /GW:%s /DNS:none /DNS2:none",
		start, end, mask, gw,
	)
	if _, err := se.HubCmd(hub.HubName, "DhcpSet", dhcpArgs); err != nil {
		log.Printf("[heartbeat/recovery] DhcpSet failed for %s: %v", hub.HubName, err)
		return
	}

	// Step 5: NAT settings -- tuned for game stability
	if _, err := se.HubCmd(hub.HubName, "NatSet",
		"/MTU:1400", "/TCPTIMEOUT:86400", "/UDPTIMEOUT:3600"); err != nil {
		log.Printf("[heartbeat/recovery] NatSet failed for %s: %v", hub.HubName, err)
		return
	}

	// Step 6: Hub options for broadcast optimization
	se.HubCmd(hub.HubName, "SetHubOption",
		"/NoArpPolling:1", "/NoIPv6DefaultRouterInRA:1", "/NoMacAddressLog:1")

	log.Printf("[heartbeat/recovery] successfully recreated hub %s", hub.HubName)
}
