package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// Ajedrez relampago: la casa propone una posicion de mate en una y el jugador
// elige la jugada ganadora entre cuatro opciones. Se apuesta al abrir la
// posicion y se liquida al elegir.

type ajedrezRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	ClientSeed string `json:"client_seed"`
}

type ajedrezMoveRequest struct {
	Opcion int `json:"opcion"`
}

func ajedrezOpciones(fen string, opciones []string) []gin.H {
	out := make([]gin.H, 0, len(opciones))
	for _, uci := range opciones {
		out = append(out, gin.H{"uci": uci, "etiqueta": casino.AjedrezEtiqueta(fen, uci)})
	}
	return out
}

func ajedrezResumen(ronda models.AjedrezRonda, saldo int64) gin.H {
	return gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"fen":              ronda.FEN,
		"tema":             ronda.Tema,
		"opciones":         ajedrezOpciones(ronda.FEN, ronda.Opciones),
		"estado":           ronda.Estado,
		"multiplicador":    ronda.Multiplicador,
		"premio":           ronda.Premio,
		"premio_pagado":    ronda.Premio,
		"jugadas":          ronda.Jugadas,
		"saldo":            saldo,
		"server_seed_hash": ronda.ServerSeedHash,
		"nonce":            ronda.Nonce,
	}
}

// AjedrezStart cobra la apuesta y propone la posicion.
func (h *CasinoHandler) AjedrezStart(c *gin.Context) {
	var req ajedrezRequest
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
	juego, ok := h.nuevosJuego(c, models.CasinoAjedrez)
	if !ok {
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, models.CasinoAjedrez) {
		return
	}
	if !h.nuevosApuesta(c, user, juego, req.Monto) {
		return
	}
	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	p, opciones := casino.AjedrezElegir(clientSeed, user.CasinoSeed, nonce)
	ronda := models.AjedrezRonda{
		UserID:         user.ID,
		Monto:          req.Monto,
		FEN:            p.FEN,
		Correcta:       p.Correcta,
		Opciones:       opciones,
		Tema:           p.Tema,
		Jugadas:        []string{},
		Estado:         "abierta",
		ClientSeed:     clientSeed,
		ServerSeedHash: casino.HashSemilla(user.CasinoSeed),
		Nonce:          nonce,
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ronda).Error; err != nil {
			return err
		}
		return h.nuevosCobrar(tx, user, juego, req.Monto, nonce)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la jugada"})
		return
	}
	var actualizado models.User
	_ = h.db.First(&actualizado, user.ID)
	c.JSON(http.StatusCreated, ajedrezResumen(ronda, actualizado.Saldo))
}

// AjedrezMove elige una de las cuatro jugadas y liquida la ronda.
func (h *CasinoHandler) AjedrezMove(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.AjedrezRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	if ronda.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la jugada ya esta cerrada"})
		return
	}
	var req ajedrezMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	if req.Opcion < 0 || req.Opcion >= len(ronda.Opciones) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "opcion invalida"})
		return
	}
	elegida := ronda.Opciones[req.Opcion]
	ronda.Jugadas = []string{elegida}
	acierto := elegida == ronda.Correcta
	ronda.Estado = "pagada"
	ronda.Premio = casino.AjedrezPremio(ronda.Monto, acierto)
	ronda.Multiplicador = casino.AjedrezMultiplicador(acierto)
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ronda).Error; err != nil {
			return err
		}
		return h.nuevosPagar(tx, userID, ronda.Premio, "Ajedrez")
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar la jugada"})
		return
	}
	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, ajedrezResumen(ronda, user.Saldo))
}

// AjedrezVerificar reproduce la posicion y el acierto.
func (h *CasinoHandler) AjedrezVerificar(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.AjedrezRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	p, opciones, acierto := casino.AjedrezRepetir(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Jugadas)
	verificado := casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		p.FEN == ronda.FEN &&
		ronda.Premio == casino.AjedrezPremio(ronda.Monto, acierto)
	c.JSON(http.StatusOK, gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"fen":              ronda.FEN,
		"correcta":         ronda.Correcta,
		"opciones":         opciones,
		"jugadas":          ronda.Jugadas,
		"premio":           ronda.Premio,
		"multiplicador":    ronda.Multiplicador,
		"client_seed":      ronda.ClientSeed,
		"server_seed_hash": ronda.ServerSeedHash,
		"server_seed":      user.CasinoSeed,
		"nonce":            ronda.Nonce,
		"verificado":       verificado,
	})
}

