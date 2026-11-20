package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Backgammon simplificado: carrera de tres fichas por lado. La apuesta se cobra
// al abrir la carrera y se liquida cuando un lado saca las tres fichas.

type backgammonRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	ClientSeed string `json:"client_seed"`
}

type backgammonMoveRequest struct {
	Ficha int `json:"ficha"`
}

func backgammonResumen(ronda models.BackgammonRonda, saldo int64) gin.H {
	return gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"jugador":          ronda.Jugador,
		"casa":             ronda.Casa,
		"turno":            ronda.Turno,
		"dado1":            ronda.Dado1,
		"dado2":            ronda.Dado2,
		"dobles":           ronda.Dobles,
		"lance":            ronda.Lance,
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

// BackgammonStart cobra la apuesta y tira los dados iniciales.
func (h *CasinoHandler) BackgammonStart(c *gin.Context) {
	var req backgammonRequest
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
	juego, ok := h.nuevosJuego(c, models.CasinoBackgammon)
	if !ok {
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, models.CasinoBackgammon) {
		return
	}
	if !h.nuevosApuesta(c, user, juego, req.Monto) {
		return
	}
	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	est := casino.NuevoBackgammon(clientSeed, user.CasinoSeed, nonce)
	// Si la casa empieza, juega su turno antes de devolver el estado.
	if est.Turno == "casa" {
		casino.BackgammonCasaJuega(est)
		if !est.Fin {
			casino.BackgammonSiguiente(est, clientSeed, user.CasinoSeed, nonce)
		}
	}
	ronda := models.BackgammonRonda{
		UserID:         user.ID,
		Monto:          req.Monto,
		Jugador:        est.Jugador,
		Casa:           est.Casa,
		Turno:          est.Turno,
		Dado1:          est.Dado1,
		Dado2:          est.Dado2,
		Dobles:         est.Dobles,
		Lance:          est.Lance,
		Fin:            est.Fin,
		Ganador:        est.Ganador,
		Jugadas:        []int{},
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
	c.JSON(http.StatusCreated, backgammonResumen(ronda, actualizado.Saldo))
}

// BackgammonMove avanza una ficha del jugador y responde la casa.
func (h *CasinoHandler) BackgammonMove(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.BackgammonRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	if ronda.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la jugada ya esta cerrada"})
		return
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	var req backgammonMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	est := casino.NuevoBackgammon(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce)
	est.Jugador = ronda.Jugador
	est.Casa = ronda.Casa
	est.Turno = ronda.Turno
	est.Dado1, est.Dado2 = ronda.Dado1, ronda.Dado2
	est.Dobles, est.Lance = ronda.Dobles, ronda.Lance
	if est.Turno != "jugador" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no es tu turno"})
		return
	}
	if err := casino.BackgammonJugadorJuega(est, req.Ficha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ronda.Jugadas = append(ronda.Jugadas, req.Ficha)
	ronda.Jugador, ronda.Casa = est.Jugador, est.Casa
	ronda.Fin, ronda.Ganador = est.Fin, est.Ganador
	if est.Fin {
		h.cerrarBackgammon(c, &ronda, userID)
		return
	}
	casino.BackgammonSiguiente(est, ronda.ClientSeed, user.CasinoSeed, ronda.Nonce)
	casino.BackgammonCasaJuega(est)
	ronda.Dado1, ronda.Dado2 = est.Dado1, est.Dado2
	ronda.Dobles, ronda.Lance = est.Dobles, est.Lance
	ronda.Turno = est.Turno
	ronda.Jugador, ronda.Casa = est.Jugador, est.Casa
	ronda.Fin, ronda.Ganador = est.Fin, est.Ganador
	if est.Fin {
		h.cerrarBackgammon(c, &ronda, userID)
		return
	}
	casino.BackgammonSiguiente(est, ronda.ClientSeed, user.CasinoSeed, ronda.Nonce)
	ronda.Dado1, ronda.Dado2 = est.Dado1, est.Dado2
	ronda.Dobles, ronda.Lance = est.Dobles, est.Lance
	ronda.Turno = est.Turno
	if err := h.db.Save(&ronda).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la jugada"})
		return
	}
	var actualizado models.User
	_ = h.db.First(&actualizado, userID)
	c.JSON(http.StatusOK, backgammonResumen(ronda, actualizado.Saldo))
}

func (h *CasinoHandler) cerrarBackgammon(c *gin.Context, ronda *models.BackgammonRonda, userID uint) {
	ronda.Estado = "pagada"
	ronda.Premio = casino.BackgammonPremio(ronda.Monto, ronda.Ganador)
	ronda.Multiplicador = casino.BackgammonMultiplicador(ronda.Ganador)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(ronda).Error; err != nil {
			return err
		}
		return h.nuevosPagar(tx, userID, ronda.Premio, "Backgammon")
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar la jugada"})
		return
	}
	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, backgammonResumen(*ronda, user.Saldo))
}

// BackgammonVerificar reproduce la carrera.
func (h *CasinoHandler) BackgammonVerificar(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.BackgammonRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	rep, err := casino.BackgammonRepetir(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Jugadas)
	verificado := err == nil &&
		casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		rep.Ganador == ronda.Ganador &&
		ronda.Premio == casino.BackgammonPremio(ronda.Monto, rep.Ganador)
	c.JSON(http.StatusOK, gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"jugador":          ronda.Jugador,
		"casa":             ronda.Casa,
		"ganador":          ronda.Ganador,
		"premio":           ronda.Premio,
		"multiplicador":    ronda.Multiplicador,
		"jugadas":          ronda.Jugadas,
		"client_seed":      ronda.ClientSeed,
		"server_seed_hash": ronda.ServerSeedHash,
		"server_seed":      user.CasinoSeed,
		"nonce":            ronda.Nonce,
		"verificado":       verificado,
	})
}
