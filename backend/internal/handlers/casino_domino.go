package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Domino: la apuesta se descuenta al abrir la mano y el jugador y la casa juegan
// fichas por turnos hasta que uno se queda sin fichas o ambos pasan dos veces.

type dominoRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	ClientSeed string `json:"client_seed"`
}

type dominoMoveRequest struct {
	Ficha int `json:"ficha"`
}

func dominoManoA(mano []casino.DominoFicha) [][2]int {
	out := make([][2]int, 0, len(mano))
	for _, f := range mano {
		out = append(out, [2]int{f[0], f[1]})
	}
	return out
}

func dominoManoDe(mano [][2]int) []casino.DominoFicha {
	out := make([]casino.DominoFicha, 0, len(mano))
	for _, f := range mano {
		out = append(out, casino.DominoFicha{f[0], f[1]})
	}
	return out
}

func dominoResumen(ronda models.DominoRonda, saldo int64, revelar bool) gin.H {
	izq, der := 0, 0
	if len(ronda.Tablero) > 0 {
		izq, der = ronda.Tablero[0][0], ronda.Tablero[len(ronda.Tablero)-1][1]
	}
	casa := ronda.Casa
	if !revelar {
		casa = nil
	}
	return gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"mano":             ronda.Mano,
		"casa":             casa,
		"casa_fichas":      len(ronda.Casa),
		"tablero":          ronda.Tablero,
		"extremos":         [2]int{izq, der},
		"turno":            ronda.Turno,
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

// DominoStart cobra la apuesta y reparte la mano.
func (h *CasinoHandler) DominoStart(c *gin.Context) {
	var req dominoRequest
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
	juego, ok := h.nuevosJuego(c, models.CasinoDomino)
	if !ok {
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, models.CasinoDomino) {
		return
	}
	if !h.nuevosApuesta(c, user, juego, req.Monto) {
		return
	}
	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	mano, casa := casino.DominoSorteo(clientSeed, user.CasinoSeed, nonce)
	ronda := models.DominoRonda{
		UserID:         user.ID,
		Monto:          req.Monto,
		Mano:           dominoManoA(mano),
		Casa:           dominoManoA(casa),
		Tablero:        [][2]int{},
		Turno:          "jugador",
		Jugadas:        [][2]int{},
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
	c.JSON(http.StatusCreated, dominoResumen(ronda, actualizado.Saldo, false))
}

// DominoMove aplica la ficha elegida y la respuesta de la casa.
func (h *CasinoHandler) DominoMove(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.DominoRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	if ronda.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la jugada ya esta cerrada"})
		return
	}
	var req dominoMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	est := casino.NuevoDominoEstadoDesde(dominoManoDe(ronda.Mano), dominoManoDe(ronda.Casa), dominoManoDe(ronda.Tablero), ronda.Turno)
	if err := casino.DominoJugadorJuega(est, req.Ficha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ronda.Jugadas = append(ronda.Jugadas, [2]int{req.Ficha, 0})
	ronda.Mano = dominoManoA(est.Mano)
	ronda.Casa = dominoManoA(est.Casa)
	ronda.Tablero = dominoManoA(est.Tablero)
	ronda.Turno = est.Turno
	if est.Fin {
		h.cerrarDomino(c, &ronda, userID)
		return
	}
	casino.DominoCasaJuega(est)
	ronda.Mano = dominoManoA(est.Mano)
	ronda.Casa = dominoManoA(est.Casa)
	ronda.Tablero = dominoManoA(est.Tablero)
	ronda.Turno = est.Turno
	if est.Fin {
		h.cerrarDomino(c, &ronda, userID)
		return
	}
	if err := h.db.Save(&ronda).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la jugada"})
		return
	}
	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, dominoResumen(ronda, user.Saldo, false))
}

func (h *CasinoHandler) cerrarDomino(c *gin.Context, ronda *models.DominoRonda, userID uint) {
	ganador := casino.DominoGanador(dominoManoDe(ronda.Mano), dominoManoDe(ronda.Casa))
	ronda.Estado = "pagada"
	ronda.Premio = casino.DominoPremio(ronda.Monto, ganador)
	ronda.Multiplicador = casino.DominoMultiplicador(ganador)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(ronda).Error; err != nil {
			return err
		}
		return h.nuevosPagar(tx, userID, ronda.Premio, "Domino")
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar la jugada"})
		return
	}
	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, dominoResumen(*ronda, user.Saldo, true))
}

// DominoVerificar revela la semilla y reproduce la mano.
func (h *CasinoHandler) DominoVerificar(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.DominoRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	rep, err := casino.DominoRepetir(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Jugadas)
	verificado := err == nil &&
		casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		dominoIgualar(dominoManoA(rep.Mano), ronda.Mano) &&
		dominoIgualar(dominoManoA(rep.Casa), ronda.Casa) &&
		dominoIgualar(dominoManoA(rep.Tablero), ronda.Tablero)
	c.JSON(http.StatusOK, gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"premio":           ronda.Premio,
		"multiplicador":    ronda.Multiplicador,
		"jugadas":          ronda.Jugadas,
		"mano":             ronda.Mano,
		"casa":             ronda.Casa,
		"tablero":          ronda.Tablero,
		"client_seed":      ronda.ClientSeed,
		"server_seed_hash": ronda.ServerSeedHash,
		"server_seed":      user.CasinoSeed,
		"nonce":            ronda.Nonce,
		"verificado":       verificado,
	})
}

func dominoIgualar(a, b [][2]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
