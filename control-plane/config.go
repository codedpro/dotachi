package main

import "os"

type Config struct {
	ListenAddr      string
	DatabaseURL     string
	JWTSecret       string
	AdminPhone      string
	AdminPassword   string
	RoomIdleTimeout string // default "30m" — how long an empty room stays before cleanup
}

func LoadConfig() *Config {
	return &Config{
		ListenAddr:      envOr("LISTEN_ADDR", ":8080"),
		DatabaseURL:     envOr("DATABASE_URL", "postgres://dotachi:dotachi@localhost:5432/dotachi?sslmode=disable"),
		JWTSecret:       envOr("JWT_SECRET", "change-me-in-production"),
		AdminPhone:      envOr("ADMIN_PHONE", "09000000000"),
		AdminPassword:   envOr("ADMIN_PASSWORD", "admin123"),
		RoomIdleTimeout: envOr("ROOM_IDLE_TIMEOUT", "30m"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
