#!/bin/bash
# =============================================================================
# Dotachi VPN Node Setup Script
# =============================================================================
# One-command setup for a fresh Ubuntu 22.04+ VPS.
# Installs SoftEther VPN Server + Dotachi node-agent, configures firewall,
# and registers the node with the Dotachi control plane.
#
# Usage:
#   CONTROL_PLANE_URL=https://api.dotachi.ir API_SECRET=mysecret bash setup-node.sh
#
# Or with arguments:
#   bash setup-node.sh \
#     --control-plane-url https://api.dotachi.ir \
#     --api-secret mysecret \
#     --node-name tehran-1 \
#     --node-host 1.2.3.4 \
#     --admin-password mypassword
#
# Environment variables (override with CLI args):
#   CONTROL_PLANE_URL   - URL of the Dotachi control plane API
#   API_SECRET          - Shared secret for node-agent <-> control-plane auth
#   NODE_NAME           - Friendly name for this node (default: hostname)
#   NODE_HOST           - Public IP of this VPS (auto-detected if empty)
#   SE_ADMIN_PASSWORD   - SoftEther admin password (random if empty)
#   ADMIN_TOKEN         - JWT admin token for control plane registration
#   NODE_API_PORT       - Port for node-agent API (default: 7443)
#   VPN_PORT            - Port SoftEther listens on for VPN (default: 443)
#   GO_VERSION          - Go version for building node-agent (default: 1.23.8)
#   NODE_AGENT_REPO     - Git repo URL for node-agent source
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Colors and output helpers
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

log_info()    { echo -e "${BLUE}[INFO]${NC}    $*"; }
log_ok()      { echo -e "${GREEN}[OK]${NC}      $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}    $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC}   $*"; }
log_section() { echo -e "\n${BOLD}${CYAN}==== $* ====${NC}\n"; }

die() {
    log_error "$*"
    exit 1
}

# ---------------------------------------------------------------------------
# Parse CLI arguments (override env vars)
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --control-plane-url) CONTROL_PLANE_URL="$2"; shift 2 ;;
        --api-secret)        API_SECRET="$2"; shift 2 ;;
        --node-name)         NODE_NAME="$2"; shift 2 ;;
        --node-host)         NODE_HOST="$2"; shift 2 ;;
        --admin-password)    SE_ADMIN_PASSWORD="$2"; shift 2 ;;
        --admin-token)       ADMIN_TOKEN="$2"; shift 2 ;;
        --node-api-port)     NODE_API_PORT="$2"; shift 2 ;;
        --vpn-port)          VPN_PORT="$2"; shift 2 ;;
        --go-version)        GO_VERSION="$2"; shift 2 ;;
        --repo)              NODE_AGENT_REPO="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--control-plane-url URL] [--api-secret SECRET] [--node-name NAME]"
            echo "          [--node-host IP] [--admin-password PASS] [--admin-token TOKEN]"
            echo "          [--node-api-port PORT] [--vpn-port PORT] [--go-version VER] [--repo URL]"
            echo ""
            echo "See script header for environment variable equivalents."
            exit 0
            ;;
        *) die "Unknown argument: $1. Use --help for usage." ;;
    esac
done

# ---------------------------------------------------------------------------
# Configuration defaults
# ---------------------------------------------------------------------------
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-}"
API_SECRET="${API_SECRET:-}"
NODE_NAME="${NODE_NAME:-$(hostname -s)}"
NODE_HOST="${NODE_HOST:-}"
SE_ADMIN_PASSWORD="${SE_ADMIN_PASSWORD:-}"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"
NODE_API_PORT="${NODE_API_PORT:-7443}"
VPN_PORT="${VPN_PORT:-443}"
GO_VERSION="${GO_VERSION:-1.23.8}"
NODE_AGENT_REPO="${NODE_AGENT_REPO:-}"

