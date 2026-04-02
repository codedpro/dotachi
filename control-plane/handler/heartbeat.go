package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dotachi/control-plane/db"
)

type HeartbeatHandler struct{}

func NewHeartbeatHandler() *HeartbeatHandler {
	return &HeartbeatHandler{}
}

// heartbeatRequest is the payload sent by node agents every 30 seconds.
type heartbeatRequest struct {
	NodeName      string   `json:"node_name"`
	Host          string   `json:"host"`
	APIPort       int      `json:"api_port"`
	HubCount      int      `json:"hub_count"`
	TotalSessions int      `json:"total_sessions"`
	CPUUsage      float64  `json:"cpu_usage"`
	MemoryMB      int64    `json:"memory_mb"`
	ActiveHubs    []string `json:"active_hubs"`
}

type expectedHubResp struct {
	HubName     string `json:"hub_name"`
	MaxSessions int    `json:"max_sessions"`
	Subnet      string `json:"subnet"`
}

// Heartbeat handles POST /internal/heartbeat.
// Authenticated by X-Api-Secret header matched against the node's stored secret.
func (h *HeartbeatHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	secret := r.Header.Get("X-Api-Secret")
	if secret == "" {
		writeError(w, http.StatusUnauthorized, "missing X-Api-Secret header")
		return
	}

	var req heartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NodeName == "" {
		writeError(w, http.StatusBadRequest, "node_name is required")
		return
	}

	// Look up node by name and verify secret
	var nodeID int64
	var storedSecret string
	err := db.DB.QueryRow(
		"SELECT id, api_secret FROM nodes WHERE name = $1",
		req.NodeName,
	).Scan(&nodeID, &storedSecret)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unknown node")
		return
	}

	if secret != storedSecret {
		writeError(w, http.StatusUnauthorized, "invalid api secret for node")
		return
	}

	// Update node heartbeat data
	_, err = db.DB.Exec(`
		UPDATE nodes SET
			last_heartbeat = CURRENT_TIMESTAMP,
			hub_count = $1,
			session_count = $2,
			cpu_usage = $3,
			memory_mb = $4,
			is_active = TRUE
		WHERE id = $5`,
		req.HubCount, req.TotalSessions, req.CPUUsage, req.MemoryMB, nodeID,
	)
	if err != nil {
		log.Printf("[heartbeat] failed to update node %d: %v", nodeID, err)
		writeError(w, http.StatusInternalServerError, "failed to update node status")
		return
	}

	// Query expected hubs: active rooms assigned to this node
	rows, err := db.DB.Query(`
		SELECT hub_name, max_players, subnet
		FROM rooms
		WHERE node_id = $1 AND is_active = TRUE`,
		nodeID,
	)
	if err != nil {
		log.Printf("[heartbeat] failed to query rooms for node %d: %v", nodeID, err)
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	expectedHubs := []expectedHubResp{}
	for rows.Next() {
		var eh expectedHubResp
		if err := rows.Scan(&eh.HubName, &eh.MaxSessions, &eh.Subnet); err != nil {
			log.Printf("[heartbeat] failed to scan room row: %v", err)
			continue
		}
		expectedHubs = append(expectedHubs, eh)
	}

	log.Printf("[heartbeat] node=%s id=%d hubs=%d sessions=%d cpu=%.1f%% mem=%dMB",
		req.NodeName, nodeID, req.HubCount, req.TotalSessions, req.CPUUsage, req.MemoryMB)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":            true,
		"expected_hubs": expectedHubs,
	})
}
