package config

import (
	"flag"
	"os"
)

type Config struct {
	Addr      string
	SecretKey string
	BoltPath  string
}

func Load() Config {
	addr := flag.String("addr", envOr("CROOK_ADDR", ":8080"), "HTTP listen address")
	secret := flag.String("secret", envOr("CROOK_SECRET", "change-me-in-production"), "Cookie signing key")
	boltPath := flag.String("db", envOr("CROOK_DB", "crook.db"), "BoltDB file path")
	flag.Parse()
	return Config{
		Addr:      *addr,
		SecretKey: *secret,
		BoltPath:  *boltPath,
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
