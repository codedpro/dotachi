package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dotachi/control-plane/db"
	"github.com/dotachi/control-plane/middleware"
)

type PromoHandler struct{}

func NewPromoHandler() *PromoHandler {
	return &PromoHandler{}
}

// Redeem handles POST /promo/redeem (authenticated).
func (h *PromoHandler) Redeem(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	// Look up the promo code and lock the row
	var codeID, shardAmount, maxUses, usedCount int
	var expiresAt *time.Time
	err = tx.QueryRow(
		`SELECT id, shard_amount, max_uses, used_count, expires_at
		FROM promo_codes WHERE code = $1 FOR UPDATE`,
		req.Code,
	).Scan(&codeID, &shardAmount, &maxUses, &usedCount, &expiresAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "promo code not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Check expiry
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		writeError(w, http.StatusGone, "promo code has expired")
		return
	}

	// Check max uses
	if usedCount >= maxUses {
		writeError(w, http.StatusConflict, "promo code has been fully redeemed")
		return
	}

	// Check if user already redeemed
	var alreadyRedeemed int
	err = tx.QueryRow(
		"SELECT COUNT(*) FROM promo_redemptions WHERE code_id = $1 AND user_id = $2",
		codeID, userID,
	).Scan(&alreadyRedeemed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if alreadyRedeemed > 0 {
		writeError(w, http.StatusConflict, "you have already redeemed this code")
		return
	}

	// Add shards to user balance
	var balance int
	err = tx.QueryRow("SELECT shard_balance FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	const maxShardBalance = 999_999_999 // ~1 billion shards
	newBalance := balance + shardAmount
	if newBalance > maxShardBalance {
		writeError(w, http.StatusBadRequest, "balance would exceed maximum")
		return
	}
	_, err = tx.Exec("UPDATE users SET shard_balance = $1 WHERE id = $2", newBalance, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update balance")
		return
	}

	// Record redemption
	_, err = tx.Exec(
		"INSERT INTO promo_redemptions (code_id, user_id) VALUES ($1, $2)",
		codeID, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record redemption")
		return
	}

	// Increment used_count
	_, err = tx.Exec("UPDATE promo_codes SET used_count = used_count + 1 WHERE id = $1", codeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update promo code")
		return
	}

	// Record shard transaction
	_, err = tx.Exec(
		`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description)
		VALUES ($1, $2, $3, 'promo_redeem', $4)`,
		userID, shardAmount, newBalance, "Promo code: "+req.Code,
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
		"message":       "promo code redeemed",
		"shards_added":  shardAmount,
		"new_balance":   newBalance,
	})
}

// CreatePromo handles POST /admin/promo/create (admin only).
func (h *PromoHandler) CreatePromo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code        string  `json:"code"`
		ShardAmount int     `json:"shard_amount"`
		MaxUses     int     `json:"max_uses"`
		ExpiresAt   *string `json:"expires_at"` // optional RFC3339
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}
	if req.ShardAmount <= 0 {
		writeError(w, http.StatusBadRequest, "shard_amount must be positive")
		return
	}
	if req.MaxUses <= 0 {
		req.MaxUses = 1
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid expires_at format, must be RFC3339")
			return
		}
		expiresAt = &t
	}

	var id int
	err := db.DB.QueryRow(
		`INSERT INTO promo_codes (code, shard_amount, max_uses, expires_at)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		req.Code, req.ShardAmount, req.MaxUses, expiresAt,
	).Scan(&id)
	if err != nil {
		writeError(w, http.StatusConflict, "promo code already exists")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":           id,
		"code":         req.Code,
		"shard_amount": req.ShardAmount,
		"max_uses":     req.MaxUses,
		"expires_at":   expiresAt,
	})
}

// ListPromos handles GET /admin/promo/list (admin only).
func (h *PromoHandler) ListPromos(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(
		`SELECT id, code, shard_amount, max_uses, used_count, expires_at, created_at
		FROM promo_codes ORDER BY id DESC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	type promoRow struct {
		ID          int     `json:"id"`
		Code        string  `json:"code"`
		ShardAmount int     `json:"shard_amount"`
		MaxUses     int     `json:"max_uses"`
		UsedCount   int     `json:"used_count"`
		ExpiresAt   *string `json:"expires_at"`
		CreatedAt   string  `json:"created_at"`
	}

	promos := []promoRow{}
	for rows.Next() {
		var p promoRow
		if err := rows.Scan(&p.ID, &p.Code, &p.ShardAmount, &p.MaxUses, &p.UsedCount, &p.ExpiresAt, &p.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		promos = append(promos, p)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"promo_codes": promos})
}
