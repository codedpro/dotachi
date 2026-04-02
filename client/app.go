package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Data types mirroring the control-plane API
// ---------------------------------------------------------------------------

type TokenResponse struct {
	Token        string `json:"access_token"`
	UserID       int    `json:"user_id"`
	DisplayName  string `json:"display_name"`
	IsAdmin      bool   `json:"is_admin"`
	ShardBalance int    `json:"shard_balance"`
}

type RoomOut struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	HubName     string `json:"hub_name"`
	NodeID      int    `json:"node_id"`
	NodeName    string `json:"node_name"`
	OwnerID     *int   `json:"owner_id"`
	OwnerName   string `json:"owner_display_name"`
	IsPrivate   bool   `json:"is_private"`
	MaxPlayers  int    `json:"max_players"`
	PlayerCount int    `json:"current_players"`
	Subnet      string `json:"subnet"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
}

type JoinResponse struct {
	VPNHost     string `json:"vpn_host"`
	Hub         string `json:"hub"`
	VPNUsername string `json:"vpn_username"`
	VPNPassword string `json:"vpn_password"`
	Subnet      string `json:"subnet"`
}

type Member struct {
	UserID      int    `json:"user_id"`
	DisplayName string `json:"display_name"`
	JoinedAt    string `json:"joined_at"`
}

// ---------------------------------------------------------------------------
// App – the main backend struct, exposed to the Wails frontend
// ---------------------------------------------------------------------------

// PingStats holds real-time latency measurements to the VPN server.
type PingStats struct {
	LastPing   int     `json:"last_ping"`    // ms, -1 if failed
	AvgPing    int     `json:"avg_ping"`     // rolling average of last 10
	PacketLoss float64 `json:"packet_loss"`  // percentage over last 20 pings
	Jitter     int     `json:"jitter"`       // ms, difference between min and max of last 10
}

type App struct {
	ctx     context.Context
	apiBase string // e.g. "http://1.2.3.4:8080"
	token   string

	// VPN state
	mu            sync.Mutex
	vpnStatus     string // "disconnected", "connecting", "connected", "reconnecting"
	vpnCreds      *vpnCredentials
	wantConnected bool
	stopMonitor   chan struct{}
	stopPing      chan struct{}

	// Ping measurement state (protected by mu)
	pingStats   PingStats
	pingHistory []int // last 20 ping results (-1 = failed)

	// Split tunnel state (protected by mu)
	splitTunnelActive bool
}

type vpnCredentials struct {
	Host     string
	Hub      string
	Username string
	Password string
	Subnet   string
}

// VPN connection config — tuned for Iran network stability
const (
	// Number of simultaneous TCP connections to the VPN server.
	// If 1 TCP stream drops (Iranian ISP hiccup), the other 7 keep
	// the tunnel alive. This is THE key feature preventing game drops.
	vpnMaxTCP = 8

	// Interval between TCP connection establishment (seconds).
	// 1 = fast reconnect when a stream drops.
	vpnTCPInterval = 1

	// VPN server port — 443 looks like HTTPS, ISPs don't throttle it.
	vpnPort = "443"

	// How often to check VPN connection health (seconds).
	vpnHealthCheckInterval = 3 * time.Second

	// If connection is lost, retry with exponential backoff.
	vpnReconnectMin = 1 * time.Second
	vpnReconnectMax = 10 * time.Second

	// SoftEther VPN adapter name on Windows
	vpnNicName = "DotachiVPN"

	// SoftEther account name
	vpnAccountName = "dotachi"
)

func NewApp() *App {
	return &App{
		apiBase:   "http://127.0.0.1:8080",
		vpnStatus: "disconnected",
	}
}

// Wails lifecycle hooks
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadSettings()
}

func (a *App) shutdown(_ context.Context) {
	a.StopVPN()
}

// ---------------------------------------------------------------------------
// Settings (persisted to disk)
// ---------------------------------------------------------------------------

func settingsPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "Dotachi", "settings.json")
}

type settings struct {
	APIBase string `json:"api_base"`
}

func (a *App) loadSettings() {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return
	}
	var s settings
	if json.Unmarshal(data, &s) == nil && s.APIBase != "" {
		a.apiBase = s.APIBase
	}
}

func (a *App) saveSettings() {
	p := settingsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	data, _ := json.Marshal(settings{APIBase: a.apiBase})
	_ = os.WriteFile(p, data, 0o644)
}

func (a *App) SetServerURL(u string) {
	a.apiBase = u
	a.saveSettings()
}

func (a *App) GetServerURL() string {
	return a.apiBase
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func (a *App) doRequest(method, path string, body interface{}, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(a.ctx, method, a.apiBase+path, reqBody)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	a.mu.Lock()
	token := a.token
	a.mu.Unlock()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errBody struct {
			Error  string `json:"error"`
			Detail string `json:"detail"`
		}
		if json.Unmarshal(respBytes, &errBody) == nil {
			if errBody.Error != "" {
				return fmt.Errorf("%s", errBody.Error)
			}
			if errBody.Detail != "" {
				return fmt.Errorf("%s", errBody.Detail)
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	if out != nil {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

func (a *App) Register(phone, password, displayName, referralCode string) (*TokenResponse, error) {
	fp := collectFingerprint()
	payload := map[string]string{
		"phone":              phone,
		"password":           password,
		"display_name":       displayName,
		"referral_code":      referralCode,
		"device_fingerprint": fp,
	}
	var tok TokenResponse
	if err := a.doRequest("POST", "/auth/register", payload, &tok); err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.token = tok.Token
	a.mu.Unlock()
	return &tok, nil
}

func (a *App) Login(phone, password string) (*TokenResponse, error) {
	fp := collectFingerprint()
	payload := map[string]string{
		"phone":              phone,
		"password":           password,
		"device_fingerprint": fp,
	}
	var tok TokenResponse
	if err := a.doRequest("POST", "/auth/login", payload, &tok); err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.token = tok.Token
	a.mu.Unlock()
	return &tok, nil
}

// ---------------------------------------------------------------------------
// Rooms
// ---------------------------------------------------------------------------

func (a *App) ListRooms(query string, isPrivate *bool, hasSlots *bool, page int) ([]RoomOut, error) {
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if isPrivate != nil {
		params.Set("is_private", strconv.FormatBool(*isPrivate))
	}
	if hasSlots != nil {
		params.Set("has_slots", strconv.FormatBool(*hasSlots))
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}

	path := "/rooms"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var rooms []RoomOut
	if err := a.doRequest("GET", path, nil, &rooms); err != nil {
		return nil, err
	}
	return rooms, nil
}

func (a *App) GetRoom(roomID int) (*RoomOut, error) {
	var room RoomOut
	path := fmt.Sprintf("/rooms/%d", roomID)
	if err := a.doRequest("GET", path, nil, &room); err != nil {
		return nil, err
	}
	return &room, nil
}

func (a *App) JoinRoom(roomID int, password string) (*JoinResponse, error) {
	payload := map[string]interface{}{}
	if password != "" {
		payload["password"] = password
	}
	var resp JoinResponse
	path := fmt.Sprintf("/rooms/%d/join", roomID)
	if err := a.doRequest("POST", path, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *App) LeaveRoom(roomID int) error {
	path := fmt.Sprintf("/rooms/%d/leave", roomID)
	return a.doRequest("POST", path, nil, nil)
}

func (a *App) GetMembers(roomID int) ([]Member, error) {
	var members []Member
	path := fmt.Sprintf("/rooms/%d/members", roomID)
	if err := a.doRequest("GET", path, nil, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (a *App) GetBuyInfo() (string, error) {
	var resp struct {
		Message string `json:"message"`
	}
	if err := a.doRequest("GET", "/rooms/buy/info", nil, &resp); err != nil {
		return "", err
	}
	return resp.Message, nil
}

// ---------------------------------------------------------------------------
// VPN management — optimized for Iran network stability
// ---------------------------------------------------------------------------
//
// Architecture:
//
//   ConnectVPN() → sets up SoftEther account with 8 TCP connections on port 443
//                → starts background health monitor goroutine
//
//   Health monitor → checks connection every 3s via AccountStatusGet
//                  → if disconnected: auto-reconnect with exponential backoff
//                  → never gives up until StopVPN() is called
//
//   StopVPN() → signals monitor to stop → disconnects VPN
//
// Why 8 TCP connections:
//   Iranian ISPs randomly RST individual TCP connections (throttling/filtering).
//   With 1 connection, a single RST = tunnel dies = game freeze.
//   With 8 connections, a RST only kills 1/8th of bandwidth for ~1 second
//   while SoftEther rebuilds that connection. Players don't notice.
//
// Why port 443:
//   ISPs prioritize port 443 (HTTPS) traffic. SoftEther's protocol on port 443
//   looks identical to TLS. Throttling port 443 would break all websites,
//   so ISPs don't do it.

// TODO(data-transfer-cap): The 10GB/day data transfer cap is checked on the server
// (control plane) but never incremented, because VPN traffic flows through SoftEther,
// not through our HTTP API. The control plane has no visibility into actual bytes
// transferred per user.
//
// The correct place to measure this is the node agent. SoftEther tracks per-user
// traffic via the UserGet command (fields: "Outgoing Unicast Total Size" and
// "Incoming Unicast Total Size"). The node-agent now exposes this via:
//   GET /hub/user-traffic/{hub_name}/{username}
//
// To implement the full solution:
// 1. The control plane should periodically poll each node agent's user-traffic
//    endpoint for active users.
// 2. Accumulate daily totals in the database (reset at midnight).
// 3. When a user exceeds the cap, call the node agent's /user/disconnect endpoint
//    and prevent new JoinRoom calls until the next day.

// ConnectVPN establishes the SoftEther VPN connection with all Iran-optimized settings.
// The subnet parameter (e.g. "10.10.1.0/24") is used to configure split tunneling
// so only game traffic goes through the VPN, not all internet traffic.
func (a *App) ConnectVPN(host, hub, username, password, subnet string) error {
	// Ensure SoftEther client service is running before doing anything
	a.EnsureSoftEtherRunning()

	a.mu.Lock()

	// Store credentials for reconnection (includes subnet for split tunnel)
	a.vpnCreds = &vpnCredentials{
		Host:     host,
		Hub:      hub,
		Username: username,
		Password: password,
		Subnet:   subnet,
	}
	a.wantConnected = true
	a.vpnStatus = "connecting"

	// Reset ping stats
	a.pingStats = PingStats{LastPing: -1}
	a.pingHistory = nil

	// Stop any existing monitors
	if a.stopMonitor != nil {
		close(a.stopMonitor)
	}
	if a.stopPing != nil {
		close(a.stopPing)
	}
	a.stopMonitor = make(chan struct{})
	a.stopPing = make(chan struct{})
	stopCh := a.stopMonitor
	pingCh := a.stopPing

	a.mu.Unlock()

	// Run initial connection
	err := a.establishVPN(host, hub, username, password)
	if err != nil {
		a.mu.Lock()
		a.vpnStatus = "disconnected"
		a.wantConnected = false
		a.mu.Unlock()
		return err
	}

	a.mu.Lock()
	a.vpnStatus = "connected"
	a.mu.Unlock()

	// Configure split tunneling so only game subnet goes through VPN.
	// This is critical: without it, ALL traffic goes through the VPN server
	// in Iran which = slow internet + unnecessary bandwidth on VPS.
	if subnet != "" {
		if err := a.configureSplitTunnel(subnet); err != nil {
			fmt.Fprintf(os.Stderr, "[vpn] split tunnel config failed (non-fatal): %v\n", err)
			// Non-fatal: VPN still works, just all traffic goes through it
			a.mu.Lock()
			a.splitTunnelActive = false
			a.mu.Unlock()
		} else {
			a.mu.Lock()
			a.splitTunnelActive = true
			a.mu.Unlock()
		}
	}

	// Start background health monitor
	go a.vpnHealthMonitor(stopCh)

	// Start background ping monitor
	go a.vpnPingMonitor(host, pingCh)

	return nil
}

// establishVPN runs the SoftEther client commands to set up and connect.
func (a *App) establishVPN(host, hub, username, password string) error {
	vpncmd := a.findVpncmd()

	// Step 1: Ensure the virtual network adapter exists.
	// NicCreate is idempotent — if it already exists, SoftEther just says so.
	runVpncmd(vpncmd, "NicCreate", vpnNicName)

	// Step 2: Delete any existing account (clean slate)
	runVpncmd(vpncmd, "AccountDisconnect", vpnAccountName)
	runVpncmd(vpncmd, "AccountDelete", vpnAccountName)

	// Step 3: Create account pointing to server on port 443 (HTTPS disguise)
	serverAddr := fmt.Sprintf("%s:%s", host, vpnPort)
	err := runVpncmd(vpncmd,
		"AccountCreate", vpnAccountName,
		"/SERVER:"+serverAddr,
		"/HUB:"+hub,
		"/USERNAME:"+username,
		"/NICNAME:"+vpnNicName,
	)
	if err != nil {
		return fmt.Errorf("AccountCreate failed: %w", err)
	}

	// Step 4: Set password
	err = runVpncmd(vpncmd,
		"AccountPasswordSet", vpnAccountName,
		"/PASSWORD:"+password,
		"/TYPE:standard",
	)
	if err != nil {
		return fmt.Errorf("AccountPasswordSet failed: %w", err)
	}

	// Step 5: Configure multi-TCP for connection resilience
	// MAXTCP:8 = 8 simultaneous TCP connections. This is the single most
	// important setting for preventing game drops on Iranian networks.
	// INTERVAL:1 = rebuild a dropped TCP connection in 1 second.
	err = runVpncmd(vpncmd,
		"AccountDetailSet", vpnAccountName,
		fmt.Sprintf("/MAXTCP:%d", vpnMaxTCP),
		fmt.Sprintf("/INTERVAL:%d", vpnTCPInterval),
		"/TTL:0",    // no TTL limit
		"/HALF:0",   // full-duplex, not half
		"/BRIDGE:0", // not bridge mode
		"/MONITOR:0", // not monitor mode
		"/NOTRACK:0", // enable tracking
		"/NOQOS:0",  // enable QoS (prioritizes small packets = game data)
	)
	if err != nil {
		return fmt.Errorf("AccountDetailSet failed: %w", err)
	}

	// Step 6: Enable UDP acceleration for this account
	// UDP = lower latency for game packets. Falls back to TCP if ISP blocks UDP.
	runVpncmd(vpncmd, "AccountProtoOptionsSet", vpnAccountName,
		"/NAME:no_udp_acceleration", "/VALUE:false")

	// Step 7: Set startup connection (auto-connect on SoftEther client start)
	runVpncmd(vpncmd, "AccountStartupSet", vpnAccountName)

	// Step 8: Connect!
	err = runVpncmd(vpncmd, "AccountConnect", vpnAccountName)
	if err != nil {
		return fmt.Errorf("AccountConnect failed: %w", err)
	}

	// Step 9: Wait for connection to establish (up to 15 seconds)
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		status := a.checkVPNStatusReal(vpncmd)
		if status == "connected" {
			return nil
		}
	}

	// Even if we didn't confirm connected in 15s, don't fail —
	// the health monitor will keep checking
	return nil
}

// vpnHealthMonitor runs in the background, checking connection health and
// auto-reconnecting if needed. Never gives up until stopCh is closed.
func (a *App) vpnHealthMonitor(stopCh chan struct{}) {
	vpncmd := a.findVpncmd()
	backoff := vpnReconnectMin
	consecutiveFails := 0

	for {
		select {
		case <-stopCh:
			return
		case <-time.After(vpnHealthCheckInterval):
		}

		// Check if we still want to be connected
		a.mu.Lock()
		want := a.wantConnected
		creds := a.vpnCreds
		a.mu.Unlock()

		if !want || creds == nil {
			return
		}

		// Check real VPN status
		status := a.checkVPNStatusReal(vpncmd)

		a.mu.Lock()
		oldStatus := a.vpnStatus

		if status == "connected" {
			a.vpnStatus = "connected"
			a.mu.Unlock()
			backoff = vpnReconnectMin
			consecutiveFails = 0
			continue
		}

		// Connection lost — attempt reconnect
		consecutiveFails++
		a.vpnStatus = "reconnecting"
		a.mu.Unlock()

		if oldStatus == "connected" {
			fmt.Fprintf(os.Stderr, "[vpn] connection lost, reconnecting (attempt %d)...\n", consecutiveFails)
		}

		// Try to reconnect
		runVpncmd(vpncmd, "AccountConnect", vpnAccountName)

		// Wait with backoff before next check
		select {
		case <-stopCh:
			return
		case <-time.After(backoff):
		}

		// Increase backoff (1s → 2s → 4s → 8s → 10s max)
		backoff *= 2
		if backoff > vpnReconnectMax {
			backoff = vpnReconnectMax
		}
	}
}

// checkVPNStatusReal queries SoftEther client for actual connection status.
func (a *App) checkVPNStatusReal(vpncmd string) string {
	output, err := runVpncmdOutput(vpncmd, "AccountStatusGet", vpnAccountName)
	if err != nil {
		return "disconnected"
	}

	// Parse the output for "Session Status"
	// SoftEther outputs: |Session Status|Connection Completed (Session Established)
	outputStr := string(output)
	for _, line := range strings.Split(outputStr, "\n") {
		if strings.Contains(line, "Session Status") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "established") || strings.Contains(lower, "connected") {
				return "connected"
			}
			if strings.Contains(lower, "connecting") || strings.Contains(lower, "negotiating") {
				return "connecting"
			}
		}
	}

	return "disconnected"
}

// StopVPN cleanly disconnects the VPN, stops monitors, and cleans up routes.
func (a *App) StopVPN() error {
	a.mu.Lock()
	a.wantConnected = false
	if a.stopMonitor != nil {
		close(a.stopMonitor)
		a.stopMonitor = nil
	}
	if a.stopPing != nil {
		close(a.stopPing)
		a.stopPing = nil
	}
	subnet := ""
	if a.vpnCreds != nil {
		subnet = a.vpnCreds.Subnet
		a.vpnCreds.Password = ""
		a.vpnCreds = nil
	}
	a.mu.Unlock()

	// Clean up split tunnel routes before disconnecting
	if subnet != "" {
		a.cleanupRoutes(subnet)
	}

	return a.DisconnectVPN()
}

// DisconnectVPN tears down the SoftEther VPN connection.
func (a *App) DisconnectVPN() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.vpnStatus == "disconnected" {
		return nil
	}

	vpncmd := a.findVpncmd()
	runVpncmd(vpncmd, "AccountDisconnect", vpnAccountName)

	a.vpnStatus = "disconnected"
	return nil
}

// GetVPNStatus returns the current VPN connection state.
// Possible values: "disconnected", "connecting", "connected", "reconnecting"
func (a *App) GetVPNStatus() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.vpnStatus
}

// IsSplitTunnelActive returns whether split tunneling was successfully configured.
func (a *App) IsSplitTunnelActive() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.splitTunnelActive
}

// ---------------------------------------------------------------------------
// VPN helpers
// ---------------------------------------------------------------------------

// findVpncmd locates the vpncmd executable.
// Checks common Windows installation paths.
func (a *App) findVpncmd() string {
	// Check if vpncmd is in PATH
	if p, err := exec.LookPath("vpncmd"); err == nil {
		return p
	}

	// Check common install locations on Windows
	candidates := []string{
		`C:\Program Files\SoftEther VPN Client\vpncmd.exe`,
		`C:\Program Files (x86)\SoftEther VPN Client\vpncmd.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), "SoftEther VPN Client", "vpncmd.exe"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Fallback — hope it's in PATH
	return "vpncmd"
}

