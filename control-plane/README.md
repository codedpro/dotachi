# Control Plane

Central REST API server for the Dotachi platform. Manages users, nodes, rooms, billing, and monitoring.

## Stack

- **Go** with Chi router
- **PostgreSQL** (auto-migrating schema)
- **JWT** authentication (72h tokens)
- **bcrypt** password hashing

## Running

```bash
# With Docker
docker compose up -d

# Standalone
export DATABASE_URL="postgres://dotachi:dotachi@localhost:5432/dotachi?sslmode=disable"
export JWT_SECRET="your-secret"
go build -o dotachi-cp .
./dotachi-cp
```

## Configuration

| Env Var | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `DATABASE_URL` | `postgres://dotachi:dotachi@localhost:5432/dotachi?sslmode=disable` | PostgreSQL connection |
| `JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `ADMIN_PHONE` | `09000000000` | Default admin phone |
| `ADMIN_PASSWORD` | `admin123` | Default admin password |
| `ROOM_IDLE_TIMEOUT` | `30m` | Empty room cleanup timeout |

## API Endpoints

### Public
- `POST /auth/register` - Register (phone + password + display name)
- `POST /auth/login` - Login
- `GET /rooms/pricing` - Pricing table
- `GET /rooms/shop` - Shop info and contact

### Authenticated
- `GET /auth/me` - User profile
- `GET /auth/me/stats` - Play stats
- `PATCH /auth/me/display-name` - Update display name
- `POST /auth/change-password` - Change password
- `GET /auth/me/referral` - Referral info
- `GET /rooms` - List rooms (filters: q, game, is_private, has_slots, node_id)
- `GET /rooms/{id}` - Room details
- `POST /rooms/{id}/join` - Join room
- `POST /rooms/{id}/leave` - Leave room
- `POST /rooms/purchase` - Buy a room with shards
- `POST /rooms/{id}/extend` - Extend room expiry
- `POST /rooms/{id}/invite` - Create invite link
- `POST /rooms/join-invite` - Join via invite token
- `POST /rooms/{id}/kick` - Kick player (owner/admin)
- `POST /rooms/{id}/ban` - Ban player
- `POST /rooms/{id}/unban` - Unban player
- `POST /rooms/{id}/set-role` - Set member role
- `POST /rooms/{id}/transfer` - Transfer ownership
- `GET /rooms/{id}/members` - List members
- `GET /rooms/{id}/messages` - Chat messages
- `POST /rooms/{id}/messages` - Send chat message
- `GET /rooms/favorites` - User's favorites
- `POST /rooms/{id}/favorite` - Add favorite
- `DELETE /rooms/{id}/favorite` - Remove favorite
- `POST /promo/redeem` - Redeem promo code

### Admin Only
- `POST /nodes` - Register node
- `GET /nodes` - List nodes
- `POST /nodes/{id}/ping` - Ping node
- `POST /admin/rooms` - Create room
- `POST /admin/rooms/{id}/assign-owner` - Assign owner
- `GET /admin/users` - List users
- `POST /admin/users/{id}/add-shards` - Add shards
- `POST /admin/users/{id}/remove-shards` - Remove shards
- `POST /admin/users/{id}/reset-device` - Reset device fingerprint
- `POST /admin/users/{id}/delete` - Delete user
- `POST /admin/promo/create` - Create promo code
- `GET /admin/promo/list` - List promo codes
- `GET /admin/monitor/overview` - System stats
- `GET /admin/monitor/nodes` - Node status
- `GET /admin/monitor/room/{id}` - Room detail

### Internal
- `POST /internal/heartbeat` - Node agent heartbeat

## Background Workers

- **Cleanup** (5 min): Deactivates empty rooms after idle timeout, purges old inactive rooms, cleans expired rooms
- **Health Checker** (60s): Marks nodes offline if heartbeat missed for 2+ minutes
- **Shared Billing** (60s): Charges shards for time in shared rooms

## Database Schema

10 tables: `users`, `nodes`, `rooms`, `room_members`, `room_bans`, `favorites`, `play_sessions`, `room_roles`, `shard_transactions`, `shared_sessions`, plus `promo_codes`, `promo_redemptions`, `room_invites`, `room_messages`.
