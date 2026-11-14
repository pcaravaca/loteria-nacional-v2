package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	JWTSecret    string
	DatabasePath string
	Seed         bool
}

func Load() *Config {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-only-secret-change-me"
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/loteria.db"
	}

	return &Config{
		Port:         port,
		JWTSecret:    secret,
		DatabasePath: dbPath,
		Seed:         os.Getenv("SEED") != "false",
	}
}