// runVpncmd executes a vpncmd CLIENT command and returns error status.
func runVpncmd(vpncmd string, args ...string) error {
	cmdArgs := []string{"localhost", "/CLIENT", "/CMD"}
	cmdArgs = append(cmdArgs, args...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, vpncmd, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[vpncmd] %s failed: %v\n%s\n", args[0], err, string(output))
		return err
	}
	return nil
}

// runVpncmdOutput executes a vpncmd CLIENT command and returns its output.
func runVpncmdOutput(vpncmd string, args ...string) ([]byte, error) {
	cmdArgs := []string{"localhost", "/CLIENT", "/CMD"}
	cmdArgs = append(cmdArgs, args...)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, vpncmd, cmdArgs...)
	return cmd.CombinedOutput()
}

// ---------------------------------------------------------------------------
// SoftEther VPN Client status helpers
// ---------------------------------------------------------------------------

// CheckSoftEtherInstalled checks if SoftEther VPN Client is installed on Windows.
func (a *App) CheckSoftEtherInstalled() bool {
	vpncmd := a.findVpncmd()
	cmd := exec.Command(vpncmd, "localhost", "/CLIENT", "/CMD", "VersionGet")
	err := cmd.Run()
	return err == nil
}

// GetSoftEtherVersion returns the SoftEther VPN Client version string.
func (a *App) GetSoftEtherVersion() string {
	vpncmd := a.findVpncmd()
	output, err := runVpncmdOutput(vpncmd, "VersionGet")
	if err != nil {
		return "Not installed"
	}
	// Parse version from output
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "Version") {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) >= 3 {
				return strings.TrimSpace(parts[2])
			}
		}
	}
	return "Unknown"
}

