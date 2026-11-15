package config

import "testing"

// TestDatabaseURL verifica que el DSN de PostgreSQL se arme bien desde las
// variables sueltas y que escape la contrasenna.
func TestDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "")
	if got := DatabaseURL(); got != "" {
		t.Fatalf("sin DB_HOST deberia devolver vacio, fue %q", got)
	}

	t.Setenv("DB_HOST", "100.89.167.67")
	t.Setenv("DB_PORT", "5499")
	t.Setenv("DB_USER", "loteria")
	t.Setenv("DB_PASSWORD", `p'ass\w`)
	t.Setenv("DB_NAME", "loteria_dev")

	want := `host=100.89.167.67 port=5499 user=loteria password='p\'ass\\w' dbname=loteria_dev sslmode=disable TimeZone=America/Costa_Rica`
	if got := DatabaseURL(); got != want {
		t.Fatalf("dsn inesperado:\n got %s\nwant %s", got, want)
	}

	// DATABASE_URL completo gana.
	t.Setenv("DATABASE_URL", "postgres://u:p@h:5432/d")
	if got := DatabaseURL(); got != "postgres://u:p@h:5432/d" {
		t.Fatalf("DATABASE_URL deberia tener prioridad, fue %q", got)
	}
}

func TestDatabaseName(t *testing.T) {
	// DSN por palabras clave (lo que arma DatabaseURL).
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "h")
	t.Setenv("DB_USER", "u")
	t.Setenv("DB_PASSWORD", "p")
	t.Setenv("DB_NAME", "loteria_dev")
	cfg := &Config{DatabaseURL: DatabaseURL()}
	if cfg.DatabaseName() != "loteria_dev" {
		t.Fatalf("nombre inesperado: %q", cfg.DatabaseName())
	}

	// URL completa.
	cfg = &Config{DatabaseURL: "postgres://u:p@h:5432/otra?sslmode=disable"}
	if cfg.DatabaseName() != "otra" {
		t.Fatalf("nombre desde url inesperado: %q", cfg.DatabaseName())
	}
}
