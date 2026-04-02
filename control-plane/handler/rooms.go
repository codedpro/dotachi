package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dotachi/control-plane/db"
	"github.com/dotachi/control-plane/middleware"
	"github.com/dotachi/control-plane/model"
	"github.com/dotachi/control-plane/service"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type RoomHandler struct{}

func NewRoomHandler() *RoomHandler {
	return &RoomHandler{}
}

// ---- constants ----

const (
	shardPerSlotDaily      = 1000
	monthlyDiscountPct     = 10
	quarterlyDiscountPct   = 25
	yearlyDiscountPct      = 40
	dataCapBytes           = 10 * 1024 * 1024 * 1024 // 10 GB
)

// ---- pricing helpers ----
// Duration tiers: weekly (7d, min), monthly (30d, 10% off), quarterly (90d, 25% off), yearly (365d, 40% off)

func calculatePrice(slots int, duration string, days int) (int, int, error) {
	if slots < 2 || slots > 200 {
		return 0, 0, fmt.Errorf("slots must be between 2 and 200")
	}
	var totalDays int
	var price int
	switch duration {
	case "weekly":
		totalDays = 7
		price = slots * shardPerSlotDaily * 7
	case "monthly":
		totalDays = 30
		price = int(math.Round(float64(slots*shardPerSlotDaily*30) * (1.0 - float64(monthlyDiscountPct)/100.0)))
	case "quarterly":
		totalDays = 90
		price = int(math.Round(float64(slots*shardPerSlotDaily*90) * (1.0 - float64(quarterlyDiscountPct)/100.0)))
	case "yearly":
		totalDays = 365
		price = int(math.Round(float64(slots*shardPerSlotDaily*365) * (1.0 - float64(yearlyDiscountPct)/100.0)))
	default:
		return 0, 0, fmt.Errorf("invalid duration — must be weekly, monthly, quarterly, or yearly")
	}
	return price, totalDays, nil
}

// ---- room permission check via room_roles ----

// requireRoomPermission checks whether userID has 'owner' or 'admin' role in room_roles,
// or is a global admin. Returns nil if allowed.
func requireRoomPermission(roomID int64, userID int64, isGlobalAdmin bool) error {
	if isGlobalAdmin {
		return nil
	}
	var role string
	err := db.DB.QueryRow(
		"SELECT role FROM room_roles WHERE room_id = $1 AND user_id = $2", roomID, userID,
	).Scan(&role)
	if err != nil {
		return fmt.Errorf("you do not have permission on this room")
	}
	if role != "owner" && role != "admin" {
		return fmt.Errorf("you do not have permission on this room")
	}
	return nil
}

func requireRoomOwner(roomID int64, userID int64, isGlobalAdmin bool) error {
	if isGlobalAdmin {
		return nil
	}
	var role string
	err := db.DB.QueryRow(
		"SELECT role FROM room_roles WHERE room_id = $1 AND user_id = $2", roomID, userID,
	).Scan(&role)
	if err != nil || role != "owner" {
		return fmt.Errorf("only the room owner can do this")
	}
	return nil
}

// ---- List rooms ----

