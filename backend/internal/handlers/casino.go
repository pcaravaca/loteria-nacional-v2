package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// CasinoHandler expone el mini casino. El resultado y el pago se calculan aqui;
// el cliente solo anima lo que recibe.
type CasinoHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewCasinoHandler(db *gorm.DB, cfg *config.Config) *CasinoHandler {
	return &CasinoHandler{db: db, cfg: cfg}
}

// Games devuelve el catalogo de juegos activos.
type juegoResponse struct {
	models.CasinoGame
	ApuestaMax  int64              `json:"apuesta_max"` // max_apuesta ya recortado por el techo de premio
	Modalidades []casino.Modalidad `json:"modalidades"` // variantes seleccionables (vacio si no tiene)
}

// Games lista los juegos con la apuesta maxima real: si hay techo de premio, es
// menor que max_apuesta. El cliente usa este valor para no ofrecer apuestas que
// el backend rechazaria. Los admin/agentes pueden pedir el catalogo completo
// con ?todas=1; el resto solo ve los juegos con acceso activo.
func (h *CasinoHandler) Games(c *gin.Context) {
	verTodas := c.Query("todas") == "1" && rolOperador(c)
	var juegos []models.CasinoGame
	if err := h.db.Order("orden").Find(&juegos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los juegos"})
		return
	}
	visibles := map[string]bool{}
	if !verTodas {
		v, err := JuegosVisibles(h.db, c.GetUint("userID"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los juegos"})
			return
		}
		visibles = v
	}
	salida := make([]juegoResponse, 0, len(juegos))
	for _, j := range juegos {
		if !verTodas && !visibles[j.Codigo] {
			continue
		}
		salida = append(salida, juegoResponse{
			CasinoGame:  j,
			ApuestaMax:  casino.ApuestaMaxima(j.MaxApuesta, j.MaxMultipl, j.MaxPremio, j.Codigo),
			Modalidades: casino.Modalidades(j.Codigo),
		})
	}
	c.JSON(http.StatusOK, salida)
}

type semillaResponse struct {
	Hash       string `json:"hash"`
	Nonce      int64  `json:"nonce"`
	ClientSeed string `json:"client_seed"`
}

// Seed devuelve el hash de la semilla del servidor (para comprometer el
// resultado) y la semilla del cliente, que el jugador puede rotar.
func (h *CasinoHandler) Seed(c *gin.Context) {
	userID := c.GetUint("userID")

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}

	// Si el usuario nunca jugo, se crea la semilla ahora para que el hash que
	// se muestra sea el que comprometera la primera jugada.
	if user.CasinoSeed == "" {
		semilla, _, err := casino.NuevaSemilla()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear la semilla"})
			return
		}
		if err := h.db.Model(&models.User{}).Where("id = ?", userID).
			Update("casino_seed", semilla).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la semilla"})
			return
		}
		user.CasinoSeed = semilla
	}

	clientSeed := "cli-" + strconv.FormatUint(uint64(userID), 10) + "-" + strconv.FormatInt(user.CasinoNonce, 10)
	c.JSON(http.StatusOK, semillaResponse{
		Hash:       casino.HashSemilla(user.CasinoSeed),
		Nonce:      user.CasinoNonce,
		ClientSeed: clientSeed,
	})
}

// RotarSemilla genera una semilla nueva. Solo se puede hacer cuando no hay
// rondas sin verificar con la semilla actual.
func (h *CasinoHandler) RotarSemilla(c *gin.Context) {
	userID := c.GetUint("userID")

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}

	var abiertas int64
	if err := h.db.Model(&models.PokerMano{}).
		Where("user_id = ? AND estado = ?", userID, "abierta").Count(&abiertas).Error; err == nil && abiertas > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tenes una mano de video poker abierta: cobrala antes de rotar la semilla"})
		return
	}
	if err := h.db.Model(&models.BingoRonda{}).
		Where("user_id = ? AND estado = ?", userID, "abierta").Count(&abiertas).Error; err == nil && abiertas > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tenes una jugada de bingo abierta: cobrala antes de rotar la semilla"})
		return
	}
	if err := h.db.Model(&models.GemasRonda{}).
		Where("user_id = ? AND estado = ?", userID, "abierta").Count(&abiertas).Error; err == nil && abiertas > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tenes una jugada de gemas abierta: reventala antes de rotar la semilla"})
		return
	}
	if err := h.db.Model(&models.DominoRonda{}).
		Where("user_id = ? AND estado = ?", userID, "abierta").Count(&abiertas).Error; err == nil && abiertas > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tenes una mano de domino abierta: cerrala antes de rotar la semilla"})
		return
	}
	if err := h.db.Model(&models.AjedrezRonda{}).
		Where("user_id = ? AND estado = ?", userID, "abierta").Count(&abiertas).Error; err == nil && abiertas > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tenes una jugada de ajedrez abierta: cerrala antes de rotar la semilla"})
		return
	}
	if err := h.db.Model(&models.AjedrezPartida{}).
		Where("user_id = ? AND estado = ?", userID, "abierta").Count(&abiertas).Error; err == nil && abiertas > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tenes una partida de ajedrez abierta: cerrala antes de rotar la semilla"})
		return
	}
	if err := h.db.Model(&models.BackgammonRonda{}).
		Where("user_id = ? AND estado = ?", userID, "abierta").Count(&abiertas).Error; err == nil && abiertas > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tenes una carrera de backgammon abierta: cerrala antes de rotar la semilla"})
		return
	}
	if err := h.db.Model(&models.TetrisRonda{}).
		Where("user_id = ? AND estado = ?", userID, "abierta").Count(&abiertas).Error; err == nil && abiertas > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tenes una ronda de tetris abierta: cerrala antes de rotar la semilla"})
		return
	}

	semilla, hash, err := casino.NuevaSemilla()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar la semilla"})
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]any{"casino_seed": semilla, "casino_nonce": 0}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo rotar la semilla"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"hash": hash, "nonce": 0})
}