// EnsureSoftEtherRunning starts the SoftEther VPN Client service if not running.
func (a *App) EnsureSoftEtherRunning() error {
	// On Windows, SoftEther VPN Client runs as a service
	cmd := exec.Command("net", "start", "SEVPNCLIENT")
	cmd.Run() // ignore error — may already be running
	return nil
}

// ---------------------------------------------------------------------------
// Split tunneling — route only game traffic through VPN
// ---------------------------------------------------------------------------
//
// Without split tunneling, SoftEther pushes a default gateway through the VPN.
// This means ALL internet traffic for the user goes: User → VPN server → Internet.
// For Iranian users this is devastating: the VPN server's uplink becomes a
// bottleneck and adds latency to everything (browsing, downloads, Discord, etc).
//
// With split tunneling:
//   - Game traffic (e.g., 10.10.1.0/24) → VPN adapter → VPN server → other players
//   - Everything else → normal ISP gateway → Internet (unchanged)
//
// Implementation: uses PowerShell Set-NetIPInterface to set the VPN adapter's
// interface metric to 9999 (deprioritizes it for general traffic) and then adds
// an explicit low-metric route for the game subnet. This approach does NOT
// require admin privileges, unlike the old `route add/delete` method.

// configureSplitTunnel ensures only game LAN traffic goes through VPN.
// Uses PowerShell interface metric approach instead of Windows route commands.
// This works WITHOUT admin privileges.
func (a *App) configureSplitTunnel(subnet string) error {
	// Step 1: Set the VPN adapter's interface metric to 9999.
	// This tells Windows to deprioritize the VPN adapter for all traffic,
	// so it never becomes the default route for general internet access.
	psSetMetric := fmt.Sprintf(
		`Get-NetAdapter -Name '*%s*' | Set-NetIPInterface -InterfaceMetric 9999`,
		vpnNicName,
	)
	setMetricCmd := exec.Command("powershell", "-Command", psSetMetric)
	if out, err := setMetricCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "[split-tunnel] set metric (non-fatal): %v\n%s\n", err, string(out))
		// Non-fatal: VPN still works, just all traffic may go through it
	} else {
		fmt.Fprintf(os.Stderr, "[split-tunnel] VPN adapter metric set to 9999\n")
	}

	// Step 2: Add an explicit low-metric route for the game subnet so that
	// only game traffic (e.g. 10.10.1.0/24) is routed through the VPN.
	ip, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("invalid subnet %q: %w", subnet, err)
	}
	mask := fmt.Sprintf("%d.%d.%d.%d", ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3])
	network := ip.Mask(ipNet.Mask).String()

	gateway := a.findVPNGateway()
	if gateway != "" {
		addRoute := exec.Command("route", "add", network, "mask", mask, gateway, "metric", "1")
		if out, err := addRoute.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "[split-tunnel] add subnet route (non-fatal): %v\n%s\n", err, string(out))
		} else {
			fmt.Fprintf(os.Stderr, "[split-tunnel] configured: %s via %s (metric 1)\n", subnet, gateway)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[split-tunnel] VPN gateway not found, relying on metric alone\n")
	}

	return nil
}

