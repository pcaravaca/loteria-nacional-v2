package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Tetris por lineas: la apuesta se cobra al abrir la ronda y el jugador coloca
// las piezas de la secuencia; el pago depende de las lineas acumuladas.

type tetrisRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	ClientSeed string `json:"client_seed"`
}

type tetrisMoveRequest struct {
	Columna  int `json:"columna"`
	Rotacion int `json:"rotacion"`
}

func tetrisResumen(ronda models.TetrisRonda, saldo int64) gin.H {
	return gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"tablero":          ronda.Tablero,
		"piezas":           ronda.Piezas,
		"pieza":            ronda.Pieza,
		"piezas_totales":   casino.TetrisPiezas,
		"lineas":           ronda.Lineas,
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

// TetrisStart cobra la apuesta y arma la secuencia de piezas.
func (h *CasinoHandler) TetrisStart(c *gin.Context) {
	var req tetrisRequest
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
	juego, ok := h.nuevosJuego(c, models.CasinoTetris)
	if !ok {
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, models.CasinoTetris) {
		return
	}
	if !h.nuevosApuesta(c, user, juego, req.Monto) {
		return
	}
	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	seq := casino.TetrisSecuencia(clientSeed, user.CasinoSeed, nonce)
	ronda := models.TetrisRonda{
		UserID:         user.ID,
		Monto:          req.Monto,
		Tablero:        make([]int, casino.TetrisFilas*casino.TetrisColumnas),
		Piezas:         seq,
		Pieza:          0,
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
	c.JSON(http.StatusCreated, tetrisResumen(ronda, actualizado.Saldo))
}

// TetrisMove coloca la pieza actual en la columna y rotacion elegidas.
func (h *CasinoHandler) TetrisMove(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.TetrisRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	if ronda.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la jugada ya esta cerrada"})
		return
	}
	var req tetrisMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	if req.Rotacion < 0 || req.Rotacion > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rotacion invalida"})
		return
	}
	tab := append([]int{}, ronda.Tablero...)
	lim, ok := casino.TetrisSoltar(tab, ronda.Piezas, ronda.Pieza, req.Columna, req.Rotacion)
	if !ok {
		// El pozo se cerro antes de terminar la secuencia: se liquida.
		h.cerrarTetris(c, &ronda, userID)
		return
	}
	ronda.Tablero = tab
	ronda.Jugadas = append(ronda.Jugadas, [2]int{req.Columna, req.Rotacion})
	ronda.Pieza++
	ronda.Lineas += lim
	if ronda.Pieza >= casino.TetrisPiezas {
		h.cerrarTetris(c, &ronda, userID)
		return
	}
	if err := h.db.Save(&ronda).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la jugada"})
		return
	}
	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, tetrisResumen(ronda, user.Saldo))
}

func (h *CasinoHandler) cerrarTetris(c *gin.Context, ronda *models.TetrisRonda, userID uint) {
	ronda.Estado = "pagada"
	ronda.Premio = casino.TetrisPremio(ronda.Monto, ronda.Lineas)
	ronda.Multiplicador = casino.TetrisMult(ronda.Lineas)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(ronda).Error; err != nil {
			return err
		}
		return h.nuevosPagar(tx, userID, ronda.Premio, "Tetris")
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar la jugada"})
		return
	}
	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, tetrisResumen(*ronda, user.Saldo))
}

// TetrisVerificar reproduce el tablero y las lineas.
func (h *CasinoHandler) TetrisVerificar(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.TetrisRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	tab, lineas, err := casino.TetrisRepetir(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Jugadas)
	verificado := err == nil &&
		casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		lineas == ronda.Lineas &&
		tetrisIgualar(tab, ronda.Tablero) &&
		ronda.Premio == casino.TetrisPremio(ronda.Monto, lineas)
	c.JSON(http.StatusOK, gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"tablero":          ronda.Tablero,
		"piezas":           ronda.Piezas,
		"lineas":           ronda.Lineas,
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

func tetrisIgualar(a, b []int) bool {
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