type playRequest struct {
	Juego      string        `json:"game" binding:"required"`
	Monto      int64         `json:"monto" binding:"required,min=1"`
	ClientSeed string        `json:"client_seed"`
	Params     casino.Params `json:"params"`
}

// Play resuelve una jugada: descuenta la apuesta, paga el premio y registra la
// ronda con el hash de la semilla que la comprometia.
func (h *CasinoHandler) Play(c *gin.Context) {
	var req playRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	userID := c.GetUint("userID")

	var juego models.CasinoGame
	if err := h.db.Where("codigo = ?", req.Juego).First(&juego).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "juego no encontrado"})
		return
	}
	if !juego.Activo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el juego esta desactivado"})
		return
	}
	// El acceso por paquete o por jugador puede apagar el juego sin tocar el
	// interruptor global: la jugada se rechaza igual (403 claro).
	if !ExigirAcceso(c, h.db, userID, models.ContenidoJuego, juego.Codigo) {
		return
	}
	if req.Monto < juego.MinApuesta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el minimo es " + strconv.FormatInt(juego.MinApuesta, 10)})
		return
	}
	// El techo de premio acota la apuesta maxima: con el peor multiplicador
	// posible el premio nunca supera MaxPremio.
	apuestaMax := casino.ApuestaMaxima(juego.MaxApuesta, juego.MaxMultipl, juego.MaxPremio, juego.Codigo)
	if req.Monto > apuestaMax {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el maximo es " + strconv.FormatInt(apuestaMax, 10), "maximo": apuestaMax})
		return
	}
	if req.ClientSeed == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "se requiere la semilla del cliente"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	if !GuardJugada(c, h.cfg, user) {
		return
	}
	if user.Saldo < req.Monto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente", "saldo": user.Saldo, "monto": req.Monto})
		return
	}
	if !h.verificarLimite(c, user, juego.Codigo, req.Monto) {
		return
	}

	// Limites voluntarios de gasto del jugador (juego responsable).
	if excede, disponible, motivo := EvaluarGasto(h.db, user.ID, req.Monto); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu limite " + motivo + " de gasto se excederia con esta jugada",
			"disponible": disponible, "motivo": motivo,
		})
		return
	}

	// Semilla del servidor: se crea la primera vez que el usuario juega.
	if user.CasinoSeed == "" {
		semilla, _, err := casino.NuevaSemilla()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear la semilla"})
			return
		}
		user.CasinoSeed = semilla
	}

	draws := casino.Sortear(user.CasinoSeed, req.ClientSeed, user.CasinoNonce, casino.Draws(juego.Codigo), 1<<31-1)

	out, err := casino.Jugar(juego.Codigo, req.Monto, draws, req.Params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	premio := casino.Premio(req.Monto, out)

	ronda := models.CasinoRound{
		UserID:         userID,
		GameCodigo:     juego.Codigo,
		Monto:          req.Monto,
		Multiplicador:  out.Multiplicador,
		Premio:         premio,
		ClientSeed:     req.ClientSeed,
		ServerSeedHash: casino.HashSemilla(user.CasinoSeed),
		Nonce:          user.CasinoNonce,
		Resultado:      out.Data,
	}

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ronda).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
			"saldo":        gorm.Expr("saldo - ?", req.Monto),
			"casino_nonce": user.CasinoNonce + 1,
			"casino_seed":  user.CasinoSeed,
		}).Error; err != nil {
			return err
		}
		movApuesta := models.Movimiento{
			UserID:      userID,
			Tipo:        models.MovCasinoApuesta,
			Monto:       -req.Monto,
			Descripcion: "Casino " + juego.Nombre,
		}
		if err := tx.Create(&movApuesta).Error; err != nil {
			return err
		}
		if premio > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
				return err
			}
			movPremio := models.Movimiento{
				UserID:      userID,
				Tipo:        models.MovCasinoPremio,
				Monto:       premio,
				Descripcion: "Premio casino " + juego.Nombre,
			}
			if err := tx.Create(&movPremio).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar la jugada"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, userID)

	c.JSON(http.StatusOK, gin.H{
		"saldo":            actualizado.Saldo,
		"premio":           premio,
		"multiplicador":    out.Multiplicador,
		"resultado":        out.Data,
		"server_seed_hash": ronda.ServerSeedHash,
		"nonce":            ronda.Nonce,
		"ronda_id":         ronda.ID,
	})
}