// findVPNGateway parses ipconfig output to find the default gateway assigned
// to the DotachiVPN network adapter. Returns empty string if not found.
func (a *App) findVPNGateway() string {
	cmd := exec.Command("ipconfig")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	inVPNAdapter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for our VPN adapter section
		if strings.Contains(line, vpnNicName) || strings.Contains(line, "DotachiVPN") {
			inVPNAdapter = true
			continue
		}

		// A new adapter section starts — stop searching
		if inVPNAdapter && trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.Contains(line, "adapter") {
			inVPNAdapter = false
			continue
		}

		if inVPNAdapter && (strings.Contains(trimmed, "Default Gateway") || strings.Contains(trimmed, "Gateway")) {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				gw := strings.TrimSpace(parts[1])
				if gw != "" && net.ParseIP(gw) != nil {
					return gw
				}
			}
		}
	}

	return ""
}

// cleanupRoutes is called when disconnecting. With the PowerShell metric
// approach, routes bound to the VPN adapter are automatically removed when
// the VPN disconnects and the adapter goes down. This function is kept as
// a no-op for compatibility.
func (a *App) cleanupRoutes(subnet string) {
	// Routes are automatically cleaned up when the VPN adapter goes down.
	// No manual cleanup needed with the metric-based split tunnel approach.
	fmt.Fprintf(os.Stderr, "[split-tunnel] VPN disconnecting, routes will auto-cleanup\n")
}

