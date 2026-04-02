# Deploy

One-command setup script for provisioning new VPN nodes on Ubuntu VPS.

## Usage

```bash
# Required
export API_SECRET="shared-secret-with-control-plane"

# Optional
export CONTROL_PLANE_URL="http://your-control-plane:8080"
export NODE_NAME="my-node"
export NODE_HOST="1.2.3.4"  # auto-detected if omitted
export VPN_PORT="443"

# Run
bash setup-node.sh
```

## What It Does

1. Installs system dependencies
2. Downloads and installs SoftEther VPN Server
3. Configures SoftEther as a systemd service
4. Builds or downloads the node-agent binary
5. Configures node-agent as a systemd service
6. Opens firewall ports (443, 7443, 992)
7. Optionally registers the node with the control plane

## Files

| File | Description |
|---|---|
| `setup-node.sh` | Main setup script |
| `softether.service` | systemd unit for SoftEther VPN Server |
| `dotachi-node.service` | systemd unit for node-agent |

## Firewall Ports

| Port | Protocol | Purpose |
|---|---|---|
| 443 | TCP | VPN connections (HTTPS disguise) |
| 7443 | TCP | Node agent API |
| 992 | TCP | SoftEther management |
