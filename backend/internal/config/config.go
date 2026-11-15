package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	JWTSecret    string
	DatabasePath string // SQLite (fallback cuando no hay PostgreSQL)
	DatabaseURL  string // PostgreSQL (tiene prioridad sobre DatabasePath)
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
		DatabaseURL:  DatabaseURL(),
		Seed:         os.Getenv("SEED") != "false",
	}
}

// DatabaseURL arma el DSN de PostgreSQL. Acepta un DATABASE_URL completo o las
// variables sueltas DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME. Devuelve ""
// si no hay configuracion de PostgreSQL, y entonces se usa SQLite.
func DatabaseURL() string {
	if u := os.Getenv("DATABASE_URL"); u != "" {
		return u
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		return ""
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	tz := os.Getenv("DB_TZ")
	if tz == "" {
		tz = "America/Costa_Rica"
	}
	// El DSN por palabras clave exige comillas si el valor lleva espacios o
	// comillas; se escapa tanto la barra invertida como la comilla simple.
	password := strings.ReplaceAll(os.Getenv("DB_PASSWORD"), `\`, `\\`)
	password = strings.ReplaceAll(password, `'`, `\'`)
	return fmt.Sprintf(
		"host=%s port=%s user=%s password='%s' dbname=%s sslmode=disable TimeZone=%s",
		host, port, os.Getenv("DB_USER"), password, os.Getenv("DB_NAME"), tz,
	)
}

// DatabaseName devuelve el nombre de la base configurada, para los logs.
func (c *Config) DatabaseName() string {
	if c.DatabaseURL == "" {
		return ""
	}
	// DATABASE_URL suele venir como URL (postgres://...); las variables sueltas
	// se arman como DSN por palabras clave (host=... dbname=...).
	if strings.Contains(c.DatabaseURL, "://") {
		if u, err := url.Parse(c.DatabaseURL); err == nil && strings.TrimPrefix(u.Path, "/") != "" {
			return strings.TrimPrefix(u.Path, "/")
		}
	}
	for _, campo := range strings.Fields(c.DatabaseURL) {
		if strings.HasPrefix(campo, "dbname=") {
			return strings.Trim(strings.TrimPrefix(campo, "dbname="), "'")
		}
	}
	return ""
}
