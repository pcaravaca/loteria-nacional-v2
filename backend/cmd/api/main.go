package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/handlers"
	"loteria-backend/internal/middleware"
	"loteria-backend/internal/models"
)

func cors() gin.HandlerFunc {
	origenes := map[string]bool{
		"http://localhost:5173":  true,
		"http://127.0.0.1:5173":  true,
		"http://localhost:4173":  true,
		"http://localhost:8080":  true,
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origenes[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("base de datos: %v", err)
	}

	if cfg.Seed {
		if err := database.Seed(db); err != nil {
			log.Fatalf("seed: %v", err)
		}
		if err := database.SeedCasino(db); err != nil {
			log.Fatalf("seed casino: %v", err)
		}
	}

	r := gin.Default()
	r.Use(cors())

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	authH := handlers.NewAuthHandler(db, cfg)
	juegosH := handlers.NewJuegosHandler(db)
	apuestasH := handlers.NewApuestasHandler(db)
	reportesH := handlers.NewReportesHandler(db)
	adminH := handlers.NewAdminHandler(db)
	casinoH := handlers.NewCasinoHandler(db)

	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)

	api.GET("/loterias", juegosH.ListLoterias)
	api.GET("/loterias/:id", juegosH.GetLoteria)
	api.GET("/sorteos", juegosH.ListSorteos)
	api.GET("/sorteos/proximos", juegosH.ProximosSorteos)

	authed := api.Group("")
	authed.Use(middleware.AuthRequired(cfg))

	authed.GET("/auth/me", authH.Me)
	authed.GET("/dashboard", reportesH.Dashboard)
	authed.GET("/movimientos", reportesH.Movimientos)
	authed.GET("/resultados", reportesH.Resultados)
	authed.GET("/reportes/resumen", reportesH.Resumen)

	authed.POST("/apuestas", apuestasH.Crear)
	authed.GET("/apuestas/mias", apuestasH.Mias)

	admin := authed.Group("/admin")
	admin.Use(middleware.RoleRequired(models.RolAdmin))
	admin.GET("/usuarios", adminH.ListUsers)
	admin.POST("/usuarios/:id/saldo", adminH.AjustarSaldo)
	admin.POST("/loterias", adminH.CreateLoteria)
	admin.PUT("/loterias/:id", adminH.UpdateLoteria)
	admin.POST("/sorteos", adminH.CreateSorteo)

	// Cargar resultados es tarea del agente o del administrador.
	adminAgente := authed.Group("/admin")
	adminAgente.Use(middleware.RoleRequired(models.RolAdmin, models.RolAgente))
	adminAgente.POST("/sorteos/:id/cerrar", adminH.CerrarSorteo)

	// Mini casino: el resultado y el pago se resuelven en el servidor.
	authed.GET("/casino/games", casinoH.Games)
	authed.GET("/casino/seed", casinoH.Seed)
	authed.POST("/casino/seed/rotate", casinoH.RotarSemilla)
	authed.POST("/casino/play", casinoH.Play)
	authed.GET("/casino/rounds", casinoH.Rounds)
	authed.GET("/casino/rounds/:id/verify", casinoH.Verify)
	admin.GET("/casino/stats", casinoH.Stats)

	// Blackjack: una mano de varios turnos.
	authed.POST("/casino/blackjack", casinoH.BlackjackStart)
	authed.POST("/casino/blackjack/:id/hit", casinoH.BlackjackHit)
	authed.POST("/casino/blackjack/:id/stand", casinoH.BlackjackStand)
	authed.POST("/casino/blackjack/:id/double", casinoH.BlackjackDouble)

	log.Printf("Loteria CR API escuchando en :%s (db=%s)", cfg.Port, cfg.DatabasePath)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("servidor: %v", err)
	}
}