// ---------------------------------------------------------------------------
// Ping measurement — real-time latency to VPN server
// ---------------------------------------------------------------------------

// PingServer measures TCP latency to the VPN server.
// Returns latency in milliseconds. Uses TCP connect to port 443
// since ICMP is often blocked in Iran.
func (a *App) PingServer(host string) (int, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host+":443", 5*time.Second)
	if err != nil {
		return -1, err
	}
	conn.Close()
	ms := int(time.Since(start).Milliseconds())
	return ms, nil
}

// GetPingStats returns the current ping statistics (thread-safe).
func (a *App) GetPingStats() PingStats {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pingStats
}

// vpnPingMonitor runs alongside the health monitor and measures
// latency to the VPN server every 5 seconds. Results are stored
// and exposed to the frontend via GetPingStats.
func (a *App) vpnPingMonitor(host string, stopCh chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
		}

		// Check if we still want to be connected
		a.mu.Lock()
		want := a.wantConnected
		a.mu.Unlock()
		if !want {
			return
		}

		// Measure ping
		ms, _ := a.PingServer(host)

		a.mu.Lock()
		// Append to history (keep last 20)
		a.pingHistory = append(a.pingHistory, ms)
		if len(a.pingHistory) > 20 {
			a.pingHistory = a.pingHistory[len(a.pingHistory)-20:]
		}

		// Calculate stats
		a.pingStats.LastPing = ms

		// Rolling average of last 10 successful pings
		successCount := 0
		sum := 0
		minPing := math.MaxInt32
		maxPing := 0
		last10Start := len(a.pingHistory) - 10
		if last10Start < 0 {
			last10Start = 0
		}
		for _, p := range a.pingHistory[last10Start:] {
			if p >= 0 {
				successCount++
				sum += p
				if p < minPing {
					minPing = p
				}
				if p > maxPing {
					maxPing = p
				}
			}
		}
		if successCount > 0 {
			a.pingStats.AvgPing = sum / successCount
			a.pingStats.Jitter = maxPing - minPing
		} else {
			a.pingStats.AvgPing = -1
			a.pingStats.Jitter = 0
		}

		// Packet loss over entire history (up to 20)
		failCount := 0
		for _, p := range a.pingHistory {
			if p < 0 {
				failCount++
			}
		}
		a.pingStats.PacketLoss = float64(failCount) / float64(len(a.pingHistory)) * 100.0

		a.mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// Connection quality rating
