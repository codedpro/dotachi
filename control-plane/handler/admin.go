package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dotachi/control-plane/db"
	"github.com/dotachi/control-plane/model"
	"github.com/dotachi/control-plane/service"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		NodeID      int64  `json:"node_id"`
		MaxPlayers  int    `json:"max_players"`
		IsPrivate   bool   `json:"is_private"`
		Password    string `json:"password"`
		GameTag     string `json:"game_tag"`
		Description string `json:"description"`
		ExpiresAt   string `json:"expires_at"` // optional RFC3339
		IsShared    bool   `json:"is_shared"`
		HourlyCost  int    `json:"hourly_cost"` // for shared rooms
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.MaxPlayers < 2 || req.MaxPlayers > 200 {
		writeError(w, http.StatusBadRequest, "max_players must be between 2 and 200")
		return
	}

	// Validate game tag
	if req.GameTag == "" {
		req.GameTag = "other"
	}
	validTags := map[string]bool{
		"dota2": true, "cs2": true, "warcraft3": true,
		"aoe2": true, "valorant": true, "minecraft": true, "other": true,
	}
	if !validTags[req.GameTag] {
		writeError(w, http.StatusBadRequest, "invalid game_tag")
		return
	}

	// Verify node exists
	node, err := getNodeByID(req.NodeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "node not found")
		return
	}

	// Generate hub name and subnet based on next ID
	var maxID int64
	db.DB.QueryRow("SELECT COALESCE(MAX(id), 0) FROM rooms").Scan(&maxID)
	nextNum := maxID + 1
	hubName := fmt.Sprintf("hub_%d", nextNum)
	subnet := fmt.Sprintf("10.10.%d.0/24", nextNum)

	// Hash password if private
	var passwordHash *string
	if req.IsPrivate && req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		h := string(hash)
		passwordHash = &h
	}

	// Create hub on node
	if err := service.CreateHub(node.Host, node.APIPort, node.APISecret, hubName, req.MaxPlayers, subnet); err != nil {
		writeError(w, http.StatusBadGateway, "failed to create hub on node: "+err.Error())
		return
	}

	// Build optional expires_at
	var expiresAt *string
	if req.ExpiresAt != "" {
		expiresAt = &req.ExpiresAt
	}

	// Insert room record
	var roomID int64
	err = db.DB.QueryRow(
		`INSERT INTO rooms (node_id, name, hub_name, is_private, password_hash, max_players, subnet, game_tag, description, expires_at, is_shared, hourly_cost)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`,
		req.NodeID, req.Name, hubName, req.IsPrivate, passwordHash, req.MaxPlayers, subnet,
		req.GameTag, req.Description, expiresAt, req.IsShared, req.HourlyCost,
	).Scan(&roomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create room: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          roomID,
		"name":        req.Name,
		"hub_name":    hubName,
		"node_id":     req.NodeID,
		"subnet":      subnet,
		"game_tag":    req.GameTag,
		"description": req.Description,
		"is_shared":   req.IsShared,
		"hourly_cost": req.HourlyCost,
		"expires_at":  expiresAt,
	})
}

