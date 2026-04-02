package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dotachi/node-agent/handler"
	"github.com/dotachi/node-agent/softether"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// ---- config from env ----
	listenAddr := envOr("LISTEN_ADDR", ":7443")
	apiSecret := os.Getenv("API_SECRET")
	vpncmdPath := envOr("VPNCMD_PATH", "/usr/local/vpnserver/vpncmd")
	serverHost := envOr("SERVER_HOST", "localhost")
	// Port the SoftEther server listens on for client VPN connections.
	// 443 disguises VPN as HTTPS — Iranian ISPs don't throttle port 443.
	vpnPort := envOr("VPN_PORT", "443")

	if apiSecret == "" {
		log.Fatal("API_SECRET environment variable is required")
	}

	// ---- dependencies ----
	se := &softether.Client{
		VpncmdPath: vpncmdPath,
		ServerHost: serverHost,
	}

	h := &handler.Handler{
		SE:      se,
		StartAt: time.Now(),
	}

	// ---- optimize SoftEther server on startup ----
	initSoftEther(se, vpnPort)

	// ---- heartbeat ----
	controlPlaneURL := os.Getenv("CONTROL_PLANE_URL")
	nodeName := envOr("NODE_NAME", getHostname())
	nodeHost := envOr("NODE_HOST", serverHost)
	nodePort, _ := strconv.Atoi(envOr("NODE_PORT", "7443"))

	if controlPlaneURL != "" {
		StartHeartbeat(controlPlaneURL, apiSecret, nodeName, nodeHost, nodePort, se)
	} else {
		log.Println("[heartbeat] CONTROL_PLANE_URL not set -- heartbeat disabled")
	}

	// ---- router ----
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	// Health check — no auth required.
	r.Get("/health", h.Health)

	// Authenticated routes.
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware(apiSecret))

		// Ping endpoint — the control plane calls POST /ping to check node liveness.
		r.Post("/ping", h.Health)

		r.Post("/hub/create", h.HubCreate)
		r.Post("/hub/delete", h.HubDelete)
		r.Get("/hub/status/{hub_name}", h.HubStatus)
		r.Get("/hub/user-traffic/{hub_name}/{username}", h.UserTraffic)

		r.Post("/user/create", h.UserCreate)
		r.Post("/user/delete", h.UserDelete)
		r.Post("/user/disconnect", h.UserDisconnect)

		r.Get("/stats", h.Stats)
	})

	// ---- serve ----
	log.Printf("node-agent listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// initSoftEther applies one-time server-level optimizations for LAN gaming in Iran.
// These persist across SoftEther restarts (stored in vpn_server.config).
// Safe to call on every node-agent start — idempotent.
func initSoftEther(se *softether.Client, vpnPort string) {
	log.Println("[init] applying SoftEther server optimizations...")

	// 1. Enable keepalive — SoftEther sends keepalive packets to detect dead connections fast.
	//    Without this, a dead connection isn't detected for 30+ seconds = game freeze.
	//    Interval 5s with UDP keepalive = detect dead links in ~10s.
	if _, err := se.ServerCmd("KeepEnable"); err != nil {
		log.Printf("[init] KeepEnable failed (non-fatal): %v", err)
	}
	if _, err := se.ServerCmd("KeepSet", "/HOST:keepalive.softether.org", "/PORT:80", "/INTERVAL:5", "/PROTOCOL:udp"); err != nil {
		log.Printf("[init] KeepSet failed (non-fatal): %v", err)
	}

	// 2. Add listener on port 443 — disguise as HTTPS.
	//    Iranian ISPs actively throttle unusual ports. Port 443 gets priority treatment
	//    because blocking it would break all HTTPS sites.
	//    SoftEther's protocol looks identical to TLS on the wire.
	if _, err := se.ServerCmd("ListenerCreate", vpnPort); err != nil {
		log.Printf("[init] ListenerCreate failed (non-fatal): %v", err)
	}

	// 3. UDP acceleration is configured per-client account, not server-wide.
	//    Client configures this via AccountProtoOptionsSet in ConnectVPN.

	// 4. Set server cipher to DHE-RSA-AES128-SHA — lighter than default AES256.
	//    For LAN gaming, AES128 is more than enough security and uses less CPU.
	//    On a 2-core VPS with 100 players, encryption CPU savings matter.
	if _, err := se.ServerCmd("ServerCipherSet", "DHE-RSA-AES128-SHA"); err != nil {
		log.Printf("[init] ServerCipherSet failed (non-fatal): %v", err)
	}

	log.Println("[init] SoftEther server optimizations applied")
}

// authMiddleware checks X-Api-Secret header against the expected secret.
func authMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Api-Secret") != secret {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
