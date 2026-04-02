# Client

Windows desktop application for players. Built with Wails (Go backend + React frontend).

## Stack

- **Go** backend with Wails v2 bindings
- **React 18** + Vite frontend
- **SoftEther VPN Client** for tunnel management

## Building

Requires [Wails](https://wails.io/) and [Node.js](https://nodejs.org/).

```bash
# Install dependencies
cd frontend && npm install && cd ..

# Development mode
wails dev

# Production build
wails build

# Build Windows installer (requires NSIS)
cd installer && makensis dotachi.nsi
```

## Features

### VPN Connection
- **8 simultaneous TCP streams** on port 443 for connection resilience
- **Auto-reconnect** with exponential backoff (1-10 seconds)
- **Split tunneling** - only game subnet routes through VPN
- **Health monitoring** - checks connection every 3 seconds
- **UDP acceleration** - lower latency when available

### Room Management
- Browse, search, and filter rooms by game
- Join public/private rooms
- Purchase rooms with shards
- Room chat (polling-based)
- Invite links (`dotachi://join/TOKEN` deep links)

### Player Features
- Real-time ping measurement (TCP to port 443)
- Connection quality indicator (excellent/good/fair/poor)
- Local VPN IP display (click to copy)
- Game-specific LAN setup guides
- Shard balance and transaction tracking

### Device Fingerprint
Collects hardware identifiers (BIOS serial, CPU ID, disk serial, Windows Machine GUID) to enforce one account per device. Uses SHA-256 hash with fallback chain if any identifier is unavailable.

## Pages

| Page | Description |
|---|---|
| Login | Phone + password auth with referral code |
| Rooms | Browse/search/filter rooms, purchase modal |
| Room | Active room with VPN, chat, members, management |
| Shop | Pricing info and shard purchase contact |
| Profile | Stats, shard balance, promo codes, referrals |
| Settings | Server URL, SoftEther status, VPN preferences |
| Game Guides | LAN setup instructions per game with console commands |

## Configuration

Settings are persisted to `%APPDATA%/Dotachi/settings.json`:
- `api_base` - Control plane URL (default: `http://127.0.0.1:8080`)
