package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Gemas (match-3): la apuesta se descuenta al abrir la ronda y el jugador
// intercambia celdas adyacentes hasta agotar los movimientos. El tablero y el
// relleno salen de las semillas (provably fair) y el pago se acredita al
// liquidar la ronda (por tope de movimientos o con "Reventar").

type gemasRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	ClientSeed string `json:"client_seed"`
}

type gemasMoveRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

// gemasJuego carga la configuración del juego y verifica que esté activo.
func (h *CasinoHandler) gemasJuego(c *gin.Context) (models.CasinoGame, bool) {
	var juego models.CasinoGame
	if err := h.db.Where("codigo = ?", models.CasinoGemas).First(&juego).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "el juego no esta configurado"})
		return juego, false
	}
	if !juego.Activo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el juego esta desactivado"})
		return juego, false
	}
	return juego, true
}

// gemasChorro reconstruye el chorro de gemas de relleno a partir de las
// semillas y lo ubica en el punto en que quedó la ronda.
func gemasChorro(user models.User, ronda models.GemasRonda) *casino.GemasChorro {
	c := casino.NuevoGemasChorro(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce)
	c.Saltar(ronda.Cursor)
	return c
}

// gemasResumen arma la respuesta del estado de la ronda. `premio` es lo que se
// pagaría al liquidar ahora; `premio_pagado` es lo que ya se acreditó.
func gemasResumen(ronda models.GemasRonda, saldo int64, pasos []casino.GemasPaso) gin.H {
	premio := casino.GemasPremio(ronda.Monto, ronda.Puntos)
	return gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"tablero":          ronda.Tablero,
		"jugadas":          ronda.Jugadas,
		"movimientos":      ronda.Movimientos,
		"movimientos_max":  casino.GemasMovimientos,
		"puntos":           ronda.Puntos,
		"multiplicador":    casino.GemasMultiplicador(ronda.Puntos),
		"premio":           premio,
		"premio_pagado":    ronda.Premio,
		"estado":           ronda.Estado,
		"hay_movimientos":  casino.GemasHayMovimiento(ronda.Tablero),
		"pasos":            pasos,
		"saldo":            saldo,
		"server_seed_hash": ronda.ServerSeedHash,
		"nonce":            ronda.Nonce,
		"stream_offset":    ronda.StreamOffset,
	}
}

// GemasStart cobra la apuesta y abre el tablero. El tablero inicial no tiene
// combinaciones armadas: el jugador tiene que armarlas.
func (h *CasinoHandler) GemasStart(c *gin.Context) {
	var req gemasRequest
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
	juego, ok := h.gemasJuego(c)
	if !ok {
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, models.CasinoGemas) {
		return
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
	if user.Saldo < req.Monto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente"})
		return
	}
	if !h.verificarLimite(c, user, models.CasinoGemas, req.Monto) {
		return
	}
	if excede, disponible, motivo := EvaluarGasto(h.db, user.ID, req.Monto); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu limite " + motivo + " de gasto se excederia con esta jugada",
			"disponible": disponible, "motivo": motivo,
		})
		return
	}

	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	tablero, offset := casino.GemasSorteo(clientSeed, user.CasinoSeed, nonce)

	ronda := models.GemasRonda{
		UserID:         user.ID,
		Monto:          req.Monto,
		Tablero:        tablero,
		StreamOffset:   offset,
		Cursor:         offset,
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
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"saldo":        gorm.Expr("saldo - ?", req.Monto),
			"casino_nonce": nonce + 1,
		}).Error; err != nil {
			return err
		}
		mov := models.Movimiento{UserID: user.ID, Tipo: models.MovCasinoApuesta, Monto: -req.Monto, Descripcion: "Casino " + juego.Nombre}
		return tx.Create(&mov).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la jugada"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, user.ID)
	RegistrarAccionJugador(h.db, c, user.ID, "gemas_start", "gemas_ronda",
		strconv.FormatUint(uint64(ronda.ID), 10),
		"monto="+strconv.FormatInt(ronda.Monto, 10)+" semilla_hash="+ronda.ServerSeedHash)

	c.JSON(http.StatusCreated, gemasResumen(ronda, actualizado.Saldo, nil))
}

