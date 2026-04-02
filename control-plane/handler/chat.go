package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dotachi/control-plane/db"
	"github.com/dotachi/control-plane/middleware"
	"github.com/go-chi/chi/v5"
)

type ChatHandler struct{}

func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}

// SendMessage handles POST /rooms/{id}/messages (authenticated, must be member).
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}

	// Check membership
	var memberCount int
	err = db.DB.QueryRow(
		"SELECT COUNT(*) FROM room_members WHERE room_id = $1 AND user_id = $2",
		roomID, userID,
	).Scan(&memberCount)
	if err != nil || memberCount == 0 {
		writeError(w, http.StatusForbidden, "you are not a member of this room")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	contentLen := len([]rune(req.Content))
	if contentLen < 1 || contentLen > 500 {
		writeError(w, http.StatusBadRequest, "content must be 1-500 characters")
		return
	}

	var msgID int64
	var createdAt string
	err = db.DB.QueryRow(
		`INSERT INTO room_messages (room_id, user_id, content)
		VALUES ($1, $2, $3) RETURNING id, created_at`,
		roomID, userID, req.Content,
	).Scan(&msgID, &createdAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	// Get user display name
	var displayName string
	db.DB.QueryRow("SELECT display_name FROM users WHERE id = $1", userID).Scan(&displayName)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":           msgID,
		"room_id":      roomID,
		"user_id":      userID,
		"display_name": displayName,
		"content":      req.Content,
		"created_at":   createdAt,
	})
}

// GetMessages handles GET /rooms/{id}/messages?after={id} (authenticated, must be member).
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}

	// Check membership
	var memberCount int
	err = db.DB.QueryRow(
		"SELECT COUNT(*) FROM room_members WHERE room_id = $1 AND user_id = $2",
		roomID, userID,
	).Scan(&memberCount)
	if err != nil || memberCount == 0 {
		writeError(w, http.StatusForbidden, "you are not a member of this room")
		return
	}

	afterIDStr := r.URL.Query().Get("after")
	var afterID int64
	if afterIDStr != "" {
		afterID, _ = strconv.ParseInt(afterIDStr, 10, 64)
	}

	type message struct {
		ID          int64  `json:"id"`
		RoomID      int64  `json:"room_id"`
		UserID      int64  `json:"user_id"`
		DisplayName string `json:"display_name"`
		Content     string `json:"content"`
		CreatedAt   string `json:"created_at"`
	}

	var rows_ interface{ Close() error }
	var messages []message

	if afterID > 0 {
		// Return messages after the given ID
		rowsResult, err := db.DB.Query(
			`SELECT m.id, m.room_id, m.user_id, u.display_name, m.content, m.created_at
			FROM room_messages m
			JOIN users u ON u.id = m.user_id
			WHERE m.room_id = $1 AND m.id > $2
			ORDER BY m.id ASC
			LIMIT 100`,
			roomID, afterID,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		rows_ = rowsResult
		for rowsResult.Next() {
			var m message
			if err := rowsResult.Scan(&m.ID, &m.RoomID, &m.UserID, &m.DisplayName, &m.Content, &m.CreatedAt); err != nil {
				writeError(w, http.StatusInternalServerError, "scan error")
				rowsResult.Close()
				return
			}
			messages = append(messages, m)
		}
	} else {
		// Return last 50 messages
		rowsResult, err := db.DB.Query(
			`SELECT m.id, m.room_id, m.user_id, u.display_name, m.content, m.created_at
			FROM room_messages m
			JOIN users u ON u.id = m.user_id
			WHERE m.room_id = $1
			ORDER BY m.id DESC
			LIMIT 50`,
			roomID,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		rows_ = rowsResult
		for rowsResult.Next() {
			var m message
			if err := rowsResult.Scan(&m.ID, &m.RoomID, &m.UserID, &m.DisplayName, &m.Content, &m.CreatedAt); err != nil {
				writeError(w, http.StatusInternalServerError, "scan error")
				rowsResult.Close()
				return
			}
			messages = append(messages, m)
		}
		// Reverse to chronological order
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}
	rows_.Close()

	if messages == nil {
		messages = []message{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"messages": messages})
}
