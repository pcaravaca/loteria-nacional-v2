package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
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
