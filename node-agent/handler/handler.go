package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dotachi/node-agent/softether"
	"github.com/go-chi/chi/v5"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	SE      *softether.Client
	StartAt time.Time
}

// ---------- request/response types ----------

type HubCreateReq struct {
	HubName     string `json:"hub_name"`
	MaxSessions int    `json:"max_sessions"`
	Subnet      string `json:"subnet"`
}

type HubDeleteReq struct {
	HubName string `json:"hub_name"`
}

type UserCreateReq struct {
	HubName  string `json:"hub_name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserDeleteReq struct {
	HubName  string `json:"hub_name"`
	Username string `json:"username"`
}

type UserDisconnectReq struct {
	HubName  string `json:"hub_name"`
	Username string `json:"username"`
}

type HubStatusResp struct {
	Sessions   int    `json:"sessions"`
	TrafficIn  uint64 `json:"traffic_in"`
	TrafficOut uint64 `json:"traffic_out"`
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeBody(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

// SubnetParams derives DHCP range parameters from a CIDR like "10.10.1.0/24".
func SubnetParams(cidr string) (start, end, mask, gw string, err error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", "", "", fmt.Errorf("invalid subnet: %w", err)
	}

	base := ip.To4()
	if base == nil {
		return "", "", "", "", fmt.Errorf("only IPv4 subnets are supported")
	}

	maskBytes := ipNet.Mask
	mask = fmt.Sprintf("%d.%d.%d.%d", maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3])

	// Gateway = .1
	gwIP := make(net.IP, 4)
	copy(gwIP, base)
	gwIP[3] = 1
	gw = gwIP.String()

	// DHCP start = .10, end = .200
	startIP := make(net.IP, 4)
	copy(startIP, base)
	startIP[3] = 10
	start = startIP.String()

	endIP := make(net.IP, 4)
	copy(endIP, base)
	endIP[3] = 200
	end = endIP.String()

	return start, end, mask, gw, nil
}

// ---------- handlers ----------

// HubCreate handles POST /hub/create.
// Optimized for LAN gaming in Iran:
//   - Pure L2 bridge (no NAT routing overhead)
//   - MTU 1400 to prevent fragmentation over VPN
//   - 24h DHCP lease so IPs never change mid-game
//   - Iranian-friendly DNS
//   - No gateway = no NAT engine activation = lower latency
func (h *Handler) HubCreate(w http.ResponseWriter, r *http.Request) {
	var req HubCreateReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.HubName == "" || req.Subnet == "" {
		writeErr(w, http.StatusBadRequest, "hub_name and subnet are required")
		return
	}

	start, end, mask, gw, err := SubnetParams(req.Subnet)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	maxSess := req.MaxSessions
	if maxSess <= 0 {
		maxSess = 100
	}

	// === Step 1: Create the hub ===
	if _, err := h.SE.ServerCmd("HubCreate", req.HubName, "/PASSWORD:none"); err != nil {
		writeErr(w, http.StatusInternalServerError, "HubCreate failed: "+err.Error())
		return
	}

	// === Step 2: Set max sessions (slot limit) ===
	if _, err := h.SE.HubCmd(req.HubName, "SetMaxSession", strconv.Itoa(maxSess)); err != nil {
		writeErr(w, http.StatusInternalServerError, "SetMaxSession failed: "+err.Error())
		return
	}

	// === Step 3: Enable SecureNAT (provides DHCP server for the hub) ===
	// SecureNAT runs in userspace but we disable the NAT routing part
	// by setting no gateway — this makes it a pure L2 DHCP server.
	if _, err := h.SE.HubCmd(req.HubName, "SecureNatEnable"); err != nil {
		writeErr(w, http.StatusInternalServerError, "SecureNatEnable failed: "+err.Error())
		return
	}

	// === Step 4: DHCP config — optimized for LAN gaming ===
	// - GW:none (0.0.0.0) = no default gateway = pure L2, no NAT routing overhead
	//   Players only need to see each other, NOT route to internet through VPN.
	//   This eliminates the NAT engine entirely = lower CPU, lower latency.
	// - EXPIRE:86400 (24 hours) = IP never changes during a gaming session.
	//   Short leases cause DHCP renewal which can stutter the connection for 50-200ms.
	// - DNS:none — no DNS pushed through VPN. Players don't need DNS for LAN gaming,
	//   and all Iranian DNS servers have issues. The player's real network handles DNS.
	dhcpArgs := fmt.Sprintf(
		"/START:%s /END:%s /MASK:%s /EXPIRE:86400 /GW:%s /DNS:0.0.0.0 /DNS2:0.0.0.0",
		start, end, mask, gw,
	)
	if _, err := h.SE.HubCmd(req.HubName, "DhcpSet", dhcpArgs); err != nil {
		writeErr(w, http.StatusInternalServerError, "DhcpSet failed: "+err.Error())
		return
	}

	// === Step 5: NAT settings — tuned for game stability ===
	// - MTU:1400 — VPN encapsulation adds ~60 bytes. MTU 1500 causes fragmentation
	//   at the ISP level which = packet loss and 20-50ms jitter spikes.
	//   1400 is safe for all Iranian ISPs (even those with PPPoE overhead).
	// - TCPTIMEOUT:86400 — never timeout TCP inside the tunnel during a session.
	// - UDPTIMEOUT:3600 — game traffic is UDP. 10min timeout was too aggressive;
	//   if a player pauses/ALT-TABs for 10min their UDP state would expire.
	//   1 hour is safe.
	if _, err := h.SE.HubCmd(req.HubName, "NatSet",
		"/MTU:1400", "/TCPTIMEOUT:86400", "/UDPTIMEOUT:3600"); err != nil {
		writeErr(w, http.StatusInternalServerError, "NatSet failed: "+err.Error())
		return
	}

	// === Step 6: Hub options for broadcast optimization ===
	// NoArpPolling, NoIPv6DefaultRouterInRA, NoMacAddressLog are set via
	// vpn_server.config directly if needed — there is no vpncmd command for these.
	// SoftEther does not have a "SetHubOption" command.

	log.Printf("[hub/create] created hub %s (max=%d, subnet=%s, MTU=1400, L2-optimized)", req.HubName, maxSess, req.Subnet)
	writeJSON(w, http.StatusOK, map[string]string{"status": "created", "hub_name": req.HubName})
}

// HubDelete handles POST /hub/delete.
func (h *Handler) HubDelete(w http.ResponseWriter, r *http.Request) {
	var req HubDeleteReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.HubName == "" {
		writeErr(w, http.StatusBadRequest, "hub_name is required")
		return
	}

	if _, err := h.SE.ServerCmd("HubDelete", req.HubName); err != nil {
		writeErr(w, http.StatusInternalServerError, "HubDelete failed: "+err.Error())
		return
	}

	log.Printf("[hub/delete] deleted hub %s", req.HubName)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "hub_name": req.HubName})
}

// HubStatus handles GET /hub/status/{hub_name}.
func (h *Handler) HubStatus(w http.ResponseWriter, r *http.Request) {
	hubName := chi.URLParam(r, "hub_name")
	if hubName == "" {
		writeErr(w, http.StatusBadRequest, "hub_name is required")
		return
	}

	out, err := h.SE.HubCmd(hubName, "StatusGet")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "StatusGet failed: "+err.Error())
		return
	}

	resp := parseHubStatus(out)
	writeJSON(w, http.StatusOK, resp)
}

// parseHubStatus extracts session count and traffic from StatusGet output.
func parseHubStatus(output string) HubStatusResp {
	var resp HubStatusResp

	getValue := func(key string) string {
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, key) {
				parts := strings.SplitN(line, "|", 3)
				if len(parts) >= 3 {
					return strings.TrimSpace(parts[2])
				}
			}
		}
		return ""
	}

	for _, key := range []string{"Sessions (Client)", "Num Sessions", "Sessions"} {
		v := getValue(key)
		if v != "" {
			resp.Sessions, _ = strconv.Atoi(strings.ReplaceAll(v, ",", ""))
			break
		}
	}

	numRe := regexp.MustCompile(`[0-9]+`)

	inStr := getValue("Incoming Unicast Total Size")
	if inStr == "" {
		inStr = getValue("Incoming Data Size")
	}
	if m := numRe.FindString(strings.ReplaceAll(inStr, ",", "")); m != "" {
		resp.TrafficIn, _ = strconv.ParseUint(m, 10, 64)
	}

	outStr := getValue("Outgoing Unicast Total Size")
	if outStr == "" {
		outStr = getValue("Outgoing Data Size")
	}
	if m := numRe.FindString(strings.ReplaceAll(outStr, ",", "")); m != "" {
		resp.TrafficOut, _ = strconv.ParseUint(m, 10, 64)
	}

	return resp
}

// UserCreate handles POST /user/create.
func (h *Handler) UserCreate(w http.ResponseWriter, r *http.Request) {
	var req UserCreateReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.HubName == "" || req.Username == "" || req.Password == "" {
		writeErr(w, http.StatusBadRequest, "hub_name, username, and password are required")
		return
	}

	if _, err := h.SE.HubCmd(req.HubName, "UserCreate", req.Username,
		"/GROUP:none", "/REALNAME:none", "/NOTE:none"); err != nil {
		writeErr(w, http.StatusInternalServerError, "UserCreate failed: "+err.Error())
		return
	}

	if _, err := h.SE.HubCmd(req.HubName, "UserPasswordSet", req.Username,
		"/PASSWORD:"+req.Password); err != nil {
		writeErr(w, http.StatusInternalServerError, "UserPasswordSet failed: "+err.Error())
		return
	}

	log.Printf("[user/create] created user %s in hub %s", req.Username, req.HubName)
	writeJSON(w, http.StatusOK, map[string]string{"status": "created", "username": req.Username})
}

// UserDelete handles POST /user/delete.
func (h *Handler) UserDelete(w http.ResponseWriter, r *http.Request) {
	var req UserDeleteReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.HubName == "" || req.Username == "" {
		writeErr(w, http.StatusBadRequest, "hub_name and username are required")
		return
	}

	if _, err := h.SE.HubCmd(req.HubName, "UserDelete", req.Username); err != nil {
		writeErr(w, http.StatusInternalServerError, "UserDelete failed: "+err.Error())
		return
	}

	log.Printf("[user/delete] deleted user %s from hub %s", req.Username, req.HubName)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "username": req.Username})
}

// UserDisconnect handles POST /user/disconnect.
func (h *Handler) UserDisconnect(w http.ResponseWriter, r *http.Request) {
	var req UserDisconnectReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.HubName == "" || req.Username == "" {
		writeErr(w, http.StatusBadRequest, "hub_name and username are required")
		return
	}

	out, err := h.SE.HubCmd(req.HubName, "SessionList")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "SessionList failed: "+err.Error())
		return
	}

	sessionName := findSessionByUser(out, req.Username)
	if sessionName == "" {
		writeErr(w, http.StatusNotFound, "no active session found for user "+req.Username)
		return
	}

	if _, err := h.SE.HubCmd(req.HubName, "SessionDisconnect", sessionName); err != nil {
		writeErr(w, http.StatusInternalServerError, "SessionDisconnect failed: "+err.Error())
		return
	}

	log.Printf("[user/disconnect] disconnected session %s (user %s) from hub %s",
		sessionName, req.Username, req.HubName)
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "disconnected",
		"username": req.Username,
		"session":  sessionName,
	})
}

// findSessionByUser parses SessionList output to find a session name for the given username.
func findSessionByUser(output, username string) string {
	for _, line := range strings.Split(output, "\n") {
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		sessUser := strings.TrimSpace(parts[3])
		if strings.EqualFold(sessUser, username) {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

// Health handles GET /health.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.StartAt).Seconds()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"uptime": int64(uptime),
	})
}

// Stats handles GET /stats — returns aggregate node statistics.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	uptimeSecs := int64(time.Since(h.StartAt).Seconds())

	hubCount := 0
	totalSessions := 0

	// Parse ServerStatusGet output to get hub and session counts.
	out, err := h.SE.ServerCmd("ServerStatusGet")
	if err == nil {
		hubCount = ParseServerStatusInt(out, "Number of Virtual Hubs")
		totalSessions = ParseServerStatusInt(out, "Number of Sessions")
		if totalSessions == 0 {
			totalSessions = ParseServerStatusInt(out, "Num Sessions")
		}
	} else {
		log.Printf("[stats] ServerStatusGet failed: %v", err)
	}

	cpuUsage := ReadCPUUsage()
	memoryMB := ReadMemoryUsageMB()

	writeJSON(w, http.StatusOK, map[string]any{
		"uptime":         uptimeSecs,
		"hub_count":      hubCount,
		"total_sessions": totalSessions,
		"cpu_usage":      cpuUsage,
		"memory_mb":      memoryMB,
	})
}

// ParseServerStatusInt extracts an integer value from vpncmd ServerStatusGet output.
func ParseServerStatusInt(output, key string) int {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, key) {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) >= 2 {
				valStr := strings.TrimSpace(parts[len(parts)-1])
				valStr = strings.ReplaceAll(valStr, ",", "")
				v, _ := strconv.Atoi(valStr)
				return v
			}
		}
	}
	return 0
}

// ReadCPUUsage reads /proc/stat to estimate current CPU usage percentage.
// Returns 0 on non-Linux systems or if /proc/stat is unreadable.
func ReadCPUUsage() float64 {
	if runtime.GOOS != "linux" {
		return 0
	}

	readSample := func() (idle, total uint64, ok bool) {
		f, err := os.Open("/proc/stat")
		if err != nil {
			return 0, 0, false
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "cpu ") {
				fields := strings.Fields(line)
				if len(fields) < 5 {
					return 0, 0, false
				}
				var sum uint64
				for _, field := range fields[1:] {
					v, _ := strconv.ParseUint(field, 10, 64)
					sum += v
				}
				idleVal, _ := strconv.ParseUint(fields[4], 10, 64)
				return idleVal, sum, true
			}
		}
		return 0, 0, false
	}

	idle1, total1, ok1 := readSample()
	if !ok1 {
		return 0
	}
	time.Sleep(200 * time.Millisecond)
	idle2, total2, ok2 := readSample()
	if !ok2 {
		return 0
	}

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	if totalDelta == 0 {
		return 0
	}
	usage := (1.0 - idleDelta/totalDelta) * 100.0
	// Round to one decimal place
	return float64(int(usage*10)) / 10.0
}

// ReadMemoryUsageMB reads /proc/meminfo and returns used memory in MB.
// Returns 0 on non-Linux systems or if /proc/meminfo is unreadable.
func ReadMemoryUsageMB() int64 {
	if runtime.GOOS != "linux" {
		return 0
	}

	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	values := map[string]int64{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, key := range []string{"MemTotal", "MemAvailable"} {
			if strings.HasPrefix(line, key+":") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					v, _ := strconv.ParseInt(fields[1], 10, 64)
					values[key] = v // value is in kB
				}
			}
		}
	}

	total, hasTotal := values["MemTotal"]
	avail, hasAvail := values["MemAvailable"]
	if hasTotal && hasAvail {
		return (total - avail) / 1024 // kB to MB
	}
	return 0
}

// UserTraffic handles GET /hub/user-traffic/{hub_name}/{username}.
// Returns per-user traffic stats from SoftEther's UserGet command.
// The control plane uses this to enforce data transfer caps (e.g. 10GB/day)
// because VPN traffic flows through SoftEther, not through the HTTP API,
// so the control plane cannot measure it directly.
func (h *Handler) UserTraffic(w http.ResponseWriter, r *http.Request) {
	hubName := chi.URLParam(r, "hub_name")
	username := chi.URLParam(r, "username")
	if hubName == "" || username == "" {
		writeErr(w, http.StatusBadRequest, "hub_name and username are required")
		return
	}

	out, err := h.SE.HubCmd(hubName, "UserGet", username)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "UserGet failed: "+err.Error())
		return
	}

	incoming := parseTrafficBytes(out, "Incoming Unicast Total Size")
	outgoing := parseTrafficBytes(out, "Outgoing Unicast Total Size")

	writeJSON(w, http.StatusOK, map[string]uint64{
		"incoming_bytes": incoming,
		"outgoing_bytes": outgoing,
		"total_bytes":    incoming + outgoing,
	})
}

// parseTrafficBytes extracts a byte count from vpncmd UserGet output.
// Lines look like: "Incoming Unicast Total Size|123,456 bytes"
func parseTrafficBytes(output, key string) uint64 {
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, key) {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}
		valStr := parts[len(parts)-1]
		// Strip commas, "bytes" suffix, and whitespace
		valStr = strings.ReplaceAll(valStr, ",", "")
		valStr = strings.ReplaceAll(valStr, "bytes", "")
		valStr = strings.TrimSpace(valStr)
		v, _ := strconv.ParseUint(valStr, 10, 64)
		return v
	}
	return 0
}
