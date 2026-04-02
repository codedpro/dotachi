package service

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/dotachi/control-plane/db"
)

// StartCleanupWorker launches a background goroutine that periodically
// cleans up empty and inactive rooms plus expired rooms.
//
// idleTimeout controls how long a room must be empty (zero members)
// before it is deactivated. interval controls how often the worker runs.
func StartCleanupWorker(interval time.Duration, idleTimeout time.Duration) {
	go func() {
		log.Printf("[cleanup] worker started — interval=%s, idle_timeout=%s", interval, idleTimeout)
		for {
			time.Sleep(interval)
			cleanEmptyRooms(idleTimeout)
			cleanExpiredRooms()
			purgeOldInactiveRooms()
		}
	}()
}

// StartSharedBillingWorker launches a background goroutine that charges
// users in shared rooms every minute.
func StartSharedBillingWorker(interval time.Duration) {
	go func() {
		log.Printf("[billing] shared room billing worker started — interval=%s", interval)
		for {
			time.Sleep(interval)
			chargeSharedSessions()
		}
	}()
}

// cleanEmptyRooms finds active rooms with zero members whose last_activity
// exceeds the idle timeout, deletes the hub on the node, and marks them inactive.
func cleanEmptyRooms(idleTimeout time.Duration) {
	rows, err := db.DB.Query(`
		SELECT r.id, r.hub_name, r.node_id, r.last_activity,
		       n.host, n.api_port, n.api_secret,
		       (SELECT COUNT(*) FROM room_members WHERE room_id = r.id) AS member_count
		FROM rooms r
		JOIN nodes n ON r.node_id = n.id
		WHERE r.is_active = TRUE
	`)
	if err != nil {
		log.Printf("[cleanup] error querying active rooms: %v", err)
		return
	}
	defer rows.Close()

	type emptyRoom struct {
		id           int64
		hubName      string
		nodeID       int64
		lastActivity time.Time
		host         string
		apiPort      int
		apiSecret    string
		memberCount  int
	}

	var candidates []emptyRoom
	for rows.Next() {
		var r emptyRoom
		if err := rows.Scan(&r.id, &r.hubName, &r.nodeID, &r.lastActivity,
			&r.host, &r.apiPort, &r.apiSecret, &r.memberCount); err != nil {
			log.Printf("[cleanup] error scanning room row: %v", err)
			continue
		}
		candidates = append(candidates, r)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[cleanup] rows iteration error: %v", err)
		return
	}

	now := time.Now()
	for _, r := range candidates {
		if r.memberCount > 0 {
			continue
		}
		if now.Sub(r.lastActivity) < idleTimeout {
			continue
		}

		log.Printf("[cleanup] deactivating empty room id=%d hub=%s (empty since %s)",
			r.id, r.hubName, r.lastActivity.Format(time.RFC3339))

		// Delete the hub on the node (best effort)
		if err := DeleteHub(r.host, r.apiPort, r.apiSecret, r.hubName); err != nil {
			log.Printf("[cleanup] failed to delete hub %s on node %d: %v", r.hubName, r.nodeID, err)
		}

		_, err := db.DB.Exec("UPDATE rooms SET is_active = FALSE WHERE id = $1", r.id)
		if err != nil {
			log.Printf("[cleanup] failed to mark room %d as inactive: %v", r.id, err)
		}
	}
}

// cleanExpiredRooms finds rooms whose expires_at has passed and deactivates them.
func cleanExpiredRooms() {
	rows, err := db.DB.Query(`
		SELECT r.id, r.hub_name, r.node_id,
		       n.host, n.api_port, n.api_secret
		FROM rooms r
		JOIN nodes n ON r.node_id = n.id
		WHERE r.is_active = TRUE
		  AND r.expires_at IS NOT NULL
		  AND r.expires_at < NOW()
	`)
	if err != nil {
		log.Printf("[cleanup] error querying expired rooms: %v", err)
		return
	}
	defer rows.Close()

	type expiredRoom struct {
		id        int64
		hubName   string
		nodeID    int64
		host      string
		apiPort   int
		apiSecret string
	}

	var expired []expiredRoom
	for rows.Next() {
		var r expiredRoom
		if err := rows.Scan(&r.id, &r.hubName, &r.nodeID, &r.host, &r.apiPort, &r.apiSecret); err != nil {
			log.Printf("[cleanup] error scanning expired room row: %v", err)
			continue
		}
		expired = append(expired, r)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[cleanup] expired rooms iteration error: %v", err)
		return
	}

	for _, r := range expired {
		log.Printf("[cleanup] deactivating expired room id=%d hub=%s", r.id, r.hubName)

		// Remove all members from the VPN hub (best effort)
		memberRows, err := db.DB.Query(
			"SELECT vpn_username FROM room_members WHERE room_id = $1", r.id,
		)
		if err == nil {
			for memberRows.Next() {
				var vpnUser string
				if memberRows.Scan(&vpnUser) == nil {
					DisconnectUser(r.host, r.apiPort, r.apiSecret, r.hubName, vpnUser)
					DeleteVPNUser(r.host, r.apiPort, r.apiSecret, r.hubName, vpnUser)
				}
			}
			memberRows.Close()
		}

		// Close any open play sessions for members in this room
		db.DB.Exec(`
			UPDATE play_sessions SET left_at = CURRENT_TIMESTAMP,
				duration_minutes = EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - joined_at))::INTEGER / 60
			WHERE room_id = $1 AND left_at IS NULL`, r.id)

		// Close any open shared sessions
		db.DB.Exec(
			"UPDATE shared_sessions SET ended_at = CURRENT_TIMESTAMP WHERE room_id = $1 AND ended_at IS NULL",
			r.id,
		)

		// Delete members
		db.DB.Exec("DELETE FROM room_members WHERE room_id = $1", r.id)

		// Delete hub on node
		if err := DeleteHub(r.host, r.apiPort, r.apiSecret, r.hubName); err != nil {
			log.Printf("[cleanup] failed to delete hub %s on node %d: %v", r.hubName, r.nodeID, err)
		}

		// Mark room inactive
		_, err = db.DB.Exec("UPDATE rooms SET is_active = FALSE WHERE id = $1", r.id)
		if err != nil {
			log.Printf("[cleanup] failed to mark expired room %d as inactive: %v", r.id, err)
		}
	}

	if len(expired) > 0 {
		log.Printf("[cleanup] deactivated %d expired room(s)", len(expired))
	}
}

