package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dotachi/control-plane/db"
	"github.com/dotachi/control-plane/model"
	"github.com/dotachi/control-plane/service"
	"github.com/go-chi/chi/v5"
)

type NodeHandler struct{}

func NewNodeHandler() *NodeHandler {
	return &NodeHandler{}
}

func (h *NodeHandler) AddNode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Host      string `json:"host"`
		APIPort   int    `json:"api_port"`
		APISecret string `json:"api_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Host == "" || req.APISecret == "" {
		writeError(w, http.StatusBadRequest, "name, host, and api_secret are required")
		return
	}
	if req.APIPort == 0 {
		req.APIPort = 7443
	}

	var id int64
	err := db.DB.QueryRow(
		"INSERT INTO nodes (name, host, api_port, api_secret) VALUES ($1, $2, $3, $4) RETURNING id",
		req.Name, req.Host, req.APIPort, req.APISecret,
	).Scan(&id)
	if err != nil {
		writeError(w, http.StatusConflict, "node name already exists")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":   id,
		"name": req.Name,
		"host": req.Host,
	})
}

func (h *NodeHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`
		SELECT n.id, n.name, n.host, n.api_port, n.is_active, n.max_rooms, n.created_at,
			(SELECT COUNT(*) FROM rooms WHERE node_id = n.id AND is_active = TRUE) as room_count,
			n.last_heartbeat, n.hub_count, n.session_count, n.cpu_usage, n.memory_mb
		FROM nodes n
		ORDER BY n.id ASC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	nodes := []model.Node{}
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.Name, &n.Host, &n.APIPort, &n.IsActive, &n.MaxRooms, &n.CreatedAt, &n.RoomCount,
			&n.LastHeartbeat, &n.HubCount, &n.SessionCount, &n.CPUUsage, &n.MemoryMB); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		nodes = append(nodes, n)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": nodes})
}

func (h *NodeHandler) PingNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var node model.Node
	err := db.DB.QueryRow(
		"SELECT id, name, host, api_port, api_secret FROM nodes WHERE id = $1", id,
	).Scan(&node.ID, &node.Name, &node.Host, &node.APIPort, &node.APISecret)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	if err := service.PingNode(node.Host, node.APIPort, node.APISecret); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"node_id": node.ID,
			"name":    node.Name,
			"status":  "unreachable",
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node_id": node.ID,
		"name":    node.Name,
		"status":  "ok",
	})
}
