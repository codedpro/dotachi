package service

import (
	"log"
	"time"

	"github.com/dotachi/control-plane/db"
)

// StartHealthChecker launches a background goroutine that periodically
// checks for nodes that have missed their heartbeat window and marks
// them as inactive.
func StartHealthChecker(interval time.Duration) {
	go func() {
		log.Printf("[health] checker started -- interval=%s, stale threshold=2 minutes", interval)
		for {
			time.Sleep(interval)
			checkNodeHealth()
		}
	}()
}

// checkNodeHealth finds nodes whose last_heartbeat is older than 2 minutes
// and marks them as is_active = FALSE.
func checkNodeHealth() {
	// Find stale nodes that are currently active but haven't sent a heartbeat
	// in over 2 minutes.
	rows, err := db.DB.Query(`
		SELECT id, name, last_heartbeat
		FROM nodes
		WHERE is_active = TRUE
		  AND last_heartbeat IS NOT NULL
		  AND last_heartbeat < NOW() - INTERVAL '2 minutes'
	`)
	if err != nil {
		log.Printf("[health] failed to query stale nodes: %v", err)
		return
	}
	defer rows.Close()

	type staleNode struct {
		id            int64
		name          string
		lastHeartbeat time.Time
	}

	var stale []staleNode
	for rows.Next() {
		var n staleNode
		if err := rows.Scan(&n.id, &n.name, &n.lastHeartbeat); err != nil {
			log.Printf("[health] failed to scan node row: %v", err)
			continue
		}
		stale = append(stale, n)
	}

	for _, n := range stale {
		log.Printf("[health] WARNING: node %q (id=%d) missed heartbeat -- last seen %s, marking inactive",
			n.name, n.id, n.lastHeartbeat.Format(time.RFC3339))

		_, err := db.DB.Exec("UPDATE nodes SET is_active = FALSE WHERE id = $1", n.id)
		if err != nil {
			log.Printf("[health] failed to mark node %d as inactive: %v", n.id, err)
		}
	}

	if len(stale) > 0 {
		log.Printf("[health] marked %d node(s) as inactive due to missed heartbeats", len(stale))
	}
}
