package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(url string) error {
	var err error
	DB, err = sql.Open("postgres", url)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	return migrate()
}

func migrate() error {
	_, err := DB.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            SERIAL PRIMARY KEY,
	phone         TEXT    NOT NULL UNIQUE,
	password_hash TEXT    NOT NULL,
	display_name  TEXT    NOT NULL,
	is_admin      BOOLEAN NOT NULL DEFAULT FALSE,
	created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS nodes (
	id         SERIAL PRIMARY KEY,
	name       TEXT    NOT NULL UNIQUE,
	host       TEXT    NOT NULL,
	api_port   INTEGER NOT NULL DEFAULT 7443,
	api_secret TEXT    NOT NULL,
	is_active  BOOLEAN NOT NULL DEFAULT TRUE,
	max_rooms  INTEGER NOT NULL DEFAULT 50,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rooms (
	id            SERIAL PRIMARY KEY,
	node_id       INTEGER NOT NULL REFERENCES nodes(id),
	owner_id      INTEGER REFERENCES users(id),
	name          TEXT    NOT NULL,
	hub_name      TEXT    NOT NULL UNIQUE,
	is_private    BOOLEAN NOT NULL DEFAULT FALSE,
	password_hash TEXT,
	max_players   INTEGER NOT NULL DEFAULT 10,
	subnet        TEXT    NOT NULL,
	is_active      BOOLEAN NOT NULL DEFAULT TRUE,
	last_activity  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS room_members (
	id           SERIAL PRIMARY KEY,
	room_id      INTEGER NOT NULL REFERENCES rooms(id),
	user_id      INTEGER NOT NULL REFERENCES users(id),
	vpn_username TEXT    NOT NULL,
	vpn_password TEXT    NOT NULL,
	joined_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(room_id, user_id)
);

CREATE TABLE IF NOT EXISTS room_bans (
	id         SERIAL PRIMARY KEY,
	room_id    INTEGER NOT NULL REFERENCES rooms(id),
	user_id    INTEGER NOT NULL REFERENCES users(id),
	banned_by  INTEGER NOT NULL REFERENCES users(id),
	reason     TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(room_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_rooms_active ON rooms(is_active);
CREATE INDEX IF NOT EXISTS idx_rooms_node ON rooms(node_id);
CREATE INDEX IF NOT EXISTS idx_members_room ON room_members(room_id);
CREATE INDEX IF NOT EXISTS idx_members_user ON room_members(user_id);
CREATE INDEX IF NOT EXISTS idx_bans_room ON room_bans(room_id);

-- Migration: add last_activity column to rooms if it does not exist
DO $$ BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'rooms' AND column_name = 'last_activity'
	) THEN
		ALTER TABLE rooms ADD COLUMN last_activity TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
	END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_rooms_last_activity ON rooms(last_activity);

-- Migration: user stats and shard balance
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_play_hours REAL NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_sessions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS shard_balance INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS daily_transfer_bytes BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS transfer_reset_date DATE NOT NULL DEFAULT CURRENT_DATE;

-- Migration: room game_tag, description, expiry, shared
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS game_tag TEXT NOT NULL DEFAULT 'other';
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS is_shared BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS hourly_cost INTEGER NOT NULL DEFAULT 0;

-- Favorites table
CREATE TABLE IF NOT EXISTS favorites (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    room_id    INTEGER NOT NULL REFERENCES rooms(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, room_id)
);

-- Play sessions table
CREATE TABLE IF NOT EXISTS play_sessions (
    id               SERIAL PRIMARY KEY,
    user_id          INTEGER NOT NULL REFERENCES users(id),
    room_id          INTEGER NOT NULL REFERENCES rooms(id),
    joined_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    left_at          TIMESTAMPTZ,
    duration_minutes INTEGER
);
CREATE INDEX IF NOT EXISTS idx_play_sessions_user ON play_sessions(user_id);

-- Room roles table (owner, admin, member)
CREATE TABLE IF NOT EXISTS room_roles (
    id         SERIAL PRIMARY KEY,
    room_id    INTEGER NOT NULL REFERENCES rooms(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    role       TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(room_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_room_roles_room ON room_roles(room_id);

-- Shard transactions table
CREATE TABLE IF NOT EXISTS shard_transactions (
    id            SERIAL PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES users(id),
    amount        INTEGER NOT NULL,
    balance_after INTEGER NOT NULL,
    tx_type       TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    ref_id        INTEGER,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_shard_tx_user ON shard_transactions(user_id);

-- Shared sessions table (tracks time in shared rooms for billing)
CREATE TABLE IF NOT EXISTS shared_sessions (
    id             SERIAL PRIMARY KEY,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    room_id        INTEGER NOT NULL REFERENCES rooms(id),
    started_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at       TIMESTAMPTZ,
    shards_charged INTEGER NOT NULL DEFAULT 0,
    UNIQUE(user_id, room_id, started_at)
);

-- Migration: add heartbeat columns to nodes
DO $$ BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'nodes' AND column_name = 'last_heartbeat'
	) THEN
		ALTER TABLE nodes ADD COLUMN last_heartbeat TIMESTAMPTZ;
	END IF;
END $$;

DO $$ BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'nodes' AND column_name = 'hub_count'
	) THEN
		ALTER TABLE nodes ADD COLUMN hub_count INTEGER NOT NULL DEFAULT 0;
	END IF;
END $$;

DO $$ BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'nodes' AND column_name = 'session_count'
	) THEN
		ALTER TABLE nodes ADD COLUMN session_count INTEGER NOT NULL DEFAULT 0;
	END IF;
END $$;

DO $$ BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'nodes' AND column_name = 'cpu_usage'
	) THEN
		ALTER TABLE nodes ADD COLUMN cpu_usage REAL NOT NULL DEFAULT 0;
	END IF;
END $$;

DO $$ BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'nodes' AND column_name = 'memory_mb'
	) THEN
		ALTER TABLE nodes ADD COLUMN memory_mb INTEGER NOT NULL DEFAULT 0;
	END IF;
END $$;

-- Promo codes
CREATE TABLE IF NOT EXISTS promo_codes (
    id          SERIAL PRIMARY KEY,
    code        TEXT NOT NULL UNIQUE,
    shard_amount INTEGER NOT NULL,
    max_uses    INTEGER NOT NULL DEFAULT 1,
    used_count  INTEGER NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS promo_redemptions (
    id         SERIAL PRIMARY KEY,
    code_id    INTEGER NOT NULL REFERENCES promo_codes(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(code_id, user_id)
);

-- Referral system
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code TEXT UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referred_by INTEGER REFERENCES users(id);

-- Room invites
CREATE TABLE IF NOT EXISTS room_invites (
    id         SERIAL PRIMARY KEY,
    room_id    INTEGER NOT NULL REFERENCES rooms(id),
    token      TEXT NOT NULL UNIQUE,
    created_by INTEGER NOT NULL REFERENCES users(id),
    max_uses   INTEGER NOT NULL DEFAULT 0,
    used_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Device fingerprint for 1-account-per-device enforcement
ALTER TABLE users ADD COLUMN IF NOT EXISTS device_fingerprint TEXT;
CREATE INDEX IF NOT EXISTS idx_users_fingerprint ON users(device_fingerprint);

-- Room chat messages
CREATE TABLE IF NOT EXISTS room_messages (
    id         SERIAL PRIMARY KEY,
    room_id    INTEGER NOT NULL REFERENCES rooms(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_room_messages_room ON room_messages(room_id, created_at);
`
