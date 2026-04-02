# Admin Panel

Lightweight web dashboard for platform operators. Pure vanilla HTML/CSS/JS - no build step required.

## Running

```bash
# With Docker
docker compose up -d
# Panel available at http://localhost:8081

# Or serve directly
python3 -m http.server 8081 -d admin-panel/
# Or: npx serve admin-panel/
```

## Features

### Nodes Tab
- List all VPN nodes with health status
- Add new nodes
- Ping nodes to check connectivity
- View CPU, memory, and session counts (from heartbeat)

### Rooms Tab
- List all rooms with filters (search, node, status)
- Create rooms (standard or shared)
- Assign room owners
- View expiry countdown and room type

### Users Tab
- List all users with shard balances
- Add/remove shards
- View device fingerprints
- Delete users

### Live Monitor
- Active rooms with player counts
- Total shards in circulation
- Auto-refresh every 10 seconds

## Deployment

Served via nginx in Docker. No build step - just static files.

```dockerfile
FROM nginx:1.27-alpine
COPY . /usr/share/nginx/html/
```
