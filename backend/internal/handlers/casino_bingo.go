package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Bingo 75: la apuesta se descuenta al abrir la ronda y las bolillas se cantan
// de a una. El premio se calcula por las líneas completadas en los cartones; el
// orden de las bolillas y los cartones salen de las semillas (provably fair).

type bingoRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	Cartones   int    `json:"cartones"`
	ClientSeed string `json:"client_seed"`
}

// bingoJuego carga la configuración del juego y verifica que esté activo.
func (h *CasinoHandler) bingoJuego(c *gin.Context) (models.CasinoGame, bool) {
	var juego models.CasinoGame
	if err := h.db.Where("codigo = ?", models.CasinoBingo).First(&juego).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "el juego no esta configurado"})
		return juego, false
	}
	if !juego.Activo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el juego esta desactivado"})
		return juego, false
	}
	return juego, true
}

// bingoResumen arma la respuesta del estado de la ronda. `premio` es lo que se
// pagaría con las bolillas ya cantadas; `premio_pagado` es lo que ya se acreditó
// al cerrar.
func bingoResumen(ronda models.BingoRonda, saldo int64, bolilla int) gin.H {
	premio, lineas, lleno, tiers := casino.BingoPremio(ronda.MontoPorCarton, ronda.Grids, ronda.Cantadas)
	return gin.H{
		"ronda_id":          ronda.ID,
		"monto":             ronda.Monto,
		"monto_por_carton":  ronda.MontoPorCarton,
		"cartones":          ronda.Cartones,
		"grids":             ronda.Grids,
		"orden":             ronda.Orden,
		"cantadas":          ronda.Cantadas,
		"bolilla":           bolilla,
		"cantidad_cantadas": casino.BingoCantadas,
		"estado":            ronda.Estado,
		"lineas":            lineas,
		"lleno":             lleno,
		"tiers":             tiers,
		"premio":            premio,
		"premio_pagado":     ronda.Premio,
		"multiplicador":     ronda.Multiplicador,
		"saldo":             saldo,
		"server_seed_hash":  ronda.ServerSeedHash,
		"nonce":             ronda.Nonce,
	}
}

// BingoStart cobra la apuesta, genera los cartones y devuelve el orden de las
// bolillas (que el cliente no necesita Mostrar: las revela `hit` de a una).
func (h *CasinoHandler) BingoStart(c *gin.Context) {
	var req bingoRequest
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
	juego, ok := h.bingoJuego(c)
	if !ok {
		return
	}
	if req.Cartones < 1 {
		req.Cartones = 1
	}
	if req.Cartones > casino.BingoMaxCartones {
		req.Cartones = casino.BingoMaxCartones
	}
	if req.Monto < juego.MinApuesta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la apuesta debe ser de al menos " + strconv.FormatInt(juego.MinApuesta, 10)})
		return
	}
	apuestaMax := casino.ApuestaMaxima(juego.MaxApuesta, juego.MaxMultipl, juego.MaxPremio, juego.Codigo)
	if req.Monto > apuestaMax {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el maximo es " + strconv.FormatInt(apuestaMax, 10), "maximo": apuestaMax})
		return
	}
	total := req.Monto * int64(req.Cartones)
	if user.Saldo < total {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente"})
		return
	}
	if !h.verificarLimite(c, user, models.CasinoBingo, total) {
		return
	}
	if excede, disponible, motivo := EvaluarGasto(h.db, user.ID, total); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu limite " + motivo + " de gasto se excederia con esta jugada",
			"disponible": disponible, "motivo": motivo,
		})
		return
	}

	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	orden, grids := casino.BingoSorteo(clientSeed, user.CasinoSeed, nonce, req.Cartones)

	ronda := models.BingoRonda{
		UserID:         user.ID,
		Monto:          total,
		MontoPorCarton: req.Monto,
		Cartones:       req.Cartones,
		Grids:          grids,
		Orden:          orden,
		Cantadas:       []int{},
		Estado:         "abierta",
		ClientSeed:     clientSeed,
		ServerSeedHash: casino.HashSemilla(user.CasinoSeed),
		Nonce:          nonce,
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ronda).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"saldo":        gorm.Expr("saldo - ?", total),
			"casino_nonce": nonce + 1,
		}).Error; err != nil {
			return err
		}
		mov := models.Movimiento{UserID: user.ID, Tipo: models.MovCasinoApuesta, Monto: -total, Descripcion: "Casino " + juego.Nombre}
		return tx.Create(&mov).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la jugada"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, user.ID)
	c.JSON(http.StatusCreated, bingoResumen(ronda, actualizado.Saldo, 0))
}

// BingoHit canta la siguiente bolilla. Cierra y paga cuando un cartón se llena
// o cuando se acaban las cantadas de la ronda.
func (h *CasinoHandler) BingoHit(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.BingoRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	if ronda.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la jugada ya esta cerrada"})
		return
	}
	if len(ronda.Cantadas) >= casino.BingoCantadas {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ya se cantaron todas las bolillas"})
		return
	}
	if len(ronda.Orden) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "la jugada no tiene bolillas"})
		return
	}

	bolilla := ronda.Orden[len(ronda.Cantadas)]
	ronda.Cantadas = append(ronda.Cantadas, bolilla)

	_, _, lleno, _ := casino.BingoPremio(ronda.MontoPorCarton, ronda.Grids, ronda.Cantadas)
	if lleno || len(ronda.Cantadas) >= casino.BingoCantadas {
		h.cerrarBingo(c, &ronda, userID, bolilla)
		return
	}
	if err := h.db.Save(&ronda).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la bolilla"})
		return
	}

	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, bingoResumen(ronda, user.Saldo, bolilla))
}

// BingoClose cobra las líneas completadas hasta el momento.
func (h *CasinoHandler) BingoClose(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.BingoRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	if ronda.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la jugada ya esta cerrada"})
		return
	}
	bolilla := 0
	if len(ronda.Cantadas) > 0 {
		bolilla = ronda.Cantadas[len(ronda.Cantadas)-1]
	}
	h.cerrarBingo(c, &ronda, userID, bolilla)
}

// cerrarBingo acredita el premio, marca la ronda como pagada y registra el
// movimiento en el mismo saldo.
func (h *CasinoHandler) cerrarBingo(c *gin.Context, ronda *models.BingoRonda, userID uint, bolilla int) {
	premio, lineas, _, _ := casino.BingoPremio(ronda.MontoPorCarton, ronda.Grids, ronda.Cantadas)
	ronda.Estado = "pagada"
	ronda.Premio = premio
	ronda.Lineas = lineas
	if ronda.Monto > 0 {
		ronda.Multiplicador = float64(premio) / float64(ronda.Monto)
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(ronda).Error; err != nil {
			return err
		}
		if premio > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
				return err
			}
			mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: premio, Descripcion: "Premio casino Bingo 75"}
			if err := tx.Create(&mov).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar la jugada"})
		return
	}

	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, bingoResumen(*ronda, user.Saldo, bolilla))
}