// ============================================================================
// Partida completa de ajedrez contra el adversario de la casa.
// ============================================================================
//
// Mismo esquema que los demas juegos de mesa: la apuesta se cobra al abrir, la
// partida queda abierta en su tabla y cada turno se reconstruye desde la
// posicion inicial. La casa responde con AjedrezJugadaCasa, una funcion pura
// de la posicion y las semillas, de modo que el verificador la reproduce.

type ajedrezPartidaRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	ClientSeed string `json:"client_seed"`
	Color      string `json:"color"`
}

type ajedrezPartidaMoveRequest struct {
	UCI string `json:"uci" binding:"required"`
}

func ajedrezResumenPartida(p models.AjedrezPartida, j *casino.AjedrezJuego, saldo int64) gin.H {
	ultimo := ""
	if len(p.Jugadas) > 0 {
		ultimo = p.Jugadas[len(p.Jugadas)-1]
	}
	return gin.H{
		"partida_id":       p.ID,
		"monto":            p.Monto,
		"color":            p.ColorJugador,
		"fen":              p.FEN,
		"turno":            p.Turno,
		"jugadas":          p.Jugadas,
		"ultimo":           ultimo,
		"legales":          j.LegalesUCI(),
		"estado":           p.Estado,
		"resultado":        p.Resultado,
		"motivo":           p.Motivo,
		"multiplicador":    p.Multiplicador,
		"premio":           p.Premio,
		"premio_pagado":    p.Premio,
		"saldo":            saldo,
		"server_seed_hash": p.ServerSeedHash,
		"nonce":            p.Nonce,
	}
}

// ajedrezLimitePartidas extiende el tope diario a la tabla de partidas
// completas, que no esta incluida en nuevosApostadoHoy.
func (h *CasinoHandler) ajedrezLimitePartidas(c *gin.Context, user models.User, juego models.CasinoGame, monto int64) bool {
	if juego.LimiteDiario <= 0 {
		return true
	}
	desde := tiempo.InicioDia()
	usado := h.nuevosApostadoHoy(user.ID, models.CasinoAjedrez, desde)
	if usado < 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo verificar el limite diario"})
		return false
	}
	var partidas int64
	if err := h.db.Model(&models.AjedrezPartida{}).
		Where("user_id = ? AND created_at >= ?", user.ID, desde).
		Select("COALESCE(SUM(monto),0)").Scan(&partidas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo verificar el limite diario"})
		return false
	}
	if usado+partidas+monto > juego.LimiteDiario {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "limite diario alcanzado",
			"limite":     juego.LimiteDiario,
			"usado":      usado + partidas,
			"disponible": juego.LimiteDiario - usado - partidas,
		})
		return false
	}
	return true
}

// AjedrezPartidaStart cobra la apuesta y abre la partida completa. Si el
// jugador elige negras, la casa mueve primero.
func (h *CasinoHandler) AjedrezPartidaStart(c *gin.Context) {
	var req ajedrezPartidaRequest
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
	juego, ok := h.nuevosJuego(c, models.CasinoAjedrez)
	if !ok {
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, models.CasinoAjedrez) {
		return
	}
	if !h.nuevosApuesta(c, user, juego, req.Monto) {
		return
	}
	if !h.ajedrezLimitePartidas(c, user, juego, req.Monto) {
		return
	}
	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	color := "blancas"
	if req.Color == "negras" {
		color = "negras"
	}
	j := casino.NuevoAjedrezJuego()
	if color == "negras" {
		if m, ok := casino.AjedrezJugadaCasa(clientSeed, user.CasinoSeed, nonce, &j.Posicion, 0); ok {
			j.AplicarUCI(m.UCI())
		}
	}
	partida := models.AjedrezPartida{
		UserID:         user.ID,
		Monto:          req.Monto,
		ColorJugador:   color,
		FEN:            j.Posicion.AFEN(),
		Jugadas:        j.Jugadas,
		Turno:          string(j.Posicion.Turno),
		Estado:         "abierta",
		ClientSeed:     clientSeed,
		ServerSeedHash: casino.HashSemilla(user.CasinoSeed),
		Nonce:          nonce,
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&partida).Error; err != nil {
			return err
		}
		return h.nuevosCobrar(tx, user, juego, req.Monto, nonce)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la partida"})
		return
	}
	var actualizado models.User
	_ = h.db.First(&actualizado, user.ID)
	c.JSON(http.StatusCreated, ajedrezResumenPartida(partida, &j, actualizado.Saldo))
}

