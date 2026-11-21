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
	"loteria-backend/internal/resultados"
)

func cors() gin.HandlerFunc {
	origenes := map[string]bool{
		"http://localhost:5173": true,
		"http://127.0.0.1:5173": true,
		"http://localhost:4173": true,
		"http://localhost:8080": true,
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

	db, err := database.Open(cfg)
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
		if err := database.SeedRifas(db); err != nil {
			log.Fatalf("seed rifas: %v", err)
		}
	}

	r := gin.Default()
	r.Use(cors())

	api := r.Group("/api")
	// Las rutas publicas de contenido identifican al usuario si trae token,
	// para resolver el acceso por paquete o por jugador sin exigir sesion.
	api.Use(middleware.OpcionalAuth(cfg))
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	authH := handlers.NewAuthHandler(db, cfg)
	juegosH := handlers.NewJuegosHandler(db)
	apuestasH := handlers.NewApuestasHandler(db, cfg)
	reportesH := handlers.NewReportesHandler(db)
	adminH := handlers.NewAdminHandler(db)
	cfgH := handlers.NewConfigHandler(db)
	casinoH := handlers.NewCasinoHandler(db, cfg)
	cuentaH := handlers.NewCuentaHandler(db, cfg)
	auditoriaH := handlers.NewAuditoriaHandler(db)
	exportH := handlers.NewExportacionesHandler(db)
	limitesH := handlers.NewLimitesHandler(db)
	rifasH := handlers.NewRifasHandler(db)
	adminRifasH := handlers.NewAdminRifasHandler(db)
	accesosH := handlers.NewAdminAccesosHandler(db)

	api.POST("/auth/register", authH.Register)
	api.GET("/config", cfgH.Config)
	api.POST("/auth/login", authH.Login)

	api.GET("/loterias", juegosH.ListLoterias)
	api.GET("/loterias/:id", juegosH.GetLoteria)
	api.GET("/sorteos", juegosH.ListSorteos)
	api.GET("/sorteos/proximos", juegosH.ProximosSorteos)
	api.GET("/rifas", rifasH.Listar)
	api.GET("/rifas/:id", rifasH.Detalle)
	// Auditoria publica del sorteo: hash siempre, semilla recien cuando ya se sorteo.
	api.GET("/rifas/:id/verificar", adminRifasH.Verificar)

	authed := api.Group("")
	authed.Use(middleware.AuthRequired(cfg))

	authed.GET("/auth/me", authH.Me)
	authed.PUT("/auth/tema", authH.GuardarTema)
	authed.PUT("/auth/tema-app", authH.GuardarTemaApp)
	authed.GET("/dashboard", reportesH.Dashboard)
	authed.GET("/movimientos", reportesH.Movimientos)
	authed.GET("/resultados", reportesH.Resultados)
	authed.GET("/reportes/resumen", reportesH.Resumen)

	authed.POST("/apuestas", middleware.GeoBlock(cfg), apuestasH.Crear)
	authed.GET("/apuestas/mias", apuestasH.Mias)
	authed.POST("/rifas/:id/comprar", middleware.GeoBlock(cfg), rifasH.Comprar)
	authed.GET("/rifas/mias", rifasH.Mias)

	// Juego responsable e identificacion (KYC basico).
	authed.GET("/cuenta/autoexclusion", cuentaH.AutoexclusionEstado)
	authed.POST("/cuenta/autoexclusion", cuentaH.AutoexclusionActivar)
	authed.PUT("/cuenta/identificacion", cuentaH.IdentificacionGuardar)
	authed.POST("/cuenta/identificacion/foto", cuentaH.SubirFotoDocumento)
	authed.GET("/cuenta/limites", limitesH.Obtener)
	authed.PUT("/cuenta/limites", limitesH.Actualizar)

	admin := authed.Group("/admin")
	admin.Use(middleware.RoleRequired(models.RolAdmin))
	admin.GET("/usuarios", adminH.ListUsers)
	admin.POST("/usuarios/:id/verificar", adminH.VerificarUsuario)
	admin.GET("/usuarios/:id/documento", adminH.VerDocumento)
	admin.POST("/usuarios/:id/anonimizar", adminH.AnonimizarUsuario)
	admin.POST("/usuarios/:id/saldo", adminH.AjustarSaldo)
	admin.POST("/loterias", adminH.CreateLoteria)
	admin.PUT("/loterias/:id", adminH.UpdateLoteria)
	admin.PUT("/config", cfgH.Guardar)
	admin.POST("/resultados/importar", adminH.ImportarResultados)
	admin.GET("/resultados/estado", adminH.EstadoResultados)
	admin.POST("/sorteos/calendario", adminH.GenerarCalendario)
	admin.POST("/sorteos", adminH.CreateSorteo)
	admin.GET("/auditoria", auditoriaH.Listar)
	admin.GET("/rifas", adminRifasH.Listar)
	admin.POST("/rifas", adminRifasH.Crear)
	admin.POST("/rifas/:id/sortear", adminRifasH.Sortear)
	admin.POST("/rifas/:id/cancelar", adminRifasH.Cancelar)
	// Accesos por paquete de jugadores o por jugador individual.
	admin.GET("/grupos", accesosH.ListarGrupos)
	admin.POST("/grupos", accesosH.CrearGrupo)
	admin.PUT("/grupos/:id", accesosH.ActualizarGrupo)
	admin.DELETE("/grupos/:id", accesosH.EliminarGrupo)
	admin.GET("/grupos/:id/jugadores", accesosH.ListarMiembros)
	admin.POST("/grupos/:id/jugadores", accesosH.AsignarJugadores)
	admin.DELETE("/grupos/:id/jugadores/:user_id", accesosH.QuitarJugador)
	admin.GET("/accesos", accesosH.ListarAccesos)
	admin.POST("/accesos", accesosH.CrearAcceso)
	admin.PUT("/accesos/:id", accesosH.ActualizarAcceso)
	admin.DELETE("/accesos/:id", accesosH.EliminarAcceso)
	admin.GET("/reportes/csv/:tipo", func(c *gin.Context) {
		switch c.Param("tipo") {
		case "apuestas":
			exportH.Apuestas(c)
		case "sorteos":
			exportH.Sorteos(c)
		case "rondas":
			exportH.RondasCasino(c)
		case "blackjack":
			exportH.BlackjackManos(c)
		case "movimientos":
			exportH.Movimientos(c)
		case "usuarios":
			exportH.Usuarios(c)
		case "auditoria":
			exportH.Auditoria(c)
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": "reporte no disponible"})
		}
	})

	// Cargar resultados es tarea del agente o del administrador.
	adminAgente := authed.Group("/admin")
	adminAgente.Use(middleware.RoleRequired(models.RolAdmin, models.RolAgente))
	adminAgente.POST("/sorteos/:id/cerrar", adminH.CerrarSorteo)
	adminAgente.POST("/sorteos/:id/cancelar", adminH.CancelarSorteo)

	// Mini casino: el resultado y el pago se resuelven en el servidor.
	authed.GET("/casino/games", casinoH.Games)
	authed.GET("/casino/seed", casinoH.Seed)
	authed.POST("/casino/seed/rotate", casinoH.RotarSemilla)
	authed.POST("/casino/play", middleware.GeoBlock(cfg), casinoH.Play)
	authed.GET("/casino/rounds", casinoH.Rounds)
	authed.GET("/casino/rounds/:id/verify", casinoH.Verify)
	admin.GET("/casino/stats", casinoH.Stats)
	admin.PATCH("/casino/games/:codigo", casinoH.ActualizarJuego)

	// Blackjack: una mano de varios turnos (con bloqueo geografico).
	bj := authed.Group("")
	bj.Use(middleware.GeoBlock(cfg))
	bj.POST("/casino/blackjack", casinoH.BlackjackStart)
	bj.POST("/casino/blackjack/:id/hit", casinoH.BlackjackHit)
	bj.POST("/casino/blackjack/:id/stand", casinoH.BlackjackStand)
	bj.POST("/casino/blackjack/:id/double", casinoH.BlackjackDouble)
	bj.POST("/casino/blackjack/:id/surrender", casinoH.BlackjackSurrender)
	bj.POST("/casino/blackjack/:id/insurance", casinoH.BlackjackInsurance)

	// Bingo 75: se apuesta al abrir la jugada y las bolillas se cantan de a una.
	bg := authed.Group("")
	bg.Use(middleware.GeoBlock(cfg))
	bg.POST("/casino/bingo/start", casinoH.BingoStart)
	bg.POST("/casino/bingo/:id/hit", casinoH.BingoHit)
	bg.POST("/casino/bingo/:id/close", casinoH.BingoClose)

	// Gemas (match-3): se apuesta al abrir el tablero y se liquida al tope de
	// movimientos o al tocar "Reventar".
	gm := authed.Group("")
	gm.Use(middleware.GeoBlock(cfg))
	gm.POST("/casino/gemas/start", casinoH.GemasStart)
	gm.POST("/casino/gemas/:id/move", casinoH.GemasMove)
	gm.POST("/casino/gemas/:id/reventar", casinoH.GemasReventar)
	gm.GET("/casino/gemas/:id/verificar", casinoH.GemasVerificar)

	// Video poker: se apostó al repartir y el jugador elige qué cartas sostiene.
	pk := authed.Group("")
	pk.Use(middleware.GeoBlock(cfg))
	pk.POST("/casino/poker", casinoH.PokerRepartir)
	pk.POST("/casino/poker/:id/cobrar", casinoH.PokerCobrar)

	// Juegos de mesa: dominó, ajedrez relámpago, backgammon y tetris por líneas.
	// Todos cobran la apuesta al abrir la ronda y se verifican con las semillas.
	mesa := authed.Group("")
	mesa.Use(middleware.GeoBlock(cfg))
	mesa.POST("/casino/domino/start", casinoH.DominoStart)
	mesa.POST("/casino/domino/:id/move", casinoH.DominoMove)
	mesa.GET("/casino/domino/:id/verificar", casinoH.DominoVerificar)
	mesa.POST("/casino/ajedrez/start", casinoH.AjedrezStart)
	mesa.POST("/casino/ajedrez/:id/move", casinoH.AjedrezMove)
	mesa.GET("/casino/ajedrez/:id/verificar", casinoH.AjedrezVerificar)
	mesa.POST("/casino/ajedrez/partida/start", casinoH.AjedrezPartidaStart)
	mesa.POST("/casino/ajedrez/partida/:id/move", casinoH.AjedrezPartidaMove)
	mesa.GET("/casino/ajedrez/partida/:id/verificar", casinoH.AjedrezPartidaVerificar)
	mesa.POST("/casino/backgammon/start", casinoH.BackgammonStart)
	mesa.POST("/casino/backgammon/:id/move", casinoH.BackgammonMove)
	mesa.GET("/casino/backgammon/:id/verificar", casinoH.BackgammonVerificar)
	mesa.POST("/casino/tetris/start", casinoH.TetrisStart)
	mesa.POST("/casino/tetris/:id/move", casinoH.TetrisMove)
	mesa.GET("/casino/tetris/:id/verificar", casinoH.TetrisVerificar)

	if cfg.SorteosCierreAuto {
		handlers.IniciarSweepSorteos(db, true, time.Duration(cfg.SorteosSweepMin)*time.Minute, 15*time.Minute)
		log.Printf("cierre automatico de sorteos vencidos activado (cada %d min, gracia 15 min)", cfg.SorteosSweepMin)
	}

	// Resultados automáticos: los espejos en texto son la fuente; la JPS queda
	// como referencia oficial de verificación (publica PDF escaneados).
	if err := database.AsegurarLoterias(db); err != nil {
		log.Printf("asegurar loterias: %v", err)
	}

	impResultados := resultados.NuevoImportador(db,
		[]resultados.Proveedor{resultados.NuevoNacionAlloteria(nil), resultados.NuevoYelu(nil)}, nil)
	if n, err := resultados.GenerarSorteosVentana(db, nil, 14, 21); err != nil {
		log.Printf("calendario de sorteos: %v", err)
	} else if n > 0 {
		log.Printf("calendario de sorteos: se crearon %d sorteos", n)
	}
	handlers.IniciarImportador(impResultados, cfg.ResultadosAuto, time.Duration(cfg.ResultadosMin)*time.Minute, 14)

	if cfg.DatabaseURL != "" {
		log.Printf("Loteria CR API escuchando en :%s (postgres dbname=%s)", cfg.Port, cfg.DatabaseName())
	} else {
		log.Printf("Loteria CR API escuchando en :%s (sqlite %s)", cfg.Port, cfg.DatabasePath)
	}
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("servidor: %v", err)
	}
}