func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	isPrivate := r.URL.Query().Get("is_private")
	hasSlots := r.URL.Query().Get("has_slots")
	nodeID := r.URL.Query().Get("node_id")
	gameTag := r.URL.Query().Get("game")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	paramIndex := 0
	nextParam := func() string {
		paramIndex++
		return fmt.Sprintf("$%d", paramIndex)
	}

	query := `
		SELECT r.id, r.name, r.hub_name, r.node_id, n.name, r.owner_id,
			COALESCE(u.display_name, ''), r.is_private, r.max_players,
			(SELECT COUNT(*) FROM room_members WHERE room_id = r.id) as current_players,
			r.subnet, r.is_active, r.game_tag, r.description,
			r.expires_at, r.is_shared, r.hourly_cost, r.created_at
		FROM rooms r
		JOIN nodes n ON n.id = r.node_id
		LEFT JOIN users u ON u.id = r.owner_id
		WHERE r.is_active = TRUE`

	var args []interface{}

	if q != "" {
		query += " AND r.name ILIKE " + nextParam()
		args = append(args, "%"+q+"%")
	}
	if isPrivate == "true" {
		query += " AND r.is_private = TRUE"
	} else if isPrivate == "false" {
		query += " AND r.is_private = FALSE"
	}
	if hasSlots == "true" {
		query += " AND (SELECT COUNT(*) FROM room_members WHERE room_id = r.id) < r.max_players"
	}
	if nodeID != "" {
		query += " AND r.node_id = " + nextParam()
		args = append(args, nodeID)
	}
	if gameTag != "" {
		query += " AND r.game_tag = " + nextParam()
		args = append(args, gameTag)
	}

	query += " ORDER BY r.id DESC LIMIT " + nextParam() + " OFFSET " + nextParam()
	args = append(args, perPage, offset)

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	rooms := []model.Room{}
	for rows.Next() {
		var room model.Room
		var ownerDisplayName string
		err := rows.Scan(
			&room.ID, &room.Name, &room.HubName, &room.NodeID, &room.NodeName,
			&room.OwnerID, &ownerDisplayName, &room.IsPrivate, &room.MaxPlayers,
			&room.CurrentPlayers, &room.Subnet, &room.IsActive, &room.GameTag,
			&room.Description, &room.ExpiresAt, &room.IsShared, &room.HourlyCost,
			&room.CreatedAt,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		room.OwnerDisplayName = ownerDisplayName
		rooms = append(rooms, room)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rooms":    rooms,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *RoomHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	room, err := getRoomByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	writeJSON(w, http.StatusOK, room)
}

// ---- Pricing (public) ----

func (h *RoomHandler) Pricing(w http.ResponseWriter, r *http.Request) {
	examples := []map[string]int{}
	for _, slots := range []int{15, 25, 50, 100} {
		weekly := slots * shardPerSlotDaily * 7
		monthly := int(math.Round(float64(slots*shardPerSlotDaily*30) * 0.9))
		quarterly := int(math.Round(float64(slots*shardPerSlotDaily*90) * 0.75))
		yearly := int(math.Round(float64(slots*shardPerSlotDaily*365) * 0.6))
		examples = append(examples, map[string]int{
			"slots":     slots,
			"weekly":    weekly,
			"monthly":   monthly,
			"quarterly": quarterly,
			"yearly":    yearly,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"per_slot_daily":      shardPerSlotDaily,
		"min_duration":        "weekly",
		"monthly_discount":    monthlyDiscountPct,
		"quarterly_discount":  quarterlyDiscountPct,
		"yearly_discount":     yearlyDiscountPct,
		"examples":            examples,
	})
}

// ---- Shop info (public) ----

func (h *RoomHandler) Shop(w http.ResponseWriter, r *http.Request) {
	examples := []map[string]int{}
	for _, slots := range []int{15, 25, 50, 100} {
		weekly := slots * shardPerSlotDaily * 7
		monthly := int(math.Round(float64(slots*shardPerSlotDaily*30) * 0.9))
		quarterly := int(math.Round(float64(slots*shardPerSlotDaily*90) * 0.75))
		yearly := int(math.Round(float64(slots*shardPerSlotDaily*365) * 0.6))
		examples = append(examples, map[string]int{
			"slots":     slots,
			"weekly":    weekly,
			"monthly":   monthly,
			"quarterly": quarterly,
			"yearly":    yearly,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":          "\u0647\u0631 \u0634\u0627\u0631\u062f \u0645\u0639\u0627\u062f\u0644 \u06f1 \u062a\u0648\u0645\u0627\u0646 \u0627\u0633\u062a. \u0628\u0631\u0627\u06cc \u062e\u0631\u06cc\u062f \u0634\u0627\u0631\u062f \u0628\u0627 \u0645\u0627 \u062a\u0645\u0627\u0633 \u0628\u06af\u06cc\u0631\u06cc\u062f.",
		"shards_per_toman": 1,
		"contact": []map[string]string{
			{"platform": "Telegram", "handle": "@coded_pro"},
			{"platform": "Bale", "handle": "@coded_pro"},
		},
		"pricing": map[string]interface{}{
			"per_slot_daily":      shardPerSlotDaily,
			"min_duration":        "weekly",
			"monthly_discount":    monthlyDiscountPct,
			"quarterly_discount":  quarterlyDiscountPct,
			"yearly_discount":     yearlyDiscountPct,
			"examples":            examples,
		},
	})
}

// ---- Purchase a room ----

func (h *RoomHandler) PurchaseRoom(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req struct {
		Name      string `json:"name"`
		GameTag   string `json:"game_tag"`
		Slots     int    `json:"slots"`
		Duration  string `json:"duration"` // daily, monthly, yearly
		Days      int    `json:"days"`     // for daily: 1-30
		IsPrivate bool   `json:"is_private"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Validate game tag
	if req.GameTag == "" {
		req.GameTag = "other"
	}
	if !validGameTags[req.GameTag] {
		writeError(w, http.StatusBadRequest, "invalid game_tag")
		return
	}

	// Calculate price
	price, totalDays, err := calculatePrice(req.Slots, req.Duration, req.Days)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Begin transaction
	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Check shard balance (lock user row)
	var balance int
	err = tx.QueryRow("SELECT shard_balance FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}
	if balance < price {
		writeError(w, http.StatusPaymentRequired, fmt.Sprintf("insufficient shards: need %d, have %d", price, balance))
		return
	}

	// Deduct shards
	newBalance := balance - price
	_, err = tx.Exec("UPDATE users SET shard_balance = $1 WHERE id = $2", newBalance, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to deduct shards")
		return
	}

	// Auto-select best node (least active rooms)
	var nodeID int64
	var nodeHost string
	var nodeAPIPort int
	var nodeAPISecret string
	err = tx.QueryRow(`
		SELECT n.id, n.host, n.api_port, n.api_secret
		FROM nodes n
		WHERE n.is_active = TRUE
		ORDER BY (SELECT COUNT(*) FROM rooms WHERE node_id = n.id AND is_active = TRUE) ASC
		LIMIT 1`,
	).Scan(&nodeID, &nodeHost, &nodeAPIPort, &nodeAPISecret)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "no active nodes available")
		return
	}

	// Generate hub name and subnet
	var maxID int64
	tx.QueryRow("SELECT COALESCE(MAX(id), 0) FROM rooms").Scan(&maxID)
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
	if err := service.CreateHub(nodeHost, nodeAPIPort, nodeAPISecret, hubName, req.Slots, subnet); err != nil {
		writeError(w, http.StatusBadGateway, "failed to create hub on node: "+err.Error())
		return
	}

	// Set expiry
	expiresAt := time.Now().Add(time.Duration(totalDays) * 24 * time.Hour)

	// Insert room record
	var roomID int64
	err = tx.QueryRow(
		`INSERT INTO rooms (node_id, owner_id, name, hub_name, is_private, password_hash, max_players, subnet, game_tag, description, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '', $10) RETURNING id`,
		nodeID, userID, req.Name, hubName, req.IsPrivate, passwordHash, req.Slots, subnet, req.GameTag, expiresAt,
	).Scan(&roomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create room: "+err.Error())
		return
	}

	// Set owner in both rooms.owner_id and room_roles
	if err := setRoomOwner(tx, roomID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to assign room owner role")
		return
	}

	// Record shard transaction
	_, err = tx.Exec(
		`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description, ref_id)
		VALUES ($1, $2, $3, 'room_purchase', $4, $5)`,
		userID, -price, newBalance,
		fmt.Sprintf("Purchase room '%s' (%d slots, %s)", req.Name, req.Slots, req.Duration),
		roomID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record transaction")
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":             roomID,
		"name":           req.Name,
		"hub_name":       hubName,
		"node_id":        nodeID,
		"subnet":         subnet,
		"game_tag":       req.GameTag,
		"max_players":    req.Slots,
		"expires_at":     expiresAt.Format(time.RFC3339),
		"price_charged":  price,
		"shard_balance":  newBalance,
	})
}

// ---- Extend a room ----

func (h *RoomHandler) ExtendRoom(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())

	roomIDInt, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}

	// Only owner can extend
	if err := requireRoomOwner(roomIDInt, userID, isAdmin); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var req struct {
		Duration string `json:"duration"` // daily, monthly, yearly
		Days     int    `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get room details
	room, err := getRoomByID(roomIDStr)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	if !room.IsActive {
		// Grace period: allow extending within 7 days of expiry
		if room.ExpiresAt != nil {
			expiry, parseErr := time.Parse(time.RFC3339, *room.ExpiresAt)
			if parseErr != nil || time.Since(expiry) >= 7*24*time.Hour {
				writeError(w, http.StatusBadRequest, "room expired more than 7 days ago")
				return
			}
			// Allow extension, will re-activate below
		} else {
			writeError(w, http.StatusBadRequest, "room is not active")
			return
		}
	}

	// Calculate price based on room slots
	price, totalDays, err := calculatePrice(room.MaxPlayers, req.Duration, req.Days)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Check shard balance
	var balance int
	err = tx.QueryRow("SELECT shard_balance FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}
	if balance < price {
		writeError(w, http.StatusPaymentRequired, fmt.Sprintf("insufficient shards: need %d, have %d", price, balance))
		return
	}

	// Deduct shards
	newBalance := balance - price
	_, err = tx.Exec("UPDATE users SET shard_balance = $1 WHERE id = $2", newBalance, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to deduct shards")
		return
	}

	// Extend expires_at: if already expired or null, start from now; otherwise add to existing
	var newExpiresAt time.Time
	var currentExpiry *time.Time
	err = tx.QueryRow("SELECT expires_at FROM rooms WHERE id = $1", room.ID).Scan(&currentExpiry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read room expiry")
		return
	}

	addDuration := time.Duration(totalDays) * 24 * time.Hour
	if currentExpiry == nil || currentExpiry.Before(time.Now()) {
		newExpiresAt = time.Now().Add(addDuration)
	} else {
		newExpiresAt = currentExpiry.Add(addDuration)
	}

	_, err = tx.Exec("UPDATE rooms SET expires_at = $1, is_active = TRUE WHERE id = $2", newExpiresAt, room.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to extend room")
		return
	}

	// Record transaction
	_, err = tx.Exec(
		`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description, ref_id)
		VALUES ($1, $2, $3, 'room_extend', $4, $5)`,
		userID, -price, newBalance,
		fmt.Sprintf("Extend room '%s' by %d days", room.Name, totalDays),
		room.ID,
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
		"message":       "room extended",
		"expires_at":    newExpiresAt.Format(time.RFC3339),
		"price_charged": price,
		"shard_balance": newBalance,
	})
}

// ---- Set role in a room ----

func (h *RoomHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())

	roomIDInt, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}

	if err := requireRoomOwner(roomIDInt, userID, isAdmin); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var req struct {
		UserID int64  `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role != "admin" && req.Role != "member" {
		writeError(w, http.StatusBadRequest, "role must be 'admin' or 'member' — use transfer endpoint for ownership")
		return
	}

	// Upsert room role
	_, err = db.DB.Exec(
		`INSERT INTO room_roles (room_id, user_id, role) VALUES ($1, $2, $3)
		ON CONFLICT (room_id, user_id) DO UPDATE SET role = $3`,
		roomIDInt, req.UserID, req.Role,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set role")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "role updated",
		"user_id": req.UserID,
		"role":    req.Role,
	})
}

// ---- Transfer room ownership ----

func (h *RoomHandler) TransferRoom(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())

	roomIDInt, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}

	if err := requireRoomOwner(roomIDInt, userID, isAdmin); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Verify target user exists
	var exists int
	db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", req.UserID).Scan(&exists)
	if exists == 0 {
		writeError(w, http.StatusNotFound, "target user not found")
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Demote old owner to admin
	_, err = tx.Exec(
		`INSERT INTO room_roles (room_id, user_id, role) VALUES ($1, $2, 'admin')
		ON CONFLICT (room_id, user_id) DO UPDATE SET role = 'admin'`,
		roomIDInt, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to demote old owner")
		return
	}

	// Promote new user to owner (sets both rooms.owner_id and room_roles)
	if err := setRoomOwner(tx, roomIDInt, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to promote new owner")
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "ownership transferred",
		"new_owner_id": req.UserID,
	})
}

// ---- Join room ----

func (h *RoomHandler) Join(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	var req struct {
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	room, err := getRoomByID(roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	if !room.IsActive {
		writeError(w, http.StatusBadRequest, "room is not active")
		return
	}

	// Check room expiry for non-shared rooms
	if !room.IsShared && room.ExpiresAt != nil {
		expiresAt, err := time.Parse(time.RFC3339, *room.ExpiresAt)
		if err == nil && expiresAt.Before(time.Now()) {
			writeError(w, http.StatusForbidden, "room has expired")
			return
		}
	}

	// Check data transfer cap
	resetDataCapIfNeeded(userID)
	var dailyBytes int64
	db.DB.QueryRow("SELECT daily_transfer_bytes FROM users WHERE id = $1", userID).Scan(&dailyBytes)
	if dailyBytes >= dataCapBytes {
		writeError(w, http.StatusForbidden, "daily data transfer cap (10GB) exceeded")
		return
	}

	// For shared rooms: check shard balance
	if room.IsShared && room.HourlyCost > 0 {
		var balance int
		db.DB.QueryRow("SELECT shard_balance FROM users WHERE id = $1", userID).Scan(&balance)
		if balance < room.HourlyCost {
			writeError(w, http.StatusPaymentRequired, fmt.Sprintf("insufficient shards for shared room: need at least %d per hour", room.HourlyCost))
			return
		}
	}

	// Check ban
	var banCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM room_bans WHERE room_id = $1 AND user_id = $2", room.ID, userID).Scan(&banCount)
	if banCount > 0 {
		writeError(w, http.StatusForbidden, "you are banned from this room")
		return
	}

	// Begin transaction
	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Lock room_members rows
	var memberCount int
	if err := tx.QueryRow(
		"SELECT COUNT(*) FROM room_members WHERE room_id = $1 FOR UPDATE",
		room.ID,
	).Scan(&memberCount); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Check if already a member
	var existingUsername, existingPassword string
	err = tx.QueryRow(
		"SELECT vpn_username, vpn_password FROM room_members WHERE room_id = $1 AND user_id = $2",
		room.ID, userID,
	).Scan(&existingUsername, &existingPassword)
	if err == nil {
		tx.Commit()
		node, err := getNodeByID(room.NodeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "node not found")
			return
		}
		writeJSON(w, http.StatusOK, model.JoinResponse{
			VPNHost:     node.Host,
			Hub:         room.HubName,
			VPNUsername: existingUsername,
			VPNPassword: existingPassword,
			Subnet:      room.Subnet,
		})
		return
	}

	// Check capacity
	if memberCount >= room.MaxPlayers {
		writeError(w, http.StatusConflict, "room is full")
		return
	}

	// Check private room password
	if room.IsPrivate {
		var passwordHash sql.NullString
		tx.QueryRow("SELECT password_hash FROM rooms WHERE id = $1", room.ID).Scan(&passwordHash)
		if passwordHash.Valid && passwordHash.String != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(passwordHash.String), []byte(req.Password)); err != nil {
				writeError(w, http.StatusForbidden, "incorrect room password")
				return
			}
		}
	}

	// Get node info
	node, err := getNodeByID(room.NodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "node not found")
		return
	}

	// Generate VPN credentials
	vpnUsername := fmt.Sprintf("u%d_%s", userID, randomHex(4))
	vpnPassword := randomHex(8)

	// Create VPN user on node
	vpnErr := service.CreateVPNUser(node.Host, node.APIPort, node.APISecret, room.HubName, vpnUsername, vpnPassword)
	if vpnErr != nil {
		log.Printf("[join] first VPN user creation failed for room %d, retrying in 2s: %v", room.ID, vpnErr)
		time.Sleep(2 * time.Second)
		vpnErr = service.CreateVPNUser(node.Host, node.APIPort, node.APISecret, room.HubName, vpnUsername, vpnPassword)
	}
	if vpnErr != nil {
		writeError(w, http.StatusBadGateway, "failed to create VPN user on node: "+vpnErr.Error())
		return
	}

	// Insert member record
	_, err = tx.Exec(
		"INSERT INTO room_members (room_id, user_id, vpn_username, vpn_password) VALUES ($1, $2, $3, $4)",
		room.ID, userID, vpnUsername, vpnPassword,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			tx.Rollback()
			var existUser, existPass string
			db.DB.QueryRow(
				"SELECT vpn_username, vpn_password FROM room_members WHERE room_id = $1 AND user_id = $2",
				room.ID, userID,
			).Scan(&existUser, &existPass)
			writeJSON(w, http.StatusOK, model.JoinResponse{
				VPNHost:     node.Host,
				Hub:         room.HubName,
				VPNUsername: existUser,
				VPNPassword: existPass,
				Subnet:      room.Subnet,
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to save membership")
		return
	}

	// Insert room_role as member if not exists
	tx.Exec(
		`INSERT INTO room_roles (room_id, user_id, role) VALUES ($1, $2, 'member')
		ON CONFLICT (room_id, user_id) DO NOTHING`,
		room.ID, userID,
	)

	// For shared rooms: create shared_session
	if room.IsShared {
		tx.Exec(
			"INSERT INTO shared_sessions (user_id, room_id) VALUES ($1, $2)",
			userID, room.ID,
		)
	}

	// Update last_activity
	tx.Exec("UPDATE rooms SET last_activity = CURRENT_TIMESTAMP WHERE id = $1", room.ID)

	// Record play session start
	tx.Exec("INSERT INTO play_sessions (user_id, room_id) VALUES ($1, $2)", userID, room.ID)

	// Increment total_sessions
	tx.Exec("UPDATE users SET total_sessions = total_sessions + 1 WHERE id = $1", userID)

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, model.JoinResponse{
		VPNHost:     node.Host,
		Hub:         room.HubName,
		VPNUsername: vpnUsername,
		VPNPassword: vpnPassword,
		Subnet:      room.Subnet,
	})
}

// ---- Leave room ----

func (h *RoomHandler) Leave(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	room, err := getRoomByID(roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	var vpnUsername string
	err = db.DB.QueryRow(
		"SELECT vpn_username FROM room_members WHERE room_id = $1 AND user_id = $2",
		room.ID, userID,
	).Scan(&vpnUsername)
	if err != nil {
		writeError(w, http.StatusBadRequest, "you are not a member of this room")
		return
	}

	node, err := getNodeByID(room.NodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "node not found")
		return
	}

	// Delete VPN user on node (best effort)
	service.DeleteVPNUser(node.Host, node.APIPort, node.APISecret, room.HubName, vpnUsername)

	db.DB.Exec("DELETE FROM room_members WHERE room_id = $1 AND user_id = $2", room.ID, userID)

	// For shared rooms: close shared session and charge
	if room.IsShared {
		closeSharedSession(userID, room.ID, room.HourlyCost)
	}

	// Remove room_role on leave (but not for owner)
	var role string
	err = db.DB.QueryRow("SELECT role FROM room_roles WHERE room_id = $1 AND user_id = $2", room.ID, userID).Scan(&role)
	if err == nil && role != "owner" {
		db.DB.Exec("DELETE FROM room_roles WHERE room_id = $1 AND user_id = $2", room.ID, userID)
	}

	// Close play session and update user stats
	closePlaySession(userID, room.ID)

	// Update room last_activity
	db.DB.Exec("UPDATE rooms SET last_activity = CURRENT_TIMESTAMP WHERE id = $1", room.ID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "left room successfully"})
}

// ---- Update password ----

func (h *RoomHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())

	room, err := getRoomByID(roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	if err := requireRoomPermission(room.ID, userID, isAdmin); err != nil {
		writeError(w, http.StatusForbidden, "only room owner/admin can update password")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Password) < 4 {
		writeError(w, http.StatusBadRequest, "password must be at least 4 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	_, err = db.DB.Exec("UPDATE rooms SET password_hash = $1, is_private = TRUE WHERE id = $2", string(hash), room.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// ---- Kick ----

func (h *RoomHandler) Kick(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())

	room, err := getRoomByID(roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	if err := requireRoomPermission(room.ID, userID, isAdmin); err != nil {
		writeError(w, http.StatusForbidden, "only room owner/admin can kick members")
		return
	}

	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := removeMemberFromRoom(room, req.UserID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user kicked"})
}

// ---- Ban ----

func (h *RoomHandler) Ban(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())

	room, err := getRoomByID(roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	if err := requireRoomPermission(room.ID, userID, isAdmin); err != nil {
		writeError(w, http.StatusForbidden, "only room owner/admin can ban users")
		return
	}

	var req struct {
		UserID int64  `json:"user_id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Kick if currently a member (ignore error if not a member)
	removeMemberFromRoom(room, req.UserID)

	// Add ban record
	_, err = db.DB.Exec(
		"INSERT INTO room_bans (room_id, user_id, banned_by, reason) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
		room.ID, req.UserID, userID, req.Reason,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to ban user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user banned"})
}

// ---- Unban ----

func (h *RoomHandler) Unban(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())

	room, err := getRoomByID(roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	if err := requireRoomPermission(room.ID, userID, isAdmin); err != nil {
		writeError(w, http.StatusForbidden, "only room owner/admin can unban users")
		return
	}

	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := db.DB.Exec("DELETE FROM room_bans WHERE room_id = $1 AND user_id = $2", room.ID, req.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unban user")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "ban not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user unbanned"})
}

// ---- Members ----

func (h *RoomHandler) Members(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")

	rows, err := db.DB.Query(`
		SELECT rm.user_id, u.display_name, rm.joined_at
		FROM room_members rm
		JOIN users u ON u.id = rm.user_id
		WHERE rm.room_id = $1
		ORDER BY rm.joined_at ASC`,
		roomID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	members := []model.RoomMember{}
	for rows.Next() {
		var m model.RoomMember
		if err := rows.Scan(&m.UserID, &m.DisplayName, &m.JoinedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		members = append(members, m)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"members": members})
}

// ---- Favorites ----

func (h *RoomHandler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	var exists int
	db.DB.QueryRow("SELECT COUNT(*) FROM rooms WHERE id = $1", roomID).Scan(&exists)
	if exists == 0 {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	_, err := db.DB.Exec(
		"INSERT INTO favorites (user_id, room_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		userID, roomID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add favorite")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "added to favorites"})
}

func (h *RoomHandler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	result, err := db.DB.Exec("DELETE FROM favorites WHERE user_id = $1 AND room_id = $2", userID, roomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove favorite")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "favorite not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "removed from favorites"})
}

func (h *RoomHandler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	rows, err := db.DB.Query(`
		SELECT r.id, r.name, r.hub_name, r.node_id, n.name, r.owner_id,
			COALESCE(u.display_name, ''), r.is_private, r.max_players,
			(SELECT COUNT(*) FROM room_members WHERE room_id = r.id) as current_players,
			r.subnet, r.is_active, r.game_tag, r.description,
			r.expires_at, r.is_shared, r.hourly_cost, r.created_at
		FROM favorites f
		JOIN rooms r ON r.id = f.room_id
		JOIN nodes n ON n.id = r.node_id
		LEFT JOIN users u ON u.id = r.owner_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	rooms := []model.Room{}
	for rows.Next() {
		var room model.Room
		var ownerDisplayName string
		err := rows.Scan(
			&room.ID, &room.Name, &room.HubName, &room.NodeID, &room.NodeName,
			&room.OwnerID, &ownerDisplayName, &room.IsPrivate, &room.MaxPlayers,
			&room.CurrentPlayers, &room.Subnet, &room.IsActive, &room.GameTag,
			&room.Description, &room.ExpiresAt, &room.IsShared, &room.HourlyCost,
			&room.CreatedAt,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		room.OwnerDisplayName = ownerDisplayName
		rooms = append(rooms, room)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"rooms": rooms})
}

// ---- Create invite ----

func (h *RoomHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())

	roomIDInt, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}

	// Only owner or admin of room can create invites
	if err := requireRoomPermission(roomIDInt, userID, isAdmin); err != nil {
		writeError(w, http.StatusForbidden, "only room owner/admin can create invites")
		return
	}

	var req struct {
		MaxUses     int `json:"max_uses"`
		ExpiresHours int `json:"expires_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.MaxUses < 0 {
		req.MaxUses = 0
	}

	// Generate random 12-char token (6 bytes = 12 hex chars)
	token := randomHex(6)

	var expiresAt *time.Time
	if req.ExpiresHours > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresHours) * time.Hour)
		expiresAt = &t
	}

	var inviteID int64
	err = db.DB.QueryRow(
		`INSERT INTO room_invites (room_id, token, created_by, max_uses, expires_at)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		roomIDInt, token, userID, req.MaxUses, expiresAt,
	).Scan(&inviteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create invite")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"invite_token": token,
		"invite_url":   "dotachi://join/" + token,
		"max_uses":     req.MaxUses,
		"expires_at":   expiresAt,
	})
}

// ---- Join via invite ----

func (h *RoomHandler) JoinInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	// Look up invite
	var inviteID, roomID int64
	var maxUses, usedCount int
	var expiresAt *time.Time
	err := db.DB.QueryRow(
		`SELECT id, room_id, max_uses, used_count, expires_at
		FROM room_invites WHERE token = $1`,
		req.Token,
	).Scan(&inviteID, &roomID, &maxUses, &usedCount, &expiresAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "invite token not found")
		return
	}

	// Check expiry
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		writeError(w, http.StatusGone, "invite token has expired")
		return
	}

	// Check max uses (0 means unlimited)
	if maxUses > 0 && usedCount >= maxUses {
		writeError(w, http.StatusConflict, "invite token has been fully used")
		return
	}

	// Check user not banned from room
	var banCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM room_bans WHERE room_id = $1 AND user_id = $2", roomID, userID).Scan(&banCount)
	if banCount > 0 {
		writeError(w, http.StatusForbidden, "you are banned from this room")
		return
	}

	// Get room info
	room, err := getRoomByID(strconv.FormatInt(roomID, 10))
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	if !room.IsActive {
		writeError(w, http.StatusBadRequest, "room is not active")
		return
	}

	// Check data transfer cap
	resetDataCapIfNeeded(userID)
	var dailyBytes int64
	db.DB.QueryRow("SELECT daily_transfer_bytes FROM users WHERE id = $1", userID).Scan(&dailyBytes)
	if dailyBytes >= dataCapBytes {
		writeError(w, http.StatusForbidden, "daily data transfer cap (10GB) exceeded")
		return
	}

	// For shared rooms: check shard balance
	if room.IsShared && room.HourlyCost > 0 {
		var balance int
		db.DB.QueryRow("SELECT shard_balance FROM users WHERE id = $1", userID).Scan(&balance)
		if balance < room.HourlyCost {
			writeError(w, http.StatusPaymentRequired, fmt.Sprintf("insufficient shards for shared room: need at least %d per hour", room.HourlyCost))
			return
		}
	}

	// Begin transaction for join
	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Lock room_members rows
	var memberCount int
	if err := tx.QueryRow(
		"SELECT COUNT(*) FROM room_members WHERE room_id = $1 FOR UPDATE",
		room.ID,
	).Scan(&memberCount); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Check if already a member
	var existingUsername, existingPassword string
	err = tx.QueryRow(
		"SELECT vpn_username, vpn_password FROM room_members WHERE room_id = $1 AND user_id = $2",
		room.ID, userID,
	).Scan(&existingUsername, &existingPassword)
	if err == nil {
		// Already a member, just increment used_count and return credentials
		db.DB.Exec("UPDATE room_invites SET used_count = used_count + 1 WHERE id = $1", inviteID)
		tx.Commit()
		node, err := getNodeByID(room.NodeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "node not found")
			return
		}
		writeJSON(w, http.StatusOK, model.JoinResponse{
			VPNHost:     node.Host,
			Hub:         room.HubName,
			VPNUsername: existingUsername,
			VPNPassword: existingPassword,
			Subnet:      room.Subnet,
		})
		return
	}

	// Check capacity
	if memberCount >= room.MaxPlayers {
		writeError(w, http.StatusConflict, "room is full")
		return
	}

	// Get node info
	node, err := getNodeByID(room.NodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "node not found")
		return
	}

	// Generate VPN credentials
	vpnUsername := fmt.Sprintf("u%d_%s", userID, randomHex(4))
	vpnPassword := randomHex(8)

	// Create VPN user on node
	vpnErr := service.CreateVPNUser(node.Host, node.APIPort, node.APISecret, room.HubName, vpnUsername, vpnPassword)
	if vpnErr != nil {
		log.Printf("[join-invite] first VPN user creation failed for room %d, retrying in 2s: %v", room.ID, vpnErr)
		time.Sleep(2 * time.Second)
		vpnErr = service.CreateVPNUser(node.Host, node.APIPort, node.APISecret, room.HubName, vpnUsername, vpnPassword)
	}
	if vpnErr != nil {
		writeError(w, http.StatusBadGateway, "failed to create VPN user on node: "+vpnErr.Error())
		return
	}

	// Insert member record
	_, err = tx.Exec(
		"INSERT INTO room_members (room_id, user_id, vpn_username, vpn_password) VALUES ($1, $2, $3, $4)",
		room.ID, userID, vpnUsername, vpnPassword,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save membership")
		return
	}

	// Insert room_role as member if not exists
	tx.Exec(
		`INSERT INTO room_roles (room_id, user_id, role) VALUES ($1, $2, 'member')
		ON CONFLICT (room_id, user_id) DO NOTHING`,
		room.ID, userID,
	)

	// For shared rooms: create shared_session
	if room.IsShared {
		tx.Exec("INSERT INTO shared_sessions (user_id, room_id) VALUES ($1, $2)", userID, room.ID)
	}

	// Update last_activity
	tx.Exec("UPDATE rooms SET last_activity = CURRENT_TIMESTAMP WHERE id = $1", room.ID)

	// Record play session start
	tx.Exec("INSERT INTO play_sessions (user_id, room_id) VALUES ($1, $2)", userID, room.ID)

	// Increment total_sessions
	tx.Exec("UPDATE users SET total_sessions = total_sessions + 1 WHERE id = $1", userID)

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	// Increment invite used_count (outside the main transaction, best effort)
	db.DB.Exec("UPDATE room_invites SET used_count = used_count + 1 WHERE id = $1", inviteID)

	writeJSON(w, http.StatusOK, model.JoinResponse{
		VPNHost:     node.Host,
		Hub:         room.HubName,
		VPNUsername: vpnUsername,
		VPNPassword: vpnPassword,
		Subnet:      room.Subnet,
	})
}

// ---- helpers ----

// setRoomOwner atomically sets both rooms.owner_id and room_roles for consistency.
// The execer parameter accepts either a *sql.Tx or *sql.DB.
func setRoomOwner(execer interface {
	Exec(string, ...interface{}) (sql.Result, error)
}, roomID int64, userID int64) error {
	_, err := execer.Exec("UPDATE rooms SET owner_id = $1 WHERE id = $2", userID, roomID)
	if err != nil {
		return err
	}
	_, err = execer.Exec(`
		INSERT INTO room_roles (room_id, user_id, role) VALUES ($1, $2, 'owner')
		ON CONFLICT (room_id, user_id) DO UPDATE SET role = 'owner'
	`, roomID, userID)
	return err
}

var validGameTags = map[string]bool{
	"dota2": true, "cs2": true, "warcraft3": true,
	"aoe2": true, "valorant": true, "minecraft": true, "other": true,
}

func getRoomByID(id string) (*model.Room, error) {
	var room model.Room
	var ownerDisplayName string
	err := db.DB.QueryRow(`
		SELECT r.id, r.name, r.hub_name, r.node_id, n.name, r.owner_id,
			COALESCE(u.display_name, ''), r.is_private, r.max_players,
			(SELECT COUNT(*) FROM room_members WHERE room_id = r.id) as current_players,
			r.subnet, r.is_active, r.game_tag, r.description,
			r.expires_at, r.is_shared, r.hourly_cost, r.created_at
		FROM rooms r
		JOIN nodes n ON n.id = r.node_id
		LEFT JOIN users u ON u.id = r.owner_id
		WHERE r.id = $1`,
		id,
	).Scan(
		&room.ID, &room.Name, &room.HubName, &room.NodeID, &room.NodeName,
		&room.OwnerID, &ownerDisplayName, &room.IsPrivate, &room.MaxPlayers,
		&room.CurrentPlayers, &room.Subnet, &room.IsActive, &room.GameTag,
		&room.Description, &room.ExpiresAt, &room.IsShared, &room.HourlyCost,
		&room.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	room.OwnerDisplayName = ownerDisplayName
	return &room, nil
}

func getNodeByID(id int64) (*model.Node, error) {
	var node model.Node
	err := db.DB.QueryRow(
		"SELECT id, name, host, api_port, api_secret, is_active, max_rooms, created_at FROM nodes WHERE id = $1",
		id,
	).Scan(&node.ID, &node.Name, &node.Host, &node.APIPort, &node.APISecret, &node.IsActive, &node.MaxRooms, &node.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func removeMemberFromRoom(room *model.Room, targetUserID int64) error {
	var vpnUsername string
	err := db.DB.QueryRow(
		"SELECT vpn_username FROM room_members WHERE room_id = $1 AND user_id = $2",
		room.ID, targetUserID,
	).Scan(&vpnUsername)
	if err != nil {
		return fmt.Errorf("user is not a member of this room")
	}

	node, err := getNodeByID(room.NodeID)
	if err != nil {
		return fmt.Errorf("node not found")
	}

	// Disconnect and delete VPN user on node (best effort)
	service.DisconnectUser(node.Host, node.APIPort, node.APISecret, room.HubName, vpnUsername)
	service.DeleteVPNUser(node.Host, node.APIPort, node.APISecret, room.HubName, vpnUsername)

	db.DB.Exec("DELETE FROM room_members WHERE room_id = $1 AND user_id = $2", room.ID, targetUserID)

	// For shared rooms: close shared session and charge
	if room.IsShared {
		closeSharedSession(targetUserID, room.ID, room.HourlyCost)
	}

	// Remove room_role (but not for owner)
	var role string
	roleErr := db.DB.QueryRow("SELECT role FROM room_roles WHERE room_id = $1 AND user_id = $2", room.ID, targetUserID).Scan(&role)
	if roleErr == nil && role != "owner" {
		db.DB.Exec("DELETE FROM room_roles WHERE room_id = $1 AND user_id = $2", room.ID, targetUserID)
	}

	// Close play session and update user stats
	closePlaySession(targetUserID, room.ID)

	// Update room last_activity
	db.DB.Exec("UPDATE rooms SET last_activity = CURRENT_TIMESTAMP WHERE id = $1", room.ID)

	return nil
}

func closePlaySession(userID int64, roomID int64) {
	var sessionID int64
	err := db.DB.QueryRow(
		"SELECT id FROM play_sessions WHERE user_id = $1 AND room_id = $2 AND left_at IS NULL ORDER BY joined_at DESC LIMIT 1",
		userID, roomID,
	).Scan(&sessionID)
	if err != nil {
		return
	}

	db.DB.Exec(`
		UPDATE play_sessions
		SET left_at = CURRENT_TIMESTAMP,
			duration_minutes = EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - joined_at))::INTEGER / 60
		WHERE id = $1`, sessionID)

	var durationMinutes int
	db.DB.QueryRow("SELECT COALESCE(duration_minutes, 0) FROM play_sessions WHERE id = $1", sessionID).Scan(&durationMinutes)
	if durationMinutes > 0 {
		hours := float64(durationMinutes) / 60.0
		db.DB.Exec("UPDATE users SET total_play_hours = total_play_hours + $1 WHERE id = $2", hours, userID)
	}
}

// closeSharedSession closes the open shared session for user in room and charges shards.
// All billing operations are wrapped in a transaction for consistency.
func closeSharedSession(userID int64, roomID int64, hourlyCost int) {
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var sessionID int64
	var startedAt time.Time
	var shardsCharged int
	err = tx.QueryRow(
		`SELECT id, started_at, shards_charged FROM shared_sessions
		WHERE user_id = $1 AND room_id = $2 AND ended_at IS NULL
		ORDER BY started_at DESC LIMIT 1`,
		userID, roomID,
	).Scan(&sessionID, &startedAt, &shardsCharged)
	if err != nil {
		return // no open session
	}

	// Calculate total hours (rounded up)
	elapsed := time.Since(startedAt)
	totalHours := int(math.Ceil(elapsed.Hours()))
	if totalHours < 1 {
		totalHours = 1
	}

	totalCost := totalHours * hourlyCost
	remainingCharge := totalCost - shardsCharged
	if remainingCharge < 0 {
		remainingCharge = 0
	}

	if remainingCharge > 0 {
		// Deduct from user balance
		var balance int
		tx.QueryRow("SELECT shard_balance FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&balance)
		newBalance := balance - remainingCharge
		if newBalance < 0 {
			newBalance = 0
		}
		tx.Exec("UPDATE users SET shard_balance = $1 WHERE id = $2", newBalance, userID)

		// Record transaction
		tx.Exec(
			`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description, ref_id)
			VALUES ($1, $2, $3, 'shared_hourly', $4, $5)`,
			userID, -remainingCharge, newBalance,
			fmt.Sprintf("Shared room hourly charge (%d hours)", totalHours),
			roomID,
		)
	}

	// Close the session
	tx.Exec(
		"UPDATE shared_sessions SET ended_at = CURRENT_TIMESTAMP, shards_charged = $1 WHERE id = $2",
		totalCost, sessionID,
	)

	tx.Commit()
}

// resetDataCapIfNeeded resets daily_transfer_bytes if the transfer_reset_date is not today.
func resetDataCapIfNeeded(userID int64) {
	var resetDate string
	err := db.DB.QueryRow("SELECT transfer_reset_date::TEXT FROM users WHERE id = $1", userID).Scan(&resetDate)
	if err != nil {
		return
	}
	today := time.Now().Format("2006-01-02")
	if resetDate != today {
		db.DB.Exec(
			"UPDATE users SET daily_transfer_bytes = 0, transfer_reset_date = CURRENT_DATE WHERE id = $1",
			userID,
		)
	}
}
