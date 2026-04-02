package handler

import (
	"net/http"

	"github.com/dotachi/control-plane/db"
	"github.com/go-chi/chi/v5"
)

type MonitorHandler struct{}

func NewMonitorHandler() *MonitorHandler {
	return &MonitorHandler{}
}

func (h *MonitorHandler) Overview(w http.ResponseWriter, r *http.Request) {
	var totalNodes, activeNodes int
	if err := db.DB.QueryRow("SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active = TRUE) FROM nodes").Scan(&totalNodes, &activeNodes); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	var totalRooms, activeRooms int
	if err := db.DB.QueryRow("SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active = TRUE) FROM rooms").Scan(&totalRooms, &activeRooms); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	var totalPlayers int
	if err := db.DB.QueryRow(`
		SELECT COUNT(*) FROM room_members rm
		JOIN rooms r ON r.id = rm.room_id
		WHERE r.is_active = TRUE`).Scan(&totalPlayers); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	var totalBandwidthIn, totalBandwidthOut int64
	// Bandwidth is not tracked in the DB; report 0 for now.
	// In the future this could aggregate from node-agent /stats calls.

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_nodes":        totalNodes,
		"active_nodes":       activeNodes,
		"total_rooms":        totalRooms,
		"active_rooms":       activeRooms,
		"total_players":      totalPlayers,
		"total_bandwidth_in":  totalBandwidthIn,
		"total_bandwidth_out": totalBandwidthOut,
	})
}

func (h *MonitorHandler) Nodes(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`
		SELECT n.id, n.name, n.host, n.is_active
		FROM nodes n
		ORDER BY n.id ASC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	type monitorRoom struct {
		RoomID  int64  `json:"room_id"`
		Name    string `json:"name"`
		Players int    `json:"players"`
		Max     int    `json:"max"`
	}

	type monitorNode struct {
		NodeID      int64          `json:"node_id"`
		NodeName    string         `json:"node_name"`
		Host        string         `json:"host"`
		IsHealthy   bool           `json:"is_healthy"`
		RoomCount   int            `json:"room_count"`
		PlayerCount int            `json:"player_count"`
		Rooms       []monitorRoom  `json:"rooms"`
	}

	var nodes []monitorNode
	for rows.Next() {
		var n monitorNode
		if err := rows.Scan(&n.NodeID, &n.NodeName, &n.Host, &n.IsHealthy); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		nodes = append(nodes, n)
	}

	for i := range nodes {
		roomRows, err := db.DB.Query(`
			SELECT r.id, r.name,
				(SELECT COUNT(*) FROM room_members WHERE room_id = r.id) as current_players,
				r.max_players
			FROM rooms r
			WHERE r.node_id = $1 AND r.is_active = TRUE
			ORDER BY r.id ASC`, nodes[i].NodeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}

		rooms := []monitorRoom{}
		totalPlayers := 0
		for roomRows.Next() {
			var rm monitorRoom
			if err := roomRows.Scan(&rm.RoomID, &rm.Name, &rm.Players, &rm.Max); err != nil {
				roomRows.Close()
				writeError(w, http.StatusInternalServerError, "scan error")
				return
			}
			totalPlayers += rm.Players
			rooms = append(rooms, rm)
		}
		roomRows.Close()

		nodes[i].Rooms = rooms
		nodes[i].RoomCount = len(rooms)
		nodes[i].PlayerCount = totalPlayers
	}

	if nodes == nil {
		nodes = []monitorNode{}
	}

	writeJSON(w, http.StatusOK, nodes)
}

func (h *MonitorHandler) Room(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")

	type memberDetail struct {
		UserID      int64  `json:"user_id"`
		DisplayName string `json:"display_name"`
		JoinedAt    string `json:"joined_at"`
	}

	var id int64
	var name, hubName, nodeName, createdAt, lastActivity string
	var ownerName *string
	var currentPlayers, maxPlayers int

	err := db.DB.QueryRow(`
		SELECT r.id, r.name, r.hub_name, n.name,
			(SELECT u.display_name FROM users u WHERE u.id = r.owner_id),
			(SELECT COUNT(*) FROM room_members WHERE room_id = r.id) as current_players,
			r.max_players, r.created_at, r.last_activity
		FROM rooms r
		JOIN nodes n ON n.id = r.node_id
		WHERE r.id = $1`, roomID,
	).Scan(&id, &name, &hubName, &nodeName, &ownerName, &currentPlayers, &maxPlayers, &createdAt, &lastActivity)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	ownerDisplay := ""
	if ownerName != nil {
		ownerDisplay = *ownerName
	}

	memberRows, err := db.DB.Query(`
		SELECT rm.user_id, u.display_name, rm.joined_at
		FROM room_members rm
		JOIN users u ON u.id = rm.user_id
		WHERE rm.room_id = $1
		ORDER BY rm.joined_at ASC`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer memberRows.Close()

	members := []memberDetail{}
	for memberRows.Next() {
		var m memberDetail
		if err := memberRows.Scan(&m.UserID, &m.DisplayName, &m.JoinedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		members = append(members, m)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"room_id":         id,
		"name":            name,
		"hub_name":        hubName,
		"node":            nodeName,
		"owner":           ownerDisplay,
		"players":         members,
		"current_players": currentPlayers,
		"max_players":     maxPlayers,
		"created_at":      createdAt,
		"last_activity":   lastActivity,
	})
}