// Rounds devuelve el historial de jugadas del usuario, unificado: las de
// casino_rondas, bingo_rondas, gemas_rondas y las mesas. Ver casino_historial.go.
// Se pagina con `pagina` y `por_pagina` porque las fuentes son varias y el
// corte se hace sobre el resultado mezclado.
func (h *CasinoHandler) Rounds(c *gin.Context) {
	userID := c.GetUint("userID")
	pagina, porPagina := leerPaginacion(c)
	// Cada fuente trae las filas que alcanzan para llenar hasta esta pagina;
	// el mezclado deja las mas recientes primero y el corte final es la pagina
	// pedida (no las primeras `por_pagina` filas: eso repetiria la pagina 1).
	filas := h.historialRondas(userID, porPagina*pagina)
	desde := (pagina - 1) * porPagina
	hasta := pagina * porPagina
	switch {
	case desde >= len(filas):
		filas = nil
	case hasta > len(filas):
		filas = filas[desde:]
	default:
		filas = filas[desde:hasta]
	}
	respuestaPagina(c, filas, h.totalHistorial(userID), pagina, porPagina)
}

// Stats devuelve totales por juego (solo admin).
func (h *CasinoHandler) Stats(c *gin.Context) {
	type fila struct {
		Juego     string `json:"juego"`
		Jugadas   int64  `json:"jugadas"`
		Apostado  int64  `json:"apostado"`
		Premios   int64  `json:"premios"`
		Beneficio int64  `json:"beneficio"`
	}
	filas := []fila{}
	if err := h.db.Model(&models.CasinoRound{}).
		Select("game_codigo as juego, count(*) as jugadas, sum(monto) as apostado, sum(premio) as premios, sum(monto - premio) as beneficio").
		Group("game_codigo").Scan(&filas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las estadisticas"})
		return
	}
	// El blackjack y el bingo guardan sus rondas en tablas aparte, asi que se
	// suman sus totales a mano.
	sumar := func(codigo string, total fila) {
		if total.Jugadas == 0 {
			return
		}
		total.Juego = codigo
		for i := range filas {
			if filas[i].Juego == codigo {
				filas[i].Jugadas += total.Jugadas
				filas[i].Apostado += total.Apostado
				filas[i].Premios += total.Premios
				filas[i].Beneficio += total.Beneficio
				return
			}
		}
		filas = append(filas, total)
	}
	var bjTotal fila
	h.db.Model(&models.BlackjackMano{}).
		Select("count(*) as jugadas, COALESCE(sum(monto),0) as apostado, COALESCE(sum(premio),0) as premios, COALESCE(sum(monto - premio),0) as beneficio").
		Scan(&bjTotal)
	sumar(models.CasinoBlackjack, bjTotal)

	var bgTotal fila
	h.db.Model(&models.BingoRonda{}).
		Select("count(*) as jugadas, COALESCE(sum(monto),0) as apostado, COALESCE(sum(premio),0) as premios, COALESCE(sum(monto - premio),0) as beneficio").
		Scan(&bgTotal)
	sumar(models.CasinoBingo, bgTotal)

	var gmTotal fila
	h.db.Model(&models.GemasRonda{}).
		Select("count(*) as jugadas, COALESCE(sum(monto),0) as apostado, COALESCE(sum(premio),0) as premios, COALESCE(sum(monto - premio),0) as beneficio").
		Scan(&gmTotal)
	sumar(models.CasinoGemas, gmTotal)

	// Los cuatro juegos de mesa tambien guardan sus rondas en tablas propias.
	var dmTotal fila
	h.db.Model(&models.DominoRonda{}).
		Select("count(*) as jugadas, COALESCE(sum(monto),0) as apostado, COALESCE(sum(premio),0) as premios, COALESCE(sum(monto - premio),0) as beneficio").
		Scan(&dmTotal)
	sumar(models.CasinoDomino, dmTotal)

	var ajTotal fila
	h.db.Model(&models.AjedrezRonda{}).
		Select("count(*) as jugadas, COALESCE(sum(monto),0) as apostado, COALESCE(sum(premio),0) as premios, COALESCE(sum(monto - premio),0) as beneficio").
		Scan(&ajTotal)
	sumar(models.CasinoAjedrez, ajTotal)

	// Las partidas completas de ajedrez viven en su propia tabla pero suman al
	// mismo juego.
	var apTotal fila
	h.db.Model(&models.AjedrezPartida{}).
		Select("count(*) as jugadas, COALESCE(sum(monto),0) as apostado, COALESCE(sum(premio),0) as premios, COALESCE(sum(monto - premio),0) as beneficio").
		Scan(&apTotal)
	sumar(models.CasinoAjedrez, apTotal)

	var bgmTotal fila
	h.db.Model(&models.BackgammonRonda{}).
		Select("count(*) as jugadas, COALESCE(sum(monto),0) as apostado, COALESCE(sum(premio),0) as premios, COALESCE(sum(monto - premio),0) as beneficio").
		Scan(&bgmTotal)
	sumar(models.CasinoBackgammon, bgmTotal)

	var ttTotal fila
	h.db.Model(&models.TetrisRonda{}).
		Select("count(*) as jugadas, COALESCE(sum(monto),0) as apostado, COALESCE(sum(premio),0) as premios, COALESCE(sum(monto - premio),0) as beneficio").
		Scan(&ttTotal)
	sumar(models.CasinoTetris, ttTotal)

	c.JSON(http.StatusOK, filas)
}