// purgeOldInactiveRooms permanently deletes rooms that have been inactive
// for more than 24 hours along with their associated data.
func purgeOldInactiveRooms() {
	rows, err := db.DB.Query(`
		SELECT id, hub_name FROM rooms
		WHERE is_active = FALSE AND last_activity < NOW() - INTERVAL '24 hours'
	`)
	if err != nil {
		log.Printf("[cleanup] error querying inactive rooms for purge: %v", err)
		return
	}
	defer rows.Close()

	type staleRoom struct {
		id      int64
		hubName string
	}

	var toPurge []staleRoom
	for rows.Next() {
		var r staleRoom
		if err := rows.Scan(&r.id, &r.hubName); err != nil {
			log.Printf("[cleanup] error scanning stale room row: %v", err)
			continue
		}
		toPurge = append(toPurge, r)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[cleanup] rows iteration error: %v", err)
		return
	}

	for _, r := range toPurge {
		log.Printf("[cleanup] purging inactive room id=%d hub=%s", r.id, r.hubName)

		// Delete dependent rows first (foreign key constraints)
		if _, err := db.DB.Exec("DELETE FROM shared_sessions WHERE room_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete shared_sessions for room %d: %v", r.id, err)
			continue
		}
		if _, err := db.DB.Exec("DELETE FROM room_roles WHERE room_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete room_roles for room %d: %v", r.id, err)
			continue
		}
		if _, err := db.DB.Exec("DELETE FROM favorites WHERE room_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete favorites for room %d: %v", r.id, err)
			continue
		}
		if _, err := db.DB.Exec("DELETE FROM play_sessions WHERE room_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete play_sessions for room %d: %v", r.id, err)
			continue
		}
		if _, err := db.DB.Exec("DELETE FROM room_bans WHERE room_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete bans for room %d: %v", r.id, err)
			continue
		}
		if _, err := db.DB.Exec("DELETE FROM room_members WHERE room_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete members for room %d: %v", r.id, err)
			continue
		}
		// Delete room messages
		if _, err := db.DB.Exec("DELETE FROM room_messages WHERE room_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete room_messages for room %d: %v", r.id, err)
			continue
		}
		// Delete room invites
		if _, err := db.DB.Exec("DELETE FROM room_invites WHERE room_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete room_invites for room %d: %v", r.id, err)
			continue
		}
		// Preserve shard transactions by nulling out ref_id (audit trail)
		if _, err := db.DB.Exec("UPDATE shard_transactions SET ref_id = NULL WHERE ref_id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to null shard_transactions ref_id for room %d: %v", r.id, err)
			continue
		}
		if _, err := db.DB.Exec("DELETE FROM rooms WHERE id = $1", r.id); err != nil {
			log.Printf("[cleanup] failed to delete room %d: %v", r.id, err)
		}
	}

	if len(toPurge) > 0 {
		log.Printf("[cleanup] purged %d inactive room(s)", len(toPurge))
	}
}

