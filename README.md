# Dotachi - LAN Gaming Network Platform

A self-hosted platform that lets players create and join virtual LAN rooms for multiplayer gaming. Built for regions with restrictive or unstable networks, Dotachi uses SoftEther VPN to create rock-solid L2 bridges that make LAN games work seamlessly over the internet.

## Architecture

```
                    +------------------+
                    |   Control Plane  |
                    |   (REST API)     |
                    +--------+---------+
                             |
              +--------------+--------------+
              |              |              |
        +-----+-----+  +----+----+   +-----+-----+
        | Node Agent |  |  Node   |   |   Node    |
        | (VPS #1)   |  | (VPS #2)|   |  (VPS #3) |
        | SoftEther  |  | SoftEther|   | SoftEther |
        +-----+------+  +----+----+   +-----+-----+
              |              |              |
         Players         Players        Players
         connect         connect        connect
```

### Components

| Component | Description | Tech |
|---|---|---|
| [Control Plane](control-plane/) | Central API server - manages users, rooms, nodes, billing | Go + PostgreSQL + Chi |
| [Node Agent](node-agent/) | Runs on each VPN server alongside SoftEther | Go (single binary, ~9 MB) |
| [Client](client/) | Windows desktop app for players | Go (Wails) + React |
| [Admin Panel](admin-panel/) | Web dashboard for platform operators | Vanilla HTML/CSS/JS |
| [CLI](cli/) | Command-line admin tool | Go + Cobra |
| [Deploy](deploy/) | One-command node setup script | Bash + systemd |

## Key Features

- **Virtual LAN rooms** with isolated L2 networks per room
- **Multi-TCP VPN tunneling** (8 simultaneous TCP streams on port 443) for connection resilience
- **Auto-reconnect** with exponential backoff - players never notice brief network hiccups
- **Shard-based billing** - users purchase shards to buy private rooms or join shared LAN
- **Room management** - owners can kick, ban, set admins, transfer ownership
- **Node heartbeat** - control plane monitors all VPN nodes, auto-recovers after reboots
- **Split tunneling** - only game traffic routes through VPN, internet stays on normal connection
- **Device fingerprinting** - one account per device enforcement

## Quick Start

### Development (Docker)

```bash
docker compose up -d
```

This starts PostgreSQL + Control Plane + Node Agent + Admin Panel.

- Control Plane API: http://localhost:8080
- Admin Panel: http://localhost:8081
- Default admin: phone `09000000000`, password `admin123`

### Production Node Deployment

```bash
# On a fresh Ubuntu VPS:
export API_SECRET="your-secret-here"
export CONTROL_PLANE_URL="http://your-control-plane:8080"
bash deploy/setup-node.sh
```

## Business Model

- **Private Rooms**: Users purchase rooms with shards (virtual currency). Price = slots x 1,000 shards/day. Discounts for longer durations.
- **Shared LAN**: Public rooms charged at 2,000 shards/hour while connected.
- **Shard Purchase**: Admin manages shard balances via API or admin panel.

| Duration | Discount |
|---|---|
| 7 days (minimum) | No discount |
| 1 month | 10% off |
| 3 months | 25% off |
| 1 year | 40% off |

## Supported Games

Dotachi works with any game that supports LAN multiplayer:

- Dota 2
- Counter-Strike 2
- Warcraft III
- Age of Empires II
- Minecraft
- Valorant (Custom Games)
- Any game with LAN/local network support

## Tech Stack

- **Language**: Go (all backend + client backend)
- **Frontend**: React (client UI), Vanilla JS (admin panel)
- **Database**: PostgreSQL
- **VPN**: SoftEther VPN Server
- **Desktop**: Wails v2 (Go + Web frontend)
- **Deployment**: Docker, systemd

## License

MIT