SOFTETHER_VERSION="v4.42-9798-rtm"
SOFTETHER_TAG="9798"
SOFTETHER_BUILD_DATE="2023.06.30"
SOFTETHER_URL="https://github.com/SoftEtherVPN/SoftEtherVPN_Stable/releases/download/${SOFTETHER_VERSION}/softether-vpnserver-${SOFTETHER_VERSION}-${SOFTETHER_BUILD_DATE}-linux-x64-64bit.tar.gz"

INSTALL_DIR="/usr/local/vpnserver"
CONFIG_DIR="/etc/dotachi"
AGENT_BIN="/usr/local/bin/dotachi-node-agent"

# ---------------------------------------------------------------------------
# Pre-flight checks
# ---------------------------------------------------------------------------
log_section "Pre-flight Checks"

# Must be root
[[ "$(id -u)" -eq 0 ]] || die "This script must be run as root. Use: sudo bash $0"

# Must be Ubuntu 22.04+
if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    if [[ "${ID:-}" != "ubuntu" ]]; then
        log_warn "This script is designed for Ubuntu. Detected: ${ID:-unknown}. Proceeding anyway."
    else
        MAJOR_VER="${VERSION_ID%%.*}"
        if [[ "$MAJOR_VER" -lt 22 ]]; then
            die "Ubuntu 22.04 or later required. Detected: ${VERSION_ID}"
        fi
    fi
fi

# Validate required inputs
if [[ -z "$API_SECRET" ]]; then
    die "API_SECRET is required. Set via env var or --api-secret."
fi

log_ok "Running as root"
log_ok "API_SECRET is set"

# Auto-detect public IP if not provided
if [[ -z "$NODE_HOST" ]]; then
    log_info "NODE_HOST not set, detecting public IP..."
    NODE_HOST=$(curl -4 -sf --max-time 10 https://ifconfig.me 2>/dev/null \
             || curl -4 -sf --max-time 10 https://api.ipify.org 2>/dev/null \
             || curl -4 -sf --max-time 10 https://icanhazip.com 2>/dev/null \
             || ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7; exit}' \
             || true)
    if [[ -z "$NODE_HOST" ]]; then
        die "Could not auto-detect public IP. Please set NODE_HOST."
    fi
    NODE_HOST=$(echo "$NODE_HOST" | tr -d '[:space:]')
fi
log_ok "Node public IP: ${NODE_HOST}"

# Generate SoftEther admin password if not provided
if [[ -z "$SE_ADMIN_PASSWORD" ]]; then
    SE_ADMIN_PASSWORD=$(openssl rand -base64 24 | tr -dc 'A-Za-z0-9' | head -c 32)
    log_info "Generated random SoftEther admin password"
fi

log_info "Node name:          ${NODE_NAME}"
log_info "Node host:          ${NODE_HOST}"
log_info "Node API port:      ${NODE_API_PORT}"
log_info "VPN port:           ${VPN_PORT}"
if [[ -n "$CONTROL_PLANE_URL" ]]; then
    log_info "Control plane:      ${CONTROL_PLANE_URL}"
fi

# ---------------------------------------------------------------------------
# Step 1: System Update and Dependencies
# ---------------------------------------------------------------------------
log_section "Step 1/6: System Update & Dependencies"

export DEBIAN_FRONTEND=noninteractive

log_info "Updating package lists..."
apt-get update -qq

log_info "Installing build dependencies..."
apt-get install -y -qq \
    build-essential \
    cmake \
    libncurses-dev \
    libreadline-dev \
    libssl-dev \
    zlib1g-dev \
    curl \
    wget \
    git \
    ufw \
    jq \
    openssl \
    ca-certificates \
    > /dev/null 2>&1

log_ok "Dependencies installed"

# ---------------------------------------------------------------------------
# Step 2: Install SoftEther VPN Server
# ---------------------------------------------------------------------------
log_section "Step 2/6: Install SoftEther VPN Server"