// AjedrezPartidaMove aplica la jugada del jugador, responde la casa y liquida
// si la partida termina.
func (h *CasinoHandler) AjedrezPartidaMove(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var partida models.AjedrezPartida
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&partida).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la partida no existe"})
		return
	}
	if partida.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la partida ya esta cerrada"})
		return
	}
	var req ajedrezPartidaMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	j, ok := casino.AjedrezReconstruir(partida.Jugadas)
	if !ok || j.Posicion.AFEN() != partida.FEN {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "la partida no se puede reproducir"})
		return
	}
	if (j.Posicion.Turno == 'w') != (partida.ColorJugador == "blancas") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no es tu turno"})
		return
	}
	if !j.AplicarUCI(req.UCI) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jugada ilegal"})
		return
	}
	if j.Resultado == "" {
		var user models.User
		if err := h.db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar el usuario"})
			return
		}
		if m, ok := casino.AjedrezJugadaCasa(partida.ClientSeed, user.CasinoSeed, partida.Nonce, &j.Posicion, len(j.Jugadas)); ok {
			j.AplicarUCI(m.UCI())
		}
	}
	if j.Resultado != "" {
		partida.Estado = "pagada"
		partida.Resultado = j.Resultado
		partida.Motivo = j.Motivo
		partida.Multiplicador = casino.AjedrezPartidaMultiplicador(j.Resultado, partida.ColorJugador)
		partida.Premio = casino.AjedrezPartidaPremio(partida.Monto, j.Resultado, partida.ColorJugador)
	}
	partida.FEN = j.Posicion.AFEN()
	partida.Jugadas = j.Jugadas
	partida.Turno = string(j.Posicion.Turno)
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&partida).Error; err != nil {
			return err
		}
		return h.nuevosPagar(tx, userID, partida.Premio, "Ajedrez")
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la jugada"})
		return
	}
	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, ajedrezResumenPartida(partida, &j, user.Saldo))
}

// AjedrezPartidaVerificar reconstruye la partida y reproduce la jugada de la
// casa en cada turno desde las semillas reveladas.
func (h *CasinoHandler) AjedrezPartidaVerificar(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var partida models.AjedrezPartida
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&partida).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la partida no existe"})
		return
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	j, reconstruido := casino.AjedrezReconstruir(partida.Jugadas)
	coincide := reconstruido && j.Posicion.AFEN() == partida.FEN && j.Resultado == partida.Resultado

	pos := casino.NuevoAjedrezJuego()
	casaOK := true
	for ply, uci := range partida.Jugadas {
		esCasa := (pos.Posicion.Turno == 'w') != (partida.ColorJugador == "blancas")
		if esCasa {
			m, ok := casino.AjedrezJugadaCasa(partida.ClientSeed, user.CasinoSeed, partida.Nonce, &pos.Posicion, ply)
			if !ok || m.UCI() != uci {
				casaOK = false
				break
			}
		}
		if !pos.AplicarUCI(uci) {
			casaOK = false
			break
		}
	}
	premioOK := partida.Premio == casino.AjedrezPartidaPremio(partida.Monto, partida.Resultado, partida.ColorJugador)
	multOK := partida.Multiplicador == casino.AjedrezPartidaMultiplicador(partida.Resultado, partida.ColorJugador)
	hashOK := casino.HashSemilla(user.CasinoSeed) == partida.ServerSeedHash
	verificado := coincide && casaOK && premioOK && multOK && hashOK

	c.JSON(http.StatusOK, gin.H{
		"partida_id":       partida.ID,
		"monto":            partida.Monto,
		"color":            partida.ColorJugador,
		"fen":              partida.FEN,
		"resultado":        partida.Resultado,
		"motivo":           partida.Motivo,
		"jugadas":          partida.Jugadas,
		"premio":           partida.Premio,
		"multiplicador":    partida.Multiplicador,
		"client_seed":      partida.ClientSeed,
		"server_seed_hash": partida.ServerSeedHash,
		"server_seed":      user.CasinoSeed,
		"nonce":            partida.Nonce,
		"verificado":       verificado,
		"detalle": gin.H{
			"hash":       hashOK,
			"posicion":   coincide,
			"casa":       casaOK,
			"premio":     premioOK,
			"multipl":    multOK,
		},
	})
}