func (h *AdminHandler) AssignOwner(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")

	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Verify user exists
	var exists int
	db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", req.UserID).Scan(&exists)
	if exists == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Verify room exists
	var roomExists int
	db.DB.QueryRow("SELECT COUNT(*) FROM rooms WHERE id = $1", roomID).Scan(&roomExists)
	if roomExists == 0 {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	// Parse roomID to int64 for setRoomOwner
	roomIDInt, err := strconv.ParseInt(roomID, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}

	// Set both rooms.owner_id and room_roles atomically
	if err := setRoomOwner(db.DB, roomIDInt, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to assign owner")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "owner assigned"})
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(
		`SELECT id, phone, display_name, is_admin, created_at,
			shard_balance, total_play_hours, total_sessions, device_fingerprint
		FROM users ORDER BY id ASC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	type userWithFP struct {
		model.User
		DeviceFingerprint *string `json:"device_fingerprint"`
	}

	users := []userWithFP{}
	for rows.Next() {
		var u userWithFP
		var fp sql.NullString
		if err := rows.Scan(&u.ID, &u.Phone, &u.DisplayName, &u.IsAdmin, &u.CreatedAt,
			&u.ShardBalance, &u.TotalPlayHours, &u.TotalSessions, &fp); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		if fp.Valid {
			u.DeviceFingerprint = &fp.String
		}
		users = append(users, u)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}

// ResetDeviceFingerprint allows admin to clear a user's device fingerprint,
// enabling them to register a new account from the same device (e.g. after
// a legitimate hardware transfer or if the user sold their PC).
func (h *AdminHandler) ResetDeviceFingerprint(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "id")

	result, err := db.DB.Exec(
		"UPDATE users SET device_fingerprint = NULL WHERE id = $1", targetUserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "device fingerprint reset"})
}

func (h *AdminHandler) AddShards(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "id")

	var req struct {
		Amount      int    `json:"amount"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Lock user row and get current balance
	var balance int
	err = tx.QueryRow("SELECT shard_balance FROM users WHERE id = $1 FOR UPDATE", targetUserID).Scan(&balance)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	const maxShardBalance = 999_999_999 // ~1 billion shards
	newBalance := balance + req.Amount
	if newBalance > maxShardBalance {
		writeError(w, http.StatusBadRequest, "balance would exceed maximum")
		return
	}
	_, err = tx.Exec("UPDATE users SET shard_balance = $1 WHERE id = $2", newBalance, targetUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update balance")
		return
	}

	desc := req.Description
	if desc == "" {
		desc = "Admin top-up"
	}
	_, err = tx.Exec(
		`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description)
		VALUES ($1, $2, $3, 'admin_topup', $4)`,
		targetUserID, req.Amount, newBalance, desc,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record transaction")
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "shards added",
		"amount":        req.Amount,
		"new_balance":   newBalance,
	})
}

func (h *AdminHandler) RemoveShards(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "id")

	var req struct {
		Amount      int    `json:"amount"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	var balance int
	err = tx.QueryRow("SELECT shard_balance FROM users WHERE id = $1 FOR UPDATE", targetUserID).Scan(&balance)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	newBalance := balance - req.Amount
	if newBalance < 0 {
		newBalance = 0
	}

	_, err = tx.Exec("UPDATE users SET shard_balance = $1 WHERE id = $2", newBalance, targetUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update balance")
		return
	}

	desc := req.Description
	if desc == "" {
		desc = "Admin removal"
	}
	actualRemoved := balance - newBalance
	_, err = tx.Exec(
		`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description)
		VALUES ($1, $2, $3, 'refund', $4)`,
		targetUserID, -actualRemoved, newBalance, desc,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record transaction")
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "shards removed",
		"amount":        actualRemoved,
		"new_balance":   newBalance,
	})
}

// DeleteUser permanently deletes a user and all associated data within a transaction.
// This manually cleans up all foreign key references to avoid constraint violations
// since the schema does not use ON DELETE CASCADE.
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")

	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Verify user exists
	var exists int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", targetID).Scan(&exists)
	if err != nil || exists == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Clean up all references
	tx.Exec("DELETE FROM room_members WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM room_bans WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM room_bans WHERE banned_by = $1", targetID)
	tx.Exec("DELETE FROM favorites WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM play_sessions WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM shared_sessions WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM room_roles WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM shard_transactions WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM promo_redemptions WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM room_messages WHERE user_id = $1", targetID)
	tx.Exec("DELETE FROM room_invites WHERE created_by = $1", targetID)
	// Clear owner_id on rooms owned by this user
	tx.Exec("UPDATE rooms SET owner_id = NULL WHERE owner_id = $1", targetID)
	// Delete user
	_, err = tx.Exec("DELETE FROM users WHERE id = $1", targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