// chargeSharedSessions finds open shared sessions and charges users hourly.
func chargeSharedSessions() {
	rows, err := db.DB.Query(`
		SELECT ss.id, ss.user_id, ss.room_id, ss.started_at, ss.shards_charged,
		       r.hourly_cost, r.hub_name, r.node_id,
		       n.host, n.api_port, n.api_secret
		FROM shared_sessions ss
		JOIN rooms r ON r.id = ss.room_id
		JOIN nodes n ON n.id = r.node_id
		WHERE ss.ended_at IS NULL AND r.hourly_cost > 0
	`)
	if err != nil {
		log.Printf("[billing] error querying shared sessions: %v", err)
		return
	}
	defer rows.Close()

	type openSession struct {
		id            int64
		userID        int64
		roomID        int64
		startedAt     time.Time
		shardsCharged int
		hourlyCost    int
		hubName       string
		nodeID        int64
		host          string
		apiPort       int
		apiSecret     string
	}

	var sessions []openSession
	for rows.Next() {
		var s openSession
		if err := rows.Scan(&s.id, &s.userID, &s.roomID, &s.startedAt, &s.shardsCharged,
			&s.hourlyCost, &s.hubName, &s.nodeID, &s.host, &s.apiPort, &s.apiSecret); err != nil {
			log.Printf("[billing] error scanning shared session: %v", err)
			continue
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[billing] rows iteration error: %v", err)
		return
	}

	for _, s := range sessions {
		elapsed := time.Since(s.startedAt)
		totalHours := int(math.Floor(elapsed.Hours()))
		if totalHours < 1 {
			continue // less than 1 hour elapsed, don't charge yet
		}

		expectedCharge := totalHours * s.hourlyCost
		owed := expectedCharge - s.shardsCharged
		if owed <= 0 {
			continue // already charged for this period
		}

		// Begin transaction for billing
		tx, txErr := db.DB.Begin()
		if txErr != nil {
			log.Printf("[billing] failed to begin transaction for session %d: %v", s.id, txErr)
			continue
		}

		// Get user balance (with row lock)
		var balance int
		err := tx.QueryRow("SELECT shard_balance FROM users WHERE id = $1 FOR UPDATE", s.userID).Scan(&balance)
		if err != nil {
			tx.Rollback()
			continue
		}

		if balance < owed {
			// Not enough shards — close session and kick user
			log.Printf("[billing] user %d cannot afford shared room %d — kicking (balance=%d, owed=%d)",
				s.userID, s.roomID, balance, owed)

			// Charge whatever they have
			charged := balance
			tx.Exec("UPDATE users SET shard_balance = 0 WHERE id = $1", s.userID)

			if charged > 0 {
				tx.Exec(
					`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description, ref_id)
					VALUES ($1, $2, 0, 'shared_hourly', $3, $4)`,
					s.userID, -charged,
					fmt.Sprintf("Shared room hourly charge (insufficient funds, charged %d)", charged),
					s.roomID,
				)
			}

			// Close session
			tx.Exec(
				"UPDATE shared_sessions SET ended_at = CURRENT_TIMESTAMP, shards_charged = shards_charged + $1 WHERE id = $2",
				charged, s.id,
			)

			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("[billing] failed to commit kick transaction for session %d: %v", s.id, commitErr)
				continue
			}

			// Kick from room (network calls outside transaction)
			var vpnUsername string
			err := db.DB.QueryRow(
				"SELECT vpn_username FROM room_members WHERE room_id = $1 AND user_id = $2",
				s.roomID, s.userID,
			).Scan(&vpnUsername)
			if err == nil {
				DisconnectUser(s.host, s.apiPort, s.apiSecret, s.hubName, vpnUsername)
				DeleteVPNUser(s.host, s.apiPort, s.apiSecret, s.hubName, vpnUsername)
				db.DB.Exec("DELETE FROM room_members WHERE room_id = $1 AND user_id = $2", s.roomID, s.userID)
			}

			// Remove role (but not owner)
			var role string
			roleErr := db.DB.QueryRow("SELECT role FROM room_roles WHERE room_id = $1 AND user_id = $2", s.roomID, s.userID).Scan(&role)
			if roleErr == nil && role != "owner" {
				db.DB.Exec("DELETE FROM room_roles WHERE room_id = $1 AND user_id = $2", s.roomID, s.userID)
			}

			// Close play session
			db.DB.Exec(`
				UPDATE play_sessions SET left_at = CURRENT_TIMESTAMP,
					duration_minutes = EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - joined_at))::INTEGER / 60
				WHERE user_id = $1 AND room_id = $2 AND left_at IS NULL`, s.userID, s.roomID)

			continue
		}

		// Charge normally
		newBalance := balance - owed
		tx.Exec("UPDATE users SET shard_balance = $1 WHERE id = $2", newBalance, s.userID)
		tx.Exec(
			`INSERT INTO shard_transactions (user_id, amount, balance_after, tx_type, description, ref_id)
			VALUES ($1, $2, $3, 'shared_hourly', $4, $5)`,
			s.userID, -owed, newBalance,
			fmt.Sprintf("Shared room hourly charge (%d hours)", totalHours),
			s.roomID,
		)
		tx.Exec(
			"UPDATE shared_sessions SET shards_charged = $1 WHERE id = $2",
			expectedCharge, s.id,
		)

		if commitErr := tx.Commit(); commitErr != nil {
			log.Printf("[billing] failed to commit charge transaction for session %d: %v", s.id, commitErr)
		}
	}
}
