package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	JWTSecret    string
	DatabasePath string // SQLite (fallback cuando no hay PostgreSQL)
	DatabaseURL  string // PostgreSQL (tiene prioridad sobre DatabasePath)
	Seed         bool
	// Cierre automatico de sorteos vencidos (reembolsa las apuestas pendientes).
	SorteosCierreAuto     bool
	SorteosSweepMin       int // cada cuantos minutos corre el barrido
	SorteosSweepGraciaMin int // minutos de gracia antes de dar un sorteo por vencido
	// Importación automática de resultados (espejos en texto; la JPS publica
	// PDF escaneados y queda como referencia oficial de verificación).
	ResultadosAuto    bool // importa y paga al detectar un sorteo cantado
	ResultadosMin     int  // cada cuantos minutos se intenta importar
	ResultadosCalAuto bool // genera los sorteos futuros según el calendario
	// Cumplimiento (el operador con licencia define sus politicas):
	GeoBlockEnabled bool            // bloquea el juego desde paises no permitidos
	GeoAllowed      map[string]bool // ISO 3166-1 alpha-2, en mayusculas
	KYCRequired     bool            // exige documento verificado para jugar
	UploadDir       string          // directorio de documentos KYC subidos
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

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}

	return &Config{
		Port:                  port,
		JWTSecret:             secret,
		DatabasePath:          dbPath,
		DatabaseURL:           DatabaseURL(),
		Seed:                  os.Getenv("SEED") != "false",
		SorteosCierreAuto:     envBool("SORTEOS_CIERRE_AUTO", false),
		SorteosSweepMin:       envInt("SORTEOS_SWEEP_MIN", 60),
		SorteosSweepGraciaMin: envInt("SORTEOS_SWEEP_GRACIA_MIN", 90),
		ResultadosAuto:        envBool("RESULTADOS_AUTO", false),
		ResultadosMin:         envInt("RESULTADOS_MIN", 10),
		ResultadosCalAuto:     envBool("RESULTADOS_CALENDARIO_AUTO", false),
		GeoBlockEnabled:       envBool("GEO_BLOCK_ENABLED", false),
		GeoAllowed:            envCountries(os.Getenv("GEO_ALLOWED_COUNTRIES")),
		KYCRequired:           envBool("KYC_REQUIRED", false),
		UploadDir:             uploadDir,
	}
}

// GeoPermite indica si un pais (ISO alpha-2) puede jugar.
func (c *Config) GeoPermite(pais string) bool {
	p := strings.ToUpper(strings.TrimSpace(pais))
	if p == "" {
		return false
	}
	return c.GeoAllowed[p]
}

// envInt lee un entero; si falta o es invalido usa el defecto.
func envInt(nombre string, defecto int) int {
	v := strings.TrimSpace(os.Getenv(nombre))
	if v == "" {
		return defecto
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defecto
	}
	return n
}

func envBool(nombre string, defecto bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(nombre)))
	if v == "" {
		return defecto
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defecto
	}
	return b
}

// envCountries convierte "CR,SV,GT" en un mapa. Sin valor se asume solo CR.
func envCountries(s string) map[string]bool {
	out := map[string]bool{}
	for _, p := range strings.Split(s, ",") {
		p = strings.ToUpper(strings.TrimSpace(p))
		if len(p) == 2 {
			out[p] = true
		}
	}
	if len(out) == 0 {
		out["CR"] = true
	}
	return out
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