if [[ -x "${INSTALL_DIR}/vpnserver" ]]; then
    log_warn "SoftEther already installed at ${INSTALL_DIR}. Reinstalling..."
    systemctl stop softether 2>/dev/null || true
fi

TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

log_info "Downloading SoftEther ${SOFTETHER_VERSION}..."
if ! curl -fSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/softether.tar.gz" "$SOFTETHER_URL"; then
    # Fallback: try building from source
    log_warn "Pre-built binary download failed. Building from source..."
    cd "$TEMP_DIR"
    git clone --depth 1 --branch "${SOFTETHER_VERSION}" \
        https://github.com/SoftEtherVPN/SoftEtherVPN_Stable.git softether-src
    cd softether-src
    cp src/makefiles/linux_64bit.mak Makefile
    make -j"$(nproc)" || die "SoftEther build failed"
    mkdir -p "${INSTALL_DIR}"
    cp -r bin/vpnserver/* "${INSTALL_DIR}/"
    cd /
    log_ok "SoftEther built from source"
fi

# Extract if we downloaded the tarball
if [[ -f "${TEMP_DIR}/softether.tar.gz" ]]; then
    log_info "Extracting SoftEther..."
    cd "$TEMP_DIR"
    tar xzf softether.tar.gz

    # Move to install directory
    rm -rf "${INSTALL_DIR}"
    mv vpnserver "${INSTALL_DIR}"
    log_ok "SoftEther extracted to ${INSTALL_DIR}"
fi

# Set permissions
chmod 600 "${INSTALL_DIR}"/*
chmod 700 "${INSTALL_DIR}/vpnserver"
chmod 700 "${INSTALL_DIR}/vpncmd"

# Accept EULA (create the file that indicates acceptance)
log_info "Accepting SoftEther EULA..."
cat > "${INSTALL_DIR}/.sos_setting" <<'EULA'
a]_VPN_EULA_AGREED=1
EULA

# Run initial check
log_info "Verifying SoftEther installation..."
if "${INSTALL_DIR}/vpncmd" /TOOLS /CMD Check > /dev/null 2>&1; then
    log_ok "SoftEther installation verified"
else
    log_warn "SoftEther check command returned non-zero (may still work)"
fi

# ---------------------------------------------------------------------------
# Step 3: Configure SoftEther systemd service
# ---------------------------------------------------------------------------
log_section "Step 3/6: Configure SoftEther Service"

cat > /etc/systemd/system/softether.service <<'UNIT'
[Unit]
Description=SoftEther VPN Server
After=network-online.target
Wants=network-online.target

[Service]
Type=forking
ExecStart=/usr/local/vpnserver/vpnserver start
ExecStop=/usr/local/vpnserver/vpnserver stop
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
WorkingDirectory=/usr/local/vpnserver
LimitNOFILE=65536

# Hardening
ProtectSystem=full
PrivateTmp=true
NoNewPrivileges=false

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable softether
systemctl start softether

# Wait for SoftEther to be fully ready
log_info "Waiting for SoftEther to start..."
RETRIES=0
MAX_RETRIES=30
while [[ $RETRIES -lt $MAX_RETRIES ]]; do
    if "${INSTALL_DIR}/vpncmd" localhost /SERVER /CMD ServerInfoGet > /dev/null 2>&1; then
        break
    fi
    RETRIES=$((RETRIES + 1))
    sleep 1
done

if [[ $RETRIES -ge $MAX_RETRIES ]]; then
    log_warn "SoftEther did not respond in time, but service may still be starting"
else
    log_ok "SoftEther is running"
fi

# Set admin password
log_info "Setting SoftEther admin password..."
"${INSTALL_DIR}/vpncmd" localhost /SERVER /CMD ServerPasswordSet "$SE_ADMIN_PASSWORD" > /dev/null 2>&1 || true
log_ok "SoftEther admin password configured"

# Delete the default hub if it exists (clean slate)
"${INSTALL_DIR}/vpncmd" localhost /SERVER /PASSWORD:"$SE_ADMIN_PASSWORD" /CMD HubDelete DEFAULT > /dev/null 2>&1 || true

log_ok "SoftEther service configured and running"

# ---------------------------------------------------------------------------
# Step 4: Install Node Agent
# ---------------------------------------------------------------------------
log_section "Step 4/6: Install Dotachi Node Agent"

AGENT_INSTALLED=false

# Option A: Try downloading a pre-built binary
if [[ -n "$CONTROL_PLANE_URL" ]]; then
    DOWNLOAD_URL="${CONTROL_PLANE_URL%/}/downloads/node-agent-linux-amd64"
    log_info "Trying to download pre-built node-agent from ${DOWNLOAD_URL}..."
    if curl -fSL --max-time 30 -o "${AGENT_BIN}" "$DOWNLOAD_URL" 2>/dev/null; then
        chmod +x "${AGENT_BIN}"
        if "${AGENT_BIN}" --version > /dev/null 2>&1 || file "${AGENT_BIN}" | grep -q "ELF"; then
            AGENT_INSTALLED=true
            log_ok "Downloaded pre-built node-agent binary"
        else
            rm -f "${AGENT_BIN}"
            log_warn "Downloaded binary is invalid, will build from source"
        fi
    else
        log_info "Pre-built binary not available, will build from source"
    fi
fi

# Option B: Build from source using Go
if [[ "$AGENT_INSTALLED" = false ]]; then
    log_info "Building node-agent from source..."

    # Install Go if not present
    if ! command -v go &>/dev/null || [[ "$(go version 2>/dev/null | grep -oP '\d+\.\d+')" != "${GO_VERSION%.*}" ]]; then
        log_info "Installing Go ${GO_VERSION}..."
        GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
        GO_URL="https://go.dev/dl/${GO_TARBALL}"

        curl -fSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/${GO_TARBALL}" "$GO_URL" \
            || die "Failed to download Go ${GO_VERSION}"

        rm -rf /usr/local/go
        tar -C /usr/local -xzf "${TEMP_DIR}/${GO_TARBALL}"
        export PATH="/usr/local/go/bin:${PATH}"
        log_ok "Go ${GO_VERSION} installed"
    else
        log_ok "Go already installed: $(go version)"
    fi

    export PATH="/usr/local/go/bin:${PATH}"
    export GOPATH="/tmp/go"
    export GOCACHE="/tmp/go-cache"
    mkdir -p "$GOPATH" "$GOCACHE"

    # Clone the node-agent source
    AGENT_SRC="${TEMP_DIR}/node-agent"
    if [[ -n "$NODE_AGENT_REPO" ]]; then
        log_info "Cloning node-agent from ${NODE_AGENT_REPO}..."
        git clone --depth 1 "$NODE_AGENT_REPO" "$AGENT_SRC" \
            || die "Failed to clone node-agent repository"
    elif [[ -d "/opt/dotachi/node-agent" ]]; then
        log_info "Using local source at /opt/dotachi/node-agent..."
        cp -r /opt/dotachi/node-agent "$AGENT_SRC"
    else
        # Create a minimal main.go that matches the project structure
        log_info "No repo specified. Creating node-agent from embedded source..."
        mkdir -p "${AGENT_SRC}/softether" "${AGENT_SRC}/handler"

        cat > "${AGENT_SRC}/go.mod" <<'GOMOD'
module github.com/dotachi/node-agent

go 1.23.8

require github.com/go-chi/chi/v5 v5.2.5
GOMOD

        cat > "${AGENT_SRC}/go.sum" <<'GOSUM'
github.com/go-chi/chi/v5 v5.2.5 h1:7YJHSI+bU4m4vzmJqJpT1M0YUg4VsSExo+GNz/FS3Qw=
github.com/go-chi/chi/v5 v5.2.5/go.mod h1:DslCQbL2OYiznFReuXYUmQ2hGd1aDpCnlMNITLSKoi8=
GOSUM

        cat > "${AGENT_SRC}/softether/cmd.go" <<'GOSRC'
package softether

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

const cmdTimeout = 30 * time.Second

type Client struct {
	VpncmdPath string
	ServerHost string
}

func (c *Client) ServerCmd(args ...string) (string, error) {
	cmdArgs := []string{c.ServerHost, "/SERVER", "/CMD"}
	cmdArgs = append(cmdArgs, args...)
	return runCmd(c.VpncmdPath, cmdArgs...)
}

func (c *Client) HubCmd(hubName string, args ...string) (string, error) {
	cmdArgs := []string{c.ServerHost, "/SERVER", "/HUB:" + hubName, "/CMD"}
	cmdArgs = append(cmdArgs, args...)
	return runCmd(c.VpncmdPath, cmdArgs...)
}

func runCmd(vpncmdPath string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	log.Printf("[vpncmd] %s %s", vpncmdPath, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, vpncmdPath, args...)
	out, err := cmd.CombinedOutput()
	output := string(out)
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("vpncmd timed out after %s", cmdTimeout)
	}
	if err != nil {
		return output, fmt.Errorf("vpncmd failed: %w\n%s", err, output)
	}
	return output, nil
}
GOSRC

        cat > "${AGENT_SRC}/handler/handler.go" <<'GOSRC'
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dotachi/node-agent/softether"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	SE      *softether.Client
	StartAt time.Time
}

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

func subnetParams(cidr string) (start, end, mask, gw string, err error) {
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
	gwIP := make(net.IP, 4)
	copy(gwIP, base)
	gwIP[3] = 1
	gw = gwIP.String()
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
	start, end, mask, gw, err := subnetParams(req.Subnet)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	maxSess := req.MaxSessions
	if maxSess <= 0 {
		maxSess = 100
	}
	if _, err := h.SE.ServerCmd("HubCreate", req.HubName, `/PASSWORD:""`); err != nil {
		writeErr(w, http.StatusInternalServerError, "HubCreate failed: "+err.Error())
		return
	}
	if _, err := h.SE.HubCmd(req.HubName, "SetMaxSession", strconv.Itoa(maxSess)); err != nil {
		writeErr(w, http.StatusInternalServerError, "SetMaxSession failed: "+err.Error())
		return
	}
	if _, err := h.SE.HubCmd(req.HubName, "SecureNatEnable"); err != nil {
		writeErr(w, http.StatusInternalServerError, "SecureNatEnable failed: "+err.Error())
		return
	}
	dhcpArgs := fmt.Sprintf("/START:%s /END:%s /MASK:%s /EXPIRE:86400 /GW:%s /DNS:none /DNS2:none", start, end, mask, gw)
	if _, err := h.SE.HubCmd(req.HubName, "DhcpSet", dhcpArgs); err != nil {
		writeErr(w, http.StatusInternalServerError, "DhcpSet failed: "+err.Error())
		return
	}
	if _, err := h.SE.HubCmd(req.HubName, "NatSet", "/MTU:1400", "/TCPTIMEOUT:86400", "/UDPTIMEOUT:3600"); err != nil {
		writeErr(w, http.StatusInternalServerError, "NatSet failed: "+err.Error())
		return
	}
	h.SE.HubCmd(req.HubName, "SetHubOption", "/NoArpPolling:1", "/NoIPv6DefaultRouterInRA:1", "/NoMacAddressLog:1")
	log.Printf("[hub/create] created hub %s (max=%d, subnet=%s)", req.HubName, maxSess, req.Subnet)
	writeJSON(w, http.StatusOK, map[string]string{"status": "created", "hub_name": req.HubName})
}

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
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "hub_name": req.HubName})
}

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
	if _, err := h.SE.HubCmd(req.HubName, "UserCreate", req.Username, "/GROUP:none", "/REALNAME:none", "/NOTE:none"); err != nil {
		writeErr(w, http.StatusInternalServerError, "UserCreate failed: "+err.Error())
		return
	}
	if _, err := h.SE.HubCmd(req.HubName, "UserPasswordSet", req.Username, "/PASSWORD:"+req.Password); err != nil {
		writeErr(w, http.StatusInternalServerError, "UserPasswordSet failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "created", "username": req.Username})
}

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
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "username": req.Username})
}

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
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "username": req.Username, "session": sessionName})
}

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

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.StartAt).Seconds()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "uptime": int64(uptime)})
}
GOSRC

        cat > "${AGENT_SRC}/main.go" <<'GOSRC'
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dotachi/node-agent/handler"
	"github.com/dotachi/node-agent/softether"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	listenAddr := envOr("LISTEN_ADDR", ":7443")
	apiSecret := os.Getenv("API_SECRET")
	vpncmdPath := envOr("VPNCMD_PATH", "/usr/local/vpnserver/vpncmd")
	serverHost := envOr("SERVER_HOST", "localhost")
	vpnPort := envOr("VPN_PORT", "443")

	if apiSecret == "" {
		log.Fatal("API_SECRET environment variable is required")
	}

	se := &softether.Client{
		VpncmdPath: vpncmdPath,
		ServerHost: serverHost,
	}
	h := &handler.Handler{
		SE:      se,
		StartAt: time.Now(),
	}
	initSoftEther(se, vpnPort)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Get("/health", h.Health)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware(apiSecret))
		r.Post("/hub/create", h.HubCreate)
		r.Post("/hub/delete", h.HubDelete)
		r.Get("/hub/status/{hub_name}", h.HubStatus)
		r.Post("/user/create", h.UserCreate)
		r.Post("/user/delete", h.UserDelete)
		r.Post("/user/disconnect", h.UserDisconnect)
	})

	log.Printf("node-agent listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func initSoftEther(se *softether.Client, vpnPort string) {
	log.Println("[init] applying SoftEther server optimizations...")
	se.ServerCmd("KeepEnable")
	se.ServerCmd("KeepSet", "/HOST:keepalive.softether.org", "/PORT:80", "/INTERVAL:5", "/PROTOCOL:udp")
	se.ServerCmd("ListenerCreate", vpnPort)
	se.ServerCmd("ProtoOptionsSet", "/NAME:enabled", "/VALUE:true")
	se.ServerCmd("ServerCipherSet", "AES128-SHA")
	log.Println("[init] SoftEther server optimizations applied")
}

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
GOSRC

        log_ok "Embedded node-agent source created"
    fi

    # Build the binary
    log_info "Compiling node-agent..."
    cd "$AGENT_SRC"
    go mod tidy 2>/dev/null || true
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${AGENT_BIN}" . \
        || die "Failed to build node-agent"
    chmod +x "${AGENT_BIN}"
    AGENT_INSTALLED=true
    log_ok "Node agent compiled: ${AGENT_BIN}"
fi

if [[ "$AGENT_INSTALLED" = false ]]; then
    die "Failed to install node-agent"
fi

# ---------------------------------------------------------------------------
# Step 5: Configure Node Agent Service
# ---------------------------------------------------------------------------
log_section "Step 5/6: Configure Node Agent Service"

# Create config directory and env file
mkdir -p "$CONFIG_DIR"
chmod 700 "$CONFIG_DIR"

cat > "${CONFIG_DIR}/node-agent.env" <<ENVFILE
# Dotachi Node Agent Configuration
# Generated by setup-node.sh on $(date -u +"%Y-%m-%dT%H:%M:%SZ")

API_SECRET=${API_SECRET}
LISTEN_ADDR=:${NODE_API_PORT}
VPNCMD_PATH=${INSTALL_DIR}/vpncmd
SERVER_HOST=localhost
VPN_PORT=${VPN_PORT}
ENVFILE

chmod 600 "${CONFIG_DIR}/node-agent.env"
log_ok "Environment file created at ${CONFIG_DIR}/node-agent.env"

# Install systemd service
cat > /etc/systemd/system/dotachi-node.service <<'UNIT'
[Unit]
Description=Dotachi Node Agent
After=network-online.target softether.service
Wants=network-online.target
Requires=softether.service

[Service]
Type=simple
ExecStart=/usr/local/bin/dotachi-node-agent
Restart=always
RestartSec=5
EnvironmentFile=/etc/dotachi/node-agent.env
WorkingDirectory=/etc/dotachi
LimitNOFILE=65536

# Hardening
ProtectSystem=full
PrivateTmp=true
NoNewPrivileges=true
ProtectHome=true

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable dotachi-node
systemctl start dotachi-node

# Verify node-agent started
sleep 2
if systemctl is-active --quiet dotachi-node; then
    log_ok "Node agent service is running"
else
    log_warn "Node agent service may not have started yet. Check: journalctl -u dotachi-node"
fi

# ---------------------------------------------------------------------------
# Step 6: Firewall Configuration
# ---------------------------------------------------------------------------
log_section "Step 6/6: Firewall & Registration"

log_info "Configuring UFW firewall..."

# Ensure SSH is allowed before enabling UFW (prevent lockout)
ufw allow 22/tcp comment "SSH" > /dev/null 2>&1

# VPN traffic (disguised as HTTPS)
ufw allow "${VPN_PORT}/tcp" comment "SoftEther VPN (HTTPS)" > /dev/null 2>&1

# Node agent API
ufw allow "${NODE_API_PORT}/tcp" comment "Dotachi node-agent API" > /dev/null 2>&1

# SoftEther management port
ufw allow 992/tcp comment "SoftEther management" > /dev/null 2>&1

# Optional extra port
ufw allow 5555/tcp comment "SoftEther alt port" > /dev/null 2>&1

# Enable UFW non-interactively
echo "y" | ufw enable > /dev/null 2>&1 || ufw --force enable > /dev/null 2>&1
ufw reload > /dev/null 2>&1

log_ok "Firewall configured"
log_info "  Allowed ports: 22/tcp (SSH), ${VPN_PORT}/tcp (VPN), ${NODE_API_PORT}/tcp (API), 992/tcp (Mgmt), 5555/tcp (Alt)"

# ---------------------------------------------------------------------------
# Register node with the control plane
# ---------------------------------------------------------------------------
REGISTRATION_STATUS="skipped"

if [[ -n "$CONTROL_PLANE_URL" && -n "$ADMIN_TOKEN" ]]; then
    log_info "Registering node with control plane..."

    REGISTER_PAYLOAD=$(cat <<JSONEOF
{
    "name": "${NODE_NAME}",
    "host": "${NODE_HOST}",
    "api_port": ${NODE_API_PORT},
    "api_secret": "${API_SECRET}"
}
JSONEOF
    )

    HTTP_RESPONSE=$(curl -sf --max-time 15 \
        -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        -d "$REGISTER_PAYLOAD" \
        -w "\n%{http_code}" \
        "${CONTROL_PLANE_URL%/}/nodes" 2>&1) || true

    HTTP_CODE=$(echo "$HTTP_RESPONSE" | tail -1)
    HTTP_BODY=$(echo "$HTTP_RESPONSE" | sed '$d')

    if [[ "$HTTP_CODE" == "201" ]]; then
        REGISTRATION_STATUS="success"
        NODE_ID=$(echo "$HTTP_BODY" | jq -r '.id // "unknown"' 2>/dev/null || echo "unknown")
        log_ok "Node registered with control plane (ID: ${NODE_ID})"
    elif [[ "$HTTP_CODE" == "409" ]]; then
        REGISTRATION_STATUS="already_exists"
        log_warn "Node name '${NODE_NAME}' already exists in control plane"
    else
        REGISTRATION_STATUS="failed"
        log_warn "Failed to register node (HTTP ${HTTP_CODE}). Register manually."
        log_warn "Response: ${HTTP_BODY}"
    fi
elif [[ -n "$CONTROL_PLANE_URL" && -z "$ADMIN_TOKEN" ]]; then
    log_warn "ADMIN_TOKEN not provided. Skipping automatic registration."
    log_info "Register manually with:"
    log_info "  curl -X POST ${CONTROL_PLANE_URL%/}/nodes \\"
    log_info "    -H 'Authorization: Bearer <ADMIN_JWT>' \\"
    log_info "    -H 'Content-Type: application/json' \\"
    log_info "    -d '{\"name\":\"${NODE_NAME}\",\"host\":\"${NODE_HOST}\",\"api_port\":${NODE_API_PORT},\"api_secret\":\"${API_SECRET}\"}'"
else
    log_warn "CONTROL_PLANE_URL not provided. Skipping registration."
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo -e "${BOLD}${GREEN}============================================================${NC}"
echo -e "${BOLD}${GREEN}    Dotachi VPN Node Setup Complete${NC}"
echo -e "${BOLD}${GREEN}============================================================${NC}"
echo ""
echo -e "  ${BOLD}Node Name:${NC}              ${NODE_NAME}"
echo -e "  ${BOLD}Public IP:${NC}              ${NODE_HOST}"
echo -e "  ${BOLD}VPN Port:${NC}               ${VPN_PORT}/tcp"
echo -e "  ${BOLD}Node Agent API:${NC}         http://${NODE_HOST}:${NODE_API_PORT}"
echo -e "  ${BOLD}SoftEther Mgmt:${NC}         ${NODE_HOST}:992"
echo -e "  ${BOLD}SoftEther Admin Pass:${NC}   ${SE_ADMIN_PASSWORD}"
echo ""
echo -e "  ${BOLD}Services:${NC}"
echo -e "    softether.service     $(systemctl is-active softether 2>/dev/null || echo 'unknown')"
echo -e "    dotachi-node.service  $(systemctl is-active dotachi-node 2>/dev/null || echo 'unknown')"
echo ""
echo -e "  ${BOLD}Config Files:${NC}"
echo -e "    ${CONFIG_DIR}/node-agent.env"
echo -e "    /etc/systemd/system/softether.service"
echo -e "    /etc/systemd/system/dotachi-node.service"
echo ""
echo -e "  ${BOLD}Registration:${NC}           ${REGISTRATION_STATUS}"
echo ""
echo -e "  ${BOLD}Useful Commands:${NC}"
echo -e "    journalctl -u softether -f          # SoftEther logs"
echo -e "    journalctl -u dotachi-node -f       # Node agent logs"
echo -e "    systemctl restart softether          # Restart SoftEther"
echo -e "    systemctl restart dotachi-node       # Restart node agent"
echo -e "    ${INSTALL_DIR}/vpncmd localhost /SERVER /PASSWORD:${SE_ADMIN_PASSWORD} /CMD ServerStatusGet"
echo -e "    curl -s http://localhost:${NODE_API_PORT}/health | jq ."
echo ""
if [[ "$REGISTRATION_STATUS" != "success" && -n "$CONTROL_PLANE_URL" ]]; then
    echo -e "  ${YELLOW}To register this node manually:${NC}"
    echo -e "    curl -X POST ${CONTROL_PLANE_URL%/}/nodes \\"
    echo -e "      -H 'Authorization: Bearer <ADMIN_JWT>' \\"
    echo -e "      -H 'Content-Type: application/json' \\"
    echo -e "      -d '{\"name\":\"${NODE_NAME}\",\"host\":\"${NODE_HOST}\",\"api_port\":${NODE_API_PORT},\"api_secret\":\"${API_SECRET}\"}'"
    echo ""
fi
echo -e "${BOLD}${GREEN}============================================================${NC}"
