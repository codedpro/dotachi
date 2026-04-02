# Node Agent

Lightweight agent that runs on each VPN server alongside SoftEther. Receives commands from the control plane and manages VPN hubs and users.

## Stack

- **Go** single binary (~9 MB)
- **SoftEther VPN Server** (managed via `vpncmd`)
- **Chi** HTTP router

## Running

```bash
export API_SECRET="shared-secret-with-control-plane"
go build -o dotachi-node .
./dotachi-node
```

## Configuration

| Env Var | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:7443` | HTTP listen address |
| `API_SECRET` | *required* | Shared secret for control plane auth |
| `VPNCMD_PATH` | `/usr/local/vpnserver/vpncmd` | Path to SoftEther vpncmd |
| `SERVER_HOST` | `localhost` | SoftEther server address |
| `VPN_PORT` | `443` | VPN listener port (HTTPS disguise) |
| `CONTROL_PLANE_URL` | *(optional)* | If set, enables heartbeat reporting |
| `NODE_NAME` | hostname | Node identifier for heartbeat |

## Endpoints

All endpoints except `/health` require `X-Api-Secret` header.

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check (public) |
| `POST` | `/ping` | Health check (authenticated) |
| `POST` | `/hub/create` | Create VPN hub with DHCP |
| `POST` | `/hub/delete` | Delete VPN hub |
| `GET` | `/hub/status/{hub_name}` | Hub session/traffic stats |
| `GET` | `/hub/user-traffic/{hub_name}/{username}` | Per-user traffic stats |
| `POST` | `/user/create` | Create VPN user in hub |
| `POST` | `/user/delete` | Delete VPN user |
| `POST` | `/user/disconnect` | Force-disconnect session |
| `GET` | `/stats` | Node CPU, memory, hub count |

## SoftEther Optimizations

On startup, the agent applies server-level optimizations:

- **Port 443 listener** - VPN traffic disguised as HTTPS
- **Keepalive** - 5-second UDP keepalive to detect dead connections
- **AES128 cipher** - Lower CPU usage for resource-constrained servers
- **MTU 1400** - Prevents packet fragmentation over VPN
- **24h DHCP lease** - IPs never change mid-game
- **No DNS routing** - Players use their own DNS, only game traffic in tunnel

## Heartbeat

If `CONTROL_PLANE_URL` is configured, the agent sends a heartbeat every 30 seconds with:
- Active hub list
- Session counts
- CPU and memory usage

The control plane responds with expected hub list. The agent reconciles:
- Missing hubs are recreated (recovery after reboot)
- Orphan hubs are deleted (cleanup)