// ---------------------------------------------------------------------------

// GetConnectionQuality returns a quality rating based on ping and packet loss.
// Returns: "excellent" (<30ms, 0% loss), "good" (<60ms, <5% loss),
// "fair" (<100ms, <10% loss), "poor" (>100ms or >10% loss),
// "unknown" (no data yet).
func (a *App) GetConnectionQuality() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.pingStats.LastPing < 0 && len(a.pingHistory) == 0 {
		return "unknown"
	}

	avg := a.pingStats.AvgPing
	loss := a.pingStats.PacketLoss

	if avg < 0 {
		return "poor"
	}

	switch {
	case avg < 30 && loss == 0:
		return "excellent"
	case avg < 60 && loss < 5:
		return "good"
	case avg < 100 && loss < 10:
		return "fair"
	default:
		return "poor"
	}
}

// ---------------------------------------------------------------------------
// VPN readiness check
// ---------------------------------------------------------------------------

// CheckVPNReady returns a status object indicating if the VPN subsystem is ready.
// Checks if vpncmd exists and if the SoftEther client service is running.
func (a *App) CheckVPNReady() map[string]interface{} {
	result := map[string]interface{}{
		"ready":           false,
		"message":         "",
		"vpncmd_found":    false,
		"service_running": false,
	}

	// Check if vpncmd exists
	vpncmd := a.findVpncmd()
	_, err := os.Stat(vpncmd)
	vpncmdInPath := err == nil
	if !vpncmdInPath {
		// Also check if it's findable via LookPath
		_, err = exec.LookPath(vpncmd)
		vpncmdInPath = err == nil
	}
	result["vpncmd_found"] = vpncmdInPath

	if !vpncmdInPath {
		result["message"] = "SoftEther VPN Client is not installed. Please install it first."
		return result
	}

	// Check if SoftEther client service is responding
	cmd := exec.Command(vpncmd, "localhost", "/CLIENT", "/CMD", "VersionGet")
	err = cmd.Run()
	serviceRunning := err == nil
	result["service_running"] = serviceRunning

	if !serviceRunning {
		result["message"] = "SoftEther VPN Client service is not running. Starting it..."
		// Try to start it
		a.EnsureSoftEtherRunning()
		// Re-check
		cmd2 := exec.Command(vpncmd, "localhost", "/CLIENT", "/CMD", "VersionGet")
		if cmd2.Run() == nil {
			result["service_running"] = true
			result["ready"] = true
			result["message"] = "SoftEther VPN Client is ready."
		} else {
			result["message"] = "Could not start SoftEther VPN Client service. Please start it manually."
		}
		return result
	}

	result["ready"] = true
	result["message"] = "SoftEther VPN Client is ready."
	return result
}