// ActualizarJuego cambia los ajustes de un juego: activo, limites y tope diario.
// Solo lo puede hacer un admin.
func (h *CasinoHandler) ActualizarJuego(c *gin.Context) {
	codigo := c.Param("codigo")
	var existente models.CasinoGame
	if err := h.db.Where("codigo = ?", codigo).First(&existente).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "juego no encontrado"})
		return
	}
	var req struct {
		Activo       *bool    `json:"activo"`
		MinApuesta   *int64   `json:"min_apuesta"`
		MaxApuesta   *int64   `json:"max_apuesta"`
		RTP          *float64 `json:"rtp"`
		MaxMultipl   *int64   `json:"max_multiplicador"`
		LimiteDiario *int64   `json:"limite_diario"`
		MaxPremio    *int64   `json:"max_premio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	campos := map[string]any{}
	if req.Activo != nil {
		campos["activo"] = *req.Activo
	}
	if req.MinApuesta != nil && *req.MinApuesta >= 0 {
		campos["min_apuesta"] = *req.MinApuesta
	}
	if req.MaxApuesta != nil && *req.MaxApuesta >= 0 {
		campos["max_apuesta"] = *req.MaxApuesta
	}
	if req.LimiteDiario != nil && *req.LimiteDiario >= 0 {
		campos["limite_diario"] = *req.LimiteDiario
	}
	if req.MaxPremio != nil && *req.MaxPremio >= 0 {
		campos["max_premio"] = *req.MaxPremio
	}
	if req.RTP != nil && *req.RTP > 0 && *req.RTP <= 100 {
		campos["rtp"] = *req.RTP
	}
	if req.MaxMultipl != nil && *req.MaxMultipl >= 0 {
		campos["max_multipl"] = *req.MaxMultipl
	}

	// Se revisa sobre los valores que quedarian guardados, no solo sobre lo
	// que viene en el request.
	min := existente.MinApuesta
	max := existente.MaxApuesta
	premio := existente.MaxPremio
	if v, ok := campos["min_apuesta"]; ok {
		min = v.(int64)
	}
	if v, ok := campos["max_apuesta"]; ok {
		max = v.(int64)
	}
	if v, ok := campos["max_premio"]; ok {
		premio = v.(int64)
	}
	if max < min {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("el maximo %d no puede ser menor que el minimo %d", max, min)})
		return
	}
	mult := casino.MultiplicadorMaximo(codigo, existente.MaxMultipl)
	if v, ok := campos["max_multipl"]; ok {
		mult = casino.MultiplicadorMaximo(codigo, v.(int64))
	}
	if premio > 0 && mult > 0 && premio/mult < min {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("con un techo de premio de %d y multiplicador %d no se podria pagar ni el minimo (%d): subi el techo o baja el minimo", premio, mult, min),
		})
		return
	}
	if len(campos) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no hay cambios que aplicar"})
		return
	}
	if err := h.db.Model(&models.CasinoGame{}).Where("codigo = ?", codigo).Updates(campos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el juego"})
		return
	}
	var actualizado models.CasinoGame
	_ = h.db.Where("codigo = ?", codigo).First(&actualizado)

	detalle, _ := json.Marshal(campos)
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "casino.juego.actualizar", "juego", codigo, string(detalle))

	c.JSON(http.StatusOK, actualizado)
}

// ------------------------------------------------------------ blackjack
// El reparto es alternado y determinista: la carta n del jugador es mazo[2n]
// y la n del crupier es mazo[2n+1], con el mazo barajado por las semillas.

type blackjackRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	ClientSeed string `json:"client_seed"`
}

func (h *CasinoHandler) usuarioCasino(c *gin.Context) (models.User, bool) {
	var user models.User
	if err := h.db.First(&user, c.GetUint("userID")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return user, false
	}
	if user.CasinoSeed == "" {
		semilla, _, err := casino.NuevaSemilla()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear la semilla"})
			return user, false
		}
		user.CasinoSeed = semilla
		_ = h.db.Model(&models.User{}).Where("id = ?", user.ID).Update("casino_seed", semilla).Error
	}
	return user, true
}

func (h *CasinoHandler) semillaCliente(c *gin.Context, clientSeed string) string {
	if clientSeed != "" {
		return clientSeed
	}
	var user models.User
	_ = h.db.First(&user, c.GetUint("userID"))
	return "cli-" + strconv.FormatUint(uint64(user.ID), 10) + "-" + strconv.FormatInt(user.CasinoNonce, 10)
}

// apostadoHoy suma lo que el usuario ya aposto hoy en un juego.
func (h *CasinoHandler) apostadoHoy(userID uint, codigo string, desde time.Time) int64 {
	var total int64
	if err := h.db.Model(&models.CasinoRound{}).
		Where("user_id = ? AND game_codigo = ? AND created_at >= ?", userID, codigo, desde).
		Select("COALESCE(SUM(monto),0)").Scan(&total).Error; err != nil {
		// Si la consulta falla, no se puede aplicar el tope: se evita jugar.
		log.Printf("casino: no se pudo calcular el tope diario: %v", err)
		return -1
	}
	if codigo == models.CasinoBlackjack {
		var bj int64
		if err := h.db.Model(&models.BlackjackMano{}).
			Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&bj).Error; err == nil {
			total += bj
		}
	}
	if codigo == models.CasinoBingo {
		var bg int64
		if err := h.db.Model(&models.BingoRonda{}).
			Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&bg).Error; err == nil {
			total += bg
		}
	}
	if codigo == models.CasinoGemas {
		var gm int64
		if err := h.db.Model(&models.GemasRonda{}).
			Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&gm).Error; err == nil {
			total += gm
		}
	}
	if codigo == models.CasinoDomino {
		var dm int64
		if err := h.db.Model(&models.DominoRonda{}).
			Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&dm).Error; err == nil {
			total += dm
		}
	}
	if codigo == models.CasinoAjedrez {
		var aj int64
		if err := h.db.Model(&models.AjedrezRonda{}).
			Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&aj).Error; err == nil {
			total += aj
		}
		var ap int64
		if err := h.db.Model(&models.AjedrezPartida{}).
			Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&ap).Error; err == nil {
			total += ap
		}
	}
	if codigo == models.CasinoBackgammon {
		var bgm int64
		if err := h.db.Model(&models.BackgammonRonda{}).
			Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&bgm).Error; err == nil {
			total += bgm
		}
	}
	if codigo == models.CasinoTetris {
		var tt int64
		if err := h.db.Model(&models.TetrisRonda{}).
			Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&tt).Error; err == nil {
			total += tt
		}
	}
	return total
}

// verificarLimite rechaza la jugada si el usuario ya llego a su tope diario.
// Devuelve true si la jugada puede continuar.
func (h *CasinoHandler) verificarLimite(c *gin.Context, user models.User, codigo string, monto int64) bool {
	var juego models.CasinoGame
	if err := h.db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
		return true
	}
	if juego.LimiteDiario <= 0 {
		return true
	}
	usado := h.apostadoHoy(user.ID, codigo, tiempo.InicioDia())
	if usado < 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo verificar el limite diario"})
		return false
	}
	if usado+monto > juego.LimiteDiario {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      fmt.Sprintf("limite diario alcanzado: llevas %d de %d", usado, juego.LimiteDiario),
			"limite":     juego.LimiteDiario,
			"usado":      usado,
			"disponible": juego.LimiteDiario - usado,
		})
		return false
	}
	return true
}

// BlackjackStart reparte las dos primeras cartas y descuenta la apuesta.
func (h *CasinoHandler) BlackjackStart(c *gin.Context) {
	var req blackjackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	user, ok := h.usuarioCasino(c)
	if !ok {
		return
	}
	if !GuardJugada(c, h.cfg, user) {
		return
	}
	var bjJuego models.CasinoGame
	if err := h.db.Where("codigo = ?", models.CasinoBlackjack).First(&bjJuego).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "el juego no esta configurado"})
		return
	}
	if !bjJuego.Activo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el juego esta desactivado"})
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, models.CasinoBlackjack) {
		return
	}
	if req.Monto < bjJuego.MinApuesta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la apuesta debe ser de al menos " + strconv.FormatInt(bjJuego.MinApuesta, 10)})
		return
	}
	apuestaMax := casino.ApuestaMaxima(bjJuego.MaxApuesta, bjJuego.MaxMultipl, bjJuego.MaxPremio, bjJuego.Codigo)
	if req.Monto > apuestaMax {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el maximo es " + strconv.FormatInt(apuestaMax, 10), "maximo": apuestaMax})
		return
	}
	if user.Saldo < req.Monto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente"})
		return
	}
	if !h.verificarLimite(c, user, models.CasinoBlackjack, req.Monto) {
		return
	}
	if excede, disponible, motivo := EvaluarGasto(h.db, user.ID, req.Monto); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu limite " + motivo + " de gasto se excederia con esta mano",
			"disponible": disponible, "motivo": motivo,
		})
		return
	}

	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	mazo := casino.Baraja(clientSeed, user.CasinoSeed, nonce, 64)

	bj := models.BlackjackMano{
		UserID:         user.ID,
		Monto:          req.Monto,
		Mano:           []int{mazo[0], mazo[2]},
		Crupier:        []int{mazo[1], mazo[3]},
		Estado:         "abierta",
		Orden:          1,
		ClientSeed:     clientSeed,
		ServerSeedHash: casino.HashSemilla(user.CasinoSeed),
		Nonce:          nonce,
	}
	if casino.BlackjackNatural(bj.Mano) {
		bj.Estado = "blackjack"
		bj.Multiplicador = 2.5
		bj.Premio = req.Monto * 5 / 2
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&bj).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"saldo":        gorm.Expr("saldo - ?", req.Monto),
			"casino_nonce": nonce + 1,
		}).Error; err != nil {
			return err
		}
		mov := models.Movimiento{UserID: user.ID, Tipo: models.MovCasinoApuesta, Monto: -req.Monto, Descripcion: "Casino 21 Blackjack"}
		if err := tx.Create(&mov).Error; err != nil {
			return err
		}
		if bj.Premio > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", user.ID).
				Update("saldo", gorm.Expr("saldo + ?", bj.Premio)).Error; err != nil {
				return err
			}
			movPremio := models.Movimiento{UserID: user.ID, Tipo: models.MovCasinoPremio, Monto: bj.Premio, Descripcion: "Premio casino 21 Blackjack"}
			if err := tx.Create(&movPremio).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la mano"})
		return
	}

	// La fila se crea con grupo 0 y se apoda a si misma: asi las divisiones
	// futuras encuentran el grupo sin una consulta extra.
	_ = h.db.Model(&models.BlackjackMano{}).Where("id = ?", bj.ID).
		Update("grupo_id", bj.ID).Error
	bj.GrupoID = bj.ID

	resp := h.blackjackRespuesta(c, &bj, user.ID)
	resp["server_seed_hash"] = bj.ServerSeedHash
	resp["nonce"] = bj.Nonce
	c.JSON(http.StatusCreated, resp)
}


// blackjackGrupo devuelve las filas de un grupo (una mano y sus divisiones)
// ordenadas por Orden.
func (h *CasinoHandler) blackjackGrupo(grupoID uint) []models.BlackjackMano {
	var filas []models.BlackjackMano
	if grupoID == 0 {
		return filas
	}
	_ = h.db.Where("grupo_id = ?", grupoID).Order("orden asc, id asc").Find(&filas).Error
	return filas
}

// blackjackRespuesta arma la respuesta comun de una mano, con el detalle del
// grupo cuando esta dividida: la mano activa, las dos manos y el premio total.
func (h *CasinoHandler) blackjackRespuesta(c *gin.Context, bj *models.BlackjackMano, userID uint) gin.H {
	hermanas := h.blackjackGrupo(bj.GrupoID)
	manos := make([]gin.H, 0, len(hermanas)+1)
	var total int64
	// activa = 0 cuando el grupo ya esta completo; el frente lo interpreta.
	activa := uint(0)
	ordenAbierta := 99
	agregada := false
	for _, m := range hermanas {
		if m.ID == bj.ID {
			m.Mano = bj.Mano
			m.Crupier = bj.Crupier
			m.Estado = bj.Estado
			m.Premio = bj.Premio
			m.Orden = bj.Orden
			agregada = true
		}
		// La mano activa es la primera que sigue abierta (la de menor orden),
		// no la ultima: si hay dos abiertas hay que seguir por la primera.
		if m.Estado == "abierta" && (activa == 0 || m.Orden < ordenAbierta) {
			activa = m.ID
			ordenAbierta = m.Orden
		}
		total += m.Premio
		manos = append(manos, gin.H{
			"id": m.ID, "orden": m.Orden, "mano": m.Mano, "crupier": m.Crupier,
			"estado": m.Estado, "premio": m.Premio, "multiplicador": m.Multiplicador,
		})
	}
	if !agregada {
		manos = append(manos, gin.H{
			"id": bj.ID, "orden": bj.Orden, "mano": bj.Mano, "crupier": bj.Crupier,
			"estado": bj.Estado, "premio": bj.Premio, "multiplicador": bj.Multiplicador,
		})
		if bj.Estado == "abierta" && ordenAbierta == 99 {
			activa = bj.ID
		}
		total += bj.Premio
	}
	var actualizado models.User
	_ = h.db.First(&actualizado, userID)
	return gin.H{
		"mano_id":       bj.ID,
		"mano":          bj.Mano,
		"crupier":       bj.Crupier,
		"estado":        bj.Estado,
		"premio":        bj.Premio,
		"multiplicador": bj.Multiplicador,
		"saldo":         actualizado.Saldo,
		"grupo_id":      bj.GrupoID,
		"activa":        activa,
		"manos":         manos,
		"grupo_premio":  total,
	}
}

// BlackjackSplit divide la mano inicial en dos cuando las dos cartas valen
// igual. Cobra una apuesta adicional igual a la original y reparte una carta
// nueva a cada mano. Ambas se juegan contra el mismo crupier y el premio del
// grupo es la suma de las dos.
func (h *CasinoHandler) BlackjackSplit(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var bj models.BlackjackMano
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&bj).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mano no encontrada"})
		return
	}
	if bj.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la mano ya esta cerrada"})
		return
	}
	if bj.Orden != 1 || len(bj.Mano) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "solo se puede dividir la mano inicial"})
		return
	}
	if !casino.BlackjackMismaValor(bj.Mano[0], bj.Mano[1]) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "las dos cartas deben valer igual para dividir"})
		return
	}

	var user models.User
	_ = h.db.First(&user, userID)
	if user.Saldo < bj.Monto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente para dividir"})
		return
	}
	if !h.verificarLimite(c, user, models.CasinoBlackjack, bj.Monto) {
		return
	}
	if excede, disponible, motivo := EvaluarGasto(h.db, user.ID, bj.Monto); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu limite " + motivo + " de gasto se excederia al dividir",
			"disponible": disponible, "motivo": motivo,
		})
		return
	}

	mazo := casino.Baraja(bj.ClientSeed, user.CasinoSeed, bj.Nonce, 64)
	grupo := bj.GrupoID
	if grupo == 0 {
		grupo = bj.ID
	}
	// Cada mano usa su propio bloque del mazo para no pisarse: la primera sigue
	// en el bloque 0 (posicion 4) y la segunda arranca en el bloque 1 (posicion 20).
	mano1 := []int{bj.Mano[0], mazo[4]}
	mano2 := []int{bj.Mano[1], mazo[20]}

	bj.Mano = mano1
	bj.Orden = 1
	bj.GrupoID = grupo

	var segunda models.BlackjackMano
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&bj).Error; err != nil {
			return err
		}
		segunda = models.BlackjackMano{
			UserID:         userID,
			Monto:          bj.Monto,
			Mano:           mano2,
			Crupier:        append([]int{}, bj.Crupier...),
			Estado:         "abierta",
			Orden:          2,
			GrupoID:        grupo,
			ClientSeed:     bj.ClientSeed,
			ServerSeedHash: bj.ServerSeedHash,
			Nonce:          bj.Nonce,
		}
		if err := tx.Create(&segunda).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("saldo", gorm.Expr("saldo - ?", bj.Monto)).Error; err != nil {
			return err
		}
		mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoApuesta, Monto: -bj.Monto, Descripcion: "Casino 21 Blackjack (dividir)"}
		return tx.Create(&mov).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo dividir la mano"})
		return
	}

	c.JSON(http.StatusOK, h.blackjackRespuesta(c, &bj, userID))
}

func (h *CasinoHandler) BlackjackHit(c *gin.Context)    { h.blackjackTurno(c, "hit") }
func (h *CasinoHandler) BlackjackStand(c *gin.Context)  { h.blackjackTurno(c, "stand") }
func (h *CasinoHandler) BlackjackDouble(c *gin.Context) { h.blackjackTurno(c, "double") }

func (h *CasinoHandler) blackjackTurno(c *gin.Context, accion string) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var bj models.BlackjackMano
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&bj).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mano no encontrada"})
		return
	}
	if bj.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la mano ya esta cerrada"})
		return
	}
	var user models.User
	_ = h.db.First(&user, userID)
	mazo := casino.Baraja(bj.ClientSeed, user.CasinoSeed, bj.Nonce, 64)

	mano := append([]int{}, bj.Mano...)
	crupier := append([]int{}, bj.Crupier...)
	apostado := bj.Monto

	if accion == "double" {
		if user.Saldo < bj.Monto {
			c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente para doblar"})
			return
		}
		if !h.verificarLimite(c, user, models.CasinoBlackjack, bj.Monto) {
			return
		}
		if excede, disponible, motivo := EvaluarGasto(h.db, user.ID, bj.Monto); excede {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":      "tu limite " + motivo + " de gasto se excederia al doblar",
				"disponible": disponible, "motivo": motivo,
			})
			return
		}
		mano = append(mano, mazo[2*len(mano)])
		apostado = bj.Monto * 2
	} else if accion == "hit" {
		mano = append(mano, mazo[2*len(mano)])
	}

	valor, _, pasado := casino.ValorBlackjack(mano)

	// Con hit y menos de 21 la mano sigue abierta.
	if accion == "hit" && !pasado && valor < 21 {
		bj.Mano = mano
		if err := h.db.Save(&bj).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la carta"})
			return
		}
		bj.Mano = mano
		c.JSON(http.StatusOK, h.blackjackRespuesta(c, &bj, userID))
		return
	}

	// Cierra la mano: el crupier pide hasta 17.
	for {
		v, _, _ := casino.ValorBlackjack(crupier)
		if v >= 17 || len(crupier) >= 12 {
			break
		}
		crupier = append(crupier, mazo[2*len(crupier)+1])
	}

	estado, premio, mult := casino.BlackjackResultado(mano, crupier, apostado)
	bj.Mano = mano
	bj.Crupier = crupier
	bj.Estado = estado
	bj.Premio = premio
	bj.Multiplicador = mult
	// Si la mano esta dividida, el crupier completo es de las dos: la hermana
	// que sigue abierta lo hereda.
	hermanas := h.blackjackGrupo(bj.GrupoID)

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if accion == "double" {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("saldo", gorm.Expr("saldo - ?", bj.Monto)).Error; err != nil {
				return err
			}
			mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoApuesta, Monto: -bj.Monto, Descripcion: "Casino 21 Blackjack (doblar)"}
			if err := tx.Create(&mov).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(&bj).Error; err != nil {
			return err
		}
		for i := range hermanas {
			if hermanas[i].ID == bj.ID || hermanas[i].Estado != "abierta" {
				continue
			}
			if err := tx.Model(&models.BlackjackMano{}).Where("id = ?", hermanas[i].ID).
				Update("crupier_raw", bj.CrupierRaw).Error; err != nil {
				return err
			}
		}
		if premio > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
				return err
			}
			movPremio := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: premio, Descripcion: "Premio casino 21 Blackjack"}
			if err := tx.Create(&movPremio).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo resolver la mano"})
		return
	}

	c.JSON(http.StatusOK, h.blackjackRespuesta(c, &bj, userID))
}

// BlackjackSurrender cierra la mano devolviendo la mitad de la apuesta. Solo se
// permite con la mano inicial (dos cartas), antes de pedir o doblar.
func (h *CasinoHandler) BlackjackSurrender(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var bj models.BlackjackMano
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&bj).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mano no encontrada"})
		return
	}
	if bj.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la mano ya esta cerrada"})
		return
	}
	if len(bj.Mano) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "solo se puede rendir con la mano inicial"})
		return
	}

	premio := casino.BlackjackRendicion(bj.Monto)
	bj.Estado = "rendida"
	bj.Premio = premio
	bj.Multiplicador = 0.5

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&bj).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
			return err
		}
		mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: premio, Descripcion: "Casino 21 Blackjack (rendicion)"}
		return tx.Create(&mov).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo resolver la rendicion"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, userID)
	c.JSON(http.StatusOK, gin.H{
		"mano_id": bj.ID,
		"mano":    bj.Mano,
		"crupier": bj.Crupier,
		"estado":  bj.Estado,
		"premio":  bj.Premio,
		"saldo":   actualizado.Saldo,
	})
}

// BlackjackInsurance cobra el seguro (mitad de la apuesta) cuando el crupier
// muestra un As. Si el crupier tiene blackjack natural, el seguro paga 2:1 y el
// jugador recupera su apuesta; si no, pierde el seguro y la mano sigue abierta.
func (h *CasinoHandler) BlackjackInsurance(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var bj models.BlackjackMano
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&bj).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mano no encontrada"})
		return
	}
	if bj.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la mano ya esta cerrada"})
		return
	}
	if bj.Seguro > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el seguro ya fue tomado"})
		return
	}
	if !casino.SeguroAplica(bj.Crupier) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el seguro solo aplica con As del crupier"})
		return
	}

	var user models.User
	_ = h.db.First(&user, userID)
	costo := casino.SeguroCosto(bj.Monto)
	if user.Saldo < costo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente para el seguro"})
		return
	}
	if !h.verificarLimite(c, user, models.CasinoBlackjack, costo) {
		return
	}

	natural := casino.BlackjackNatural(bj.Crupier)
	bj.Seguro = costo

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("saldo", gorm.Expr("saldo - ?", costo)).Error; err != nil {
			return err
		}
		mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoApuesta, Monto: -costo, Descripcion: "Casino 21 Blackjack (seguro)"}
		if err := tx.Create(&mov).Error; err != nil {
			return err
		}
		if natural {
			bj.Estado = "seguro"
			bj.Premio = casino.SeguroPaga(bj.Monto)
			if err := tx.Save(&bj).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("saldo", gorm.Expr("saldo + ?", bj.Premio)).Error; err != nil {
				return err
			}
			movPremio := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: bj.Premio, Descripcion: "Premio casino 21 Blackjack (seguro)"}
			return tx.Create(&movPremio).Error
		}
		// Sin natural: el seguro se pierde y la mano continua.
		return tx.Save(&bj).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo tomar el seguro"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, userID)
	c.JSON(http.StatusOK, gin.H{
		"mano_id": bj.ID,
		"mano":    bj.Mano,
		"crupier": bj.Crupier,
		"estado":  bj.Estado,
		"premio":  bj.Premio,
		"seguro":  bj.Seguro,
		"saldo":   actualizado.Saldo,
	})
}