// GemasMove aplica un intercambio de dos celdas adyacentes y resuelve la
// cascada. Si el intercambio no forma combinación, no gasta el movimiento.
func (h *CasinoHandler) GemasMove(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.GemasRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	if ronda.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la jugada ya esta cerrada"})
		return
	}
	if ronda.Movimientos >= casino.GemasMovimientos {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ya usaste todos los movimientos"})
		return
	}
	var req gemasMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	if req.A < 0 || req.A >= casino.GemasFilas*casino.GemasColumnas || req.B < 0 || req.B >= casino.GemasFilas*casino.GemasColumnas {
		c.JSON(http.StatusBadRequest, gin.H{"error": "las celdas estan fuera del tablero"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	chorro := gemasChorro(user, ronda)
	res, err := casino.GemasAplicar(ronda.Tablero, chorro, req.A, req.B)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ronda.Tablero = res.Tablero
	ronda.Cursor = res.Cursor
	ronda.Movimientos++
	ronda.Puntos += res.Puntos
	ronda.Jugadas = append(ronda.Jugadas, [2]int{req.A, req.B})

	// Al llegar al tope de movimientos, o si el tablero se quedó sin jugadas,
	// la ronda se liquida sola.
	if ronda.Movimientos >= casino.GemasMovimientos || !casino.GemasHayMovimiento(ronda.Tablero) {
		h.cerrarGemas(c, &ronda, userID, res.Pasos)
		return
	}
	if err := h.db.Save(&ronda).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el movimiento"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, userID)
	c.JSON(http.StatusOK, gemasResumen(ronda, actualizado.Saldo, res.Pasos))
}

// GemasReventar liquida la ronda antes del tope y paga lo acumulado.
func (h *CasinoHandler) GemasReventar(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.GemasRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	if ronda.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la jugada ya esta cerrada"})
		return
	}
	h.cerrarGemas(c, &ronda, userID, nil)
}

// cerrarGemas acredita el premio, marca la ronda como pagada y registra el
// movimiento en el mismo saldo.
func (h *CasinoHandler) cerrarGemas(c *gin.Context, ronda *models.GemasRonda, userID uint, pasos []casino.GemasPaso) {
	premio := casino.GemasPremio(ronda.Monto, ronda.Puntos)
	ronda.Estado = "pagada"
	ronda.Premio = premio
	ronda.Multiplicador = casino.GemasMultiplicador(ronda.Puntos)

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(ronda).Error; err != nil {
			return err
		}
		if premio > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
				return err
			}
			mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: premio, Descripcion: "Premio casino Gemas"}
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
	c.JSON(http.StatusOK, gemasResumen(*ronda, user.Saldo, pasos))
}

// GemasVerificar revela la semilla del servidor y reproduce el tablero y los
// puntos a partir de los intercambios guardados.
func (h *CasinoHandler) GemasVerificar(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ronda models.GemasRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}

	tablero, offset := casino.GemasSorteo(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce)
	chorro := casino.NuevoGemasChorro(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce)
	chorro.Saltar(offset)
	repetido, err := casino.GemasRepetir(tablero, chorro, ronda.Jugadas)
	verificado := err == nil &&
		casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		offset == ronda.StreamOffset &&
		repetido.Puntos == ronda.Puntos &&
		igualarTablero(repetido.Tablero, ronda.Tablero)

	RegistrarAccionJugador(h.db, c, user.ID, "gemas_cerrar", "gemas_ronda",
		strconv.FormatUint(uint64(ronda.ID), 10),
		"monto="+strconv.FormatInt(ronda.Monto, 10)+" puntos="+strconv.FormatInt(ronda.Puntos, 10)+
			" premio="+strconv.FormatInt(ronda.Premio, 10)+" verificado="+strconv.FormatBool(verificado))

	c.JSON(http.StatusOK, gin.H{
		"ronda_id":         ronda.ID,
		"monto":            ronda.Monto,
		"puntos":           ronda.Puntos,
		"premio":           ronda.Premio,
		"multiplicador":    ronda.Multiplicador,
		"movimientos":      ronda.Movimientos,
		"jugadas":          ronda.Jugadas,
		"tablero":          ronda.Tablero,
		"client_seed":      ronda.ClientSeed,
		"server_seed_hash": ronda.ServerSeedHash,
		"server_seed":      user.CasinoSeed,
		"nonce":            ronda.Nonce,
		"verificado":       verificado,
	})
}

func igualarTablero(a, b []int) bool {
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