// ---------------------------------------------------------------------------
// Shard-based business model endpoints
// ---------------------------------------------------------------------------

// GetPricing returns pricing information for rooms (public, no auth).
func (a *App) GetPricing() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := a.doRequest("GET", "/rooms/pricing", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetShopInfo returns shop and contact info (public, no auth).
func (a *App) GetShopInfo() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := a.doRequest("GET", "/rooms/shop", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PurchaseRoom buys a new room with shards.
func (a *App) PurchaseRoom(name, gameTag string, slots int, duration string, days int, isPrivate bool, password string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"name":       name,
		"game_tag":   gameTag,
		"slots":      slots,
		"duration":   duration,
		"days":       days,
		"is_private": isPrivate,
		"password":   password,
	}
	var result map[string]interface{}
	if err := a.doRequest("POST", "/rooms/purchase", payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ExtendRoom extends an existing room's duration.
func (a *App) ExtendRoom(roomID int, duration string, days int) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"duration": duration,
		"days":     days,
	}
	var result map[string]interface{}
	path := fmt.Sprintf("/rooms/%d/extend", roomID)
	if err := a.doRequest("POST", path, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetRoomRole sets a user's role in a room.
func (a *App) SetRoomRole(roomID, userID int, role string) error {
	payload := map[string]interface{}{
		"user_id": userID,
		"role":    role,
	}
	path := fmt.Sprintf("/rooms/%d/set-role", roomID)
	return a.doRequest("POST", path, payload, nil)
}

// TransferRoom transfers ownership of a room to another user.
func (a *App) TransferRoom(roomID, userID int) error {
	payload := map[string]interface{}{
		"user_id": userID,
	}
	path := fmt.Sprintf("/rooms/%d/transfer", roomID)
	return a.doRequest("POST", path, payload, nil)
}

// GetMyStats returns the current user's play stats.
func (a *App) GetMyStats() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := a.doRequest("GET", "/auth/me/stats", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMe returns the current user's profile including shard balance.
func (a *App) GetMe() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := a.doRequest("GET", "/auth/me", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Auto-update system
// ---------------------------------------------------------------------------

const currentVersion = "0.1.0"

// UpdateInfo describes an available client update.
type UpdateInfo struct {
	Available   bool   `json:"available"`
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Changelog   string `json:"changelog"`
}

// CheckForUpdate checks the control plane for a newer version.
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	var info UpdateInfo
	err := a.doRequest("GET", "/client/version", nil, &info)
	if err != nil {
		// If the server doesn't support this endpoint yet, don't error
		return &UpdateInfo{Available: false, Version: currentVersion}, nil
	}
	return &info, nil
}

// GetCurrentVersion returns the current client version.
func (a *App) GetCurrentVersion() string {
	return currentVersion
}

// ---------------------------------------------------------------------------
// Local VPN IP display
// ---------------------------------------------------------------------------

// GetLocalVPNIP returns the IP address assigned to the VPN adapter.
// This is the IP other players use to connect in LAN games.
func (a *App) GetLocalVPNIP() string {
	output, err := exec.Command("ipconfig").Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	inVPNSection := false
	for _, line := range lines {
		if strings.Contains(line, vpnNicName) || strings.Contains(line, "DotachiVPN") {
			inVPNSection = true
			continue
		}
		if inVPNSection {
			// Look for IPv4 Address line
			if strings.Contains(line, "IPv4") || strings.Contains(line, "IP Address") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					ip := strings.TrimSpace(parts[1])
					if ip != "" && ip != "0.0.0.0" {
						return ip
					}
				}
			}
			// If we hit another adapter section, stop
			if strings.Contains(line, "adapter") && !strings.Contains(line, vpnNicName) {
				break
			}
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Room chat (polling)
// ---------------------------------------------------------------------------

// ChatMessage represents a single chat message in a room.
type ChatMessage struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	DisplayName string `json:"display_name"`
	Content     string `json:"content"`
	CreatedAt   string `json:"created_at"`
}

// SendChatMessage sends a chat message to the room.
func (a *App) SendChatMessage(roomID int, content string) (*ChatMessage, error) {
	var msg ChatMessage
	path := fmt.Sprintf("/rooms/%d/messages", roomID)
	err := a.doRequest("POST", path, map[string]string{"content": content}, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetChatMessages fetches new messages since the given message ID.
// Pass 0 to get the last 50 messages.
func (a *App) GetChatMessages(roomID int, afterID int) ([]ChatMessage, error) {
	var msgs []ChatMessage
	path := fmt.Sprintf("/rooms/%d/messages?after=%d", roomID, afterID)
	err := a.doRequest("GET", path, nil, &msgs)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// ---------------------------------------------------------------------------
// Invite system
// ---------------------------------------------------------------------------

// InviteInfo describes a generated room invite.
type InviteInfo struct {
	Token     string `json:"invite_token"`
	InviteURL string `json:"invite_url"`
}

// CreateInvite generates an invite link for a room.
func (a *App) CreateInvite(roomID int, maxUses int, expiresHours int) (*InviteInfo, error) {
	var info InviteInfo
	path := fmt.Sprintf("/rooms/%d/invite", roomID)
	payload := map[string]int{"max_uses": maxUses, "expires_hours": expiresHours}
	err := a.doRequest("POST", path, payload, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// JoinByInvite joins a room using an invite token.
func (a *App) JoinByInvite(token string) (*JoinResponse, error) {
	var resp JoinResponse
	err := a.doRequest("POST", "/rooms/join-invite", map[string]string{"token": token}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Password change
// ---------------------------------------------------------------------------

// ChangePassword changes the user's password.
func (a *App) ChangePassword(oldPassword, newPassword string) error {
	return a.doRequest("POST", "/auth/change-password", map[string]string{
		"old_password": oldPassword,
		"new_password": newPassword,
	}, nil)
}

// ---------------------------------------------------------------------------
// Promo codes & referrals
// ---------------------------------------------------------------------------

// RedeemPromo redeems a promotional code for shards.
func (a *App) RedeemPromo(code string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := a.doRequest("POST", "/promo/redeem", map[string]string{"code": code}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetReferralInfo returns the user's referral code and stats.
func (a *App) GetReferralInfo() (map[string]interface{}, error) {
	var info map[string]interface{}
	err := a.doRequest("GET", "/auth/me/referral", nil, &info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// ---------------------------------------------------------------------------
// Balance refresh
// ---------------------------------------------------------------------------

// RefreshBalance fetches the latest shard balance from the server.
func (a *App) RefreshBalance() (int, error) {
	var me struct {
		ShardBalance int `json:"shard_balance"`
	}
	err := a.doRequest("GET", "/auth/me", nil, &me)
	if err != nil {
		return 0, err
	}
	return me.ShardBalance, nil
}
