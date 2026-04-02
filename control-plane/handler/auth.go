package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dotachi/control-plane/db"
	"github.com/dotachi/control-plane/middleware"
	"github.com/dotachi/control-plane/model"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	JWTSecret string
}

func NewAuthHandler(jwtSecret string) *AuthHandler {
	return &AuthHandler{JWTSecret: jwtSecret}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone             string `json:"phone"`
		Password          string `json:"password"`
		DisplayName       string `json:"display_name"`
		ReferralCode      string `json:"referral_code"`
		DeviceFingerprint string `json:"device_fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 20 {
		writeError(w, http.StatusBadRequest, "phone must be 10-20 characters")
		return
	}
	if len(req.Password) < 4 {
		writeError(w, http.StatusBadRequest, "password must be at least 4 characters")
		return
	}
	if len(req.DisplayName) < 1 || len(req.DisplayName) > 64 {
		writeError(w, http.StatusBadRequest, "display_name must be 1-64 characters")
		return
	}

	// Device fingerprint validation: 1 account per device
	// Fingerprint is a SHA-256 hash of hardware identifiers (BIOS serial, CPU ID,
	// disk serial, Windows Machine GUID). If any existing user has the same
	// fingerprint, reject the registration.
	if req.DeviceFingerprint != "" && len(req.DeviceFingerprint) >= 16 {
		var existingCount int
		err := db.DB.QueryRow(
			"SELECT COUNT(*) FROM users WHERE device_fingerprint = $1",
			req.DeviceFingerprint,
		).Scan(&existingCount)
		if err == nil && existingCount > 0 {
			writeError(w, http.StatusConflict, "یک حساب کاربری قبلاً از این دستگاه ثبت شده است")
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	// Generate a random 8-char referral code for the new user
	newReferralCode := generateReferralCode()

	// Look up referrer if a referral code was provided
	var referrerID *int64
	if req.ReferralCode != "" {
		var rid int64
		err := db.DB.QueryRow("SELECT id FROM users WHERE referral_code = $1", req.ReferralCode).Scan(&rid)
		if err == nil {
			referrerID = &rid
		}
	}

	// Store the device fingerprint (nullable — if client doesn't send it, we allow but log)
	var fpPtr *string
	if req.DeviceFingerprint != "" && len(req.DeviceFingerprint) >= 16 {
		fpPtr = &req.DeviceFingerprint
	}

	var id int64
	err = db.DB.QueryRow(
		"INSERT INTO users (phone, password_hash, display_name, referral_code, referred_by, device_fingerprint) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		req.Phone, string(hash), req.DisplayName, newReferralCode, referrerID, fpPtr,
	).Scan(&id)
	if err != nil {
		writeError(w, http.StatusConflict, "phone number already registered")
		return
	}

	// If valid referral: give referrer 10,000 shards bonus
	if referrerID != nil {
		go func(refID int64) {
			tx, err := db.DB.Begin()
			if err != nil {
				return
			}
			defer tx.Rollback()

			var balance int
			err = tx.QueryRow("SELECT shard_balance FROM users WHERE id = $1 FOR UPDATE", refID).Scan(&balance)
			if err != nil {
				return
			}
			newBalance := balance + 10000
			tx.Exec("UPDATE users SET shard_balance = $1 WHERE id = $2", newBalance, refID)
			tx.Exec(
				`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description)
				VALUES ($1, $2, $3, 'referral_bonus', $4)`,
				refID, 10000, newBalance, fmt.Sprintf("Referral bonus for user #%d", id),
			)
			tx.Commit()
		}(*referrerID)
	}

	token, err := h.generateToken(id, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"access_token":  token,
		"user_id":       id,
		"display_name":  req.DisplayName,
		"is_admin":      false,
		"shard_balance": 0,
	})
}

func generateReferralCode() string {
	b := make([]byte, 4) // 4 bytes = 8 hex chars
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone             string `json:"phone"`
		Password          string `json:"password"`
		DeviceFingerprint string `json:"device_fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var user model.User
	var passwordHash string
	err := db.DB.QueryRow(
		`SELECT id, phone, password_hash, display_name, is_admin, shard_balance,
			total_play_hours, total_sessions, created_at
		FROM users WHERE phone = $1`,
		req.Phone,
	).Scan(&user.ID, &user.Phone, &passwordHash, &user.DisplayName, &user.IsAdmin, &user.ShardBalance,
		&user.TotalPlayHours, &user.TotalSessions, &user.CreatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusUnauthorized, "invalid phone or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid phone or password")
		return
	}

	// Update device fingerprint on login (non-blocking, for tracking)
	// This lets admin see which device a user logged in from last
	if req.DeviceFingerprint != "" && len(req.DeviceFingerprint) >= 16 {
		go func() {
			db.DB.Exec("UPDATE users SET device_fingerprint = $1 WHERE id = $2",
				req.DeviceFingerprint, user.ID)
		}()
	}

	token, err := h.generateToken(user.ID, user.IsAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  token,
		"user_id":       user.ID,
		"display_name":  user.DisplayName,
		"is_admin":      user.IsAdmin,
		"shard_balance": user.ShardBalance,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var user model.User
	err := db.DB.QueryRow(
		`SELECT id, phone, display_name, is_admin, created_at,
			shard_balance, total_play_hours, total_sessions
		FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Phone, &user.DisplayName, &user.IsAdmin, &user.CreatedAt,
		&user.ShardBalance, &user.TotalPlayHours, &user.TotalSessions)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) MyStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var stats model.PlayerStats
	var shardBalance int
	err := db.DB.QueryRow(
		`SELECT total_play_hours, total_sessions, shard_balance, created_at
		FROM users WHERE id = $1`, userID,
	).Scan(&stats.TotalPlayHours, &stats.TotalSessions, &shardBalance, &stats.MemberSince)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	stats.ShardBalance = shardBalance

	db.DB.QueryRow(
		"SELECT COUNT(*) FROM room_roles WHERE user_id = $1 AND role = 'owner'", userID,
	).Scan(&stats.RoomsOwned)

	db.DB.QueryRow(
		"SELECT COUNT(*) FROM favorites WHERE user_id = $1", userID,
	).Scan(&stats.FavoriteCount)

	writeJSON(w, http.StatusOK, stats)
}

func (h *AuthHandler) UpdateDisplayName(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.DisplayName) < 1 || len(req.DisplayName) > 64 {
		writeError(w, http.StatusBadRequest, "display_name must be 1-64 characters")
		return
	}

	_, err := db.DB.Exec("UPDATE users SET display_name = $1 WHERE id = $2", req.DisplayName, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update display name")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"display_name": req.DisplayName})
}

// ChangePassword handles POST /auth/change-password (authenticated).
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.NewPassword) < 4 {
		writeError(w, http.StatusBadRequest, "new password must be at least 4 characters")
		return
	}

	// Get current password hash
	var currentHash string
	err := db.DB.QueryRow("SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.OldPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "old password is incorrect")
		return
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	_, err = db.DB.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", string(newHash), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}

// MyReferral handles GET /auth/me/referral (authenticated).
func (h *AuthHandler) MyReferral(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var referralCode sql.NullString
	err := db.DB.QueryRow("SELECT referral_code FROM users WHERE id = $1", userID).Scan(&referralCode)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	code := ""
	if referralCode.Valid {
		code = referralCode.String
	}

	var referralCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE referred_by = $1", userID).Scan(&referralCount)

	totalEarned := referralCount * 10000

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"referral_code":  code,
		"referral_count": referralCount,
		"total_earned":   totalEarned,
	})
}

func (h *AuthHandler) generateToken(userID int64, isAdmin bool) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"is_admin": isAdmin,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.JWTSecret))
}
