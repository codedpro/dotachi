package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dotachi/control-plane/db"
	"github.com/dotachi/control-plane/handler"
	"github.com/dotachi/control-plane/middleware"
	"github.com/dotachi/control-plane/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Set up structured JSON logging via slog
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := LoadConfig()

	if err := db.Init(cfg.DatabaseURL); err != nil {
		log.Fatalf("failed to init db: %v", err)
	}

	ensureAdmin(cfg)

	// Start room lifecycle cleanup worker
	idleTimeout, err := time.ParseDuration(cfg.RoomIdleTimeout)
	if err != nil {
		log.Printf("invalid ROOM_IDLE_TIMEOUT %q, using default 30m", cfg.RoomIdleTimeout)
		idleTimeout = 30 * time.Minute
	}
	service.StartCleanupWorker(5*time.Minute, idleTimeout)

	// Start node health checker -- marks nodes inactive if heartbeat is missed
	service.StartHealthChecker(60 * time.Second)

	// Start shared room billing worker (every 60 seconds)
	service.StartSharedBillingWorker(60 * time.Second)

	// Rate limiter stores
	normalLimiter := middleware.NewRateLimiterStore(60)  // 60 req/min for authenticated routes
	strictLimiter := middleware.NewRateLimiterStore(10)  // 10 req/min for auth routes (brute-force protection)

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.StructuredLogger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(corsMiddleware)

	// Handlers
	authH := handler.NewAuthHandler(cfg.JWTSecret)
	roomH := handler.NewRoomHandler()
	nodeH := handler.NewNodeHandler()
	adminH := handler.NewAdminHandler()
	monitorH := handler.NewMonitorHandler()
	heartbeatH := handler.NewHeartbeatHandler()
	promoH := handler.NewPromoHandler()
	chatH := handler.NewChatHandler()

	// Internal routes (node-agent heartbeat -- authenticated by X-Api-Secret, not JWT)
	r.Post("/internal/heartbeat", heartbeatH.Heartbeat)

	// Public routes (with strict rate limiting for auth endpoints)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RateLimit(strictLimiter))

		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
	})

	// Public (no auth) — pricing and shop info (normal rate limiting)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RateLimit(normalLimiter))

		r.Get("/rooms/pricing", roomH.Pricing)
		r.Get("/rooms/shop", roomH.Shop)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))
		r.Use(middleware.RateLimit(normalLimiter))

		// Auth
		r.Get("/auth/me", authH.Me)
		r.Get("/auth/me/stats", authH.MyStats)
		r.Patch("/auth/me/display-name", authH.UpdateDisplayName)
		r.Post("/auth/change-password", authH.ChangePassword)
		r.Get("/auth/me/referral", authH.MyReferral)

		// Promo codes
		r.Post("/promo/redeem", promoH.Redeem)

		// Rooms
		r.Get("/rooms", roomH.List)
		r.Get("/rooms/favorites", roomH.ListFavorites)
		r.Post("/rooms/purchase", roomH.PurchaseRoom)
		r.Get("/rooms/{id}", roomH.Get)
		r.Post("/rooms/{id}/join", roomH.Join)
		r.Post("/rooms/{id}/leave", roomH.Leave)
		r.Patch("/rooms/{id}/password", roomH.UpdatePassword)
		r.Post("/rooms/{id}/kick", roomH.Kick)
		r.Post("/rooms/{id}/ban", roomH.Ban)
		r.Post("/rooms/{id}/unban", roomH.Unban)
		r.Get("/rooms/{id}/members", roomH.Members)
		r.Post("/rooms/{id}/favorite", roomH.AddFavorite)
		r.Delete("/rooms/{id}/favorite", roomH.RemoveFavorite)
		r.Post("/rooms/{id}/extend", roomH.ExtendRoom)
		r.Post("/rooms/{id}/set-role", roomH.SetRole)
		r.Post("/rooms/{id}/transfer", roomH.TransferRoom)

		// Room invites
		r.Post("/rooms/{id}/invite", roomH.CreateInvite)
		r.Post("/rooms/join-invite", roomH.JoinInvite)

		// Room chat
		r.Post("/rooms/{id}/messages", chatH.SendMessage)
		r.Get("/rooms/{id}/messages", chatH.GetMessages)

		// Admin-only routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminOnly)

			r.Post("/nodes", nodeH.AddNode)
			r.Get("/nodes", nodeH.ListNodes)
			r.Post("/nodes/{id}/ping", nodeH.PingNode)

			r.Post("/admin/rooms", adminH.CreateRoom)
			r.Post("/admin/rooms/{id}/assign-owner", adminH.AssignOwner)
			r.Get("/admin/users", adminH.ListUsers)
			r.Post("/admin/users/{id}/add-shards", adminH.AddShards)
			r.Post("/admin/users/{id}/remove-shards", adminH.RemoveShards)
			r.Post("/admin/users/{id}/reset-device", adminH.ResetDeviceFingerprint)
			r.Post("/admin/users/{id}/delete", adminH.DeleteUser)

			// Promo admin
			r.Post("/admin/promo/create", promoH.CreatePromo)
			r.Get("/admin/promo/list", promoH.ListPromos)

			// Monitoring
			r.Get("/admin/monitor/overview", monitorH.Overview)
			r.Get("/admin/monitor/nodes", monitorH.Nodes)
			r.Get("/admin/monitor/room/{id}", monitorH.Room)
		})
	})

	log.Printf("Dotachi control plane listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func ensureAdmin(cfg *Config) {
	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE phone = $1", cfg.AdminPhone).Scan(&count)
	if count > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}

	_, err = db.DB.Exec(
		"INSERT INTO users (phone, password_hash, display_name, is_admin) VALUES ($1, $2, $3, TRUE)",
		cfg.AdminPhone, string(hash), "Admin",
	)
	if err != nil {
		log.Fatalf("failed to create admin user: %v", err)
	}
	log.Printf("default admin user created (phone: %s)", cfg.AdminPhone)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
