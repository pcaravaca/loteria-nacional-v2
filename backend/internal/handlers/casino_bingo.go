package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Bingo: la apuesta se descuenta al abrir la ronda y las bolillas se cantan de a
// una. El premio depende de la variante (patrones y multiplicadores propios); el
// orden de las bolillas y los cartones salen de las semillas (provably fair).

type bingoRequest struct {
	Monto      int64  `json:"monto" binding:"required,min=1"`
	Cartones   int    `json:"cartones"`
	ClientSeed string `json:"client_seed"`
	Variante   string `json:"variante"`
	// IdempotencyKey: si el cliente reenvia el mismo start (sin datos, timeout,
	// 502), se devuelve la ronda ya creada en lugar de cobrar dos veces.
	IdempotencyKey string `json:"idempotency_key"`
}

// bingoHitRequest trae el estado que el cliente ya ve. Con eso el servidor
// detecta un reintento y no canta de mas.
type bingoHitRequest struct {
	// CantadasEsperadas: cuantas bolillas ya ve el cliente. Si el servidor va
	// mas adelante, la respuesta le entrega las que faltan en vez de saltar una.
	// Si no viene (cliente viejo), se mantiene el comportamiento de siempre.
	CantadasEsperadas *int   `json:"cantadas_esperadas"`
	RequestID         string `json:"request_id"`
}

// ultimaBolilla devuelve la ultima bolilla cantada (0 si ninguna).
func ultimaBolilla(ronda models.BingoRonda) int {
	if len(ronda.Cantadas) == 0 {
		return 0
	}
	return ronda.Cantadas[len(ronda.Cantadas)-1]
}

// bingoPorClave busca la ronda que abrio un idempotency key.
func bingoPorClave(db *gorm.DB, userID uint, clave string) (models.BingoRonda, bool) {
	var ronda models.BingoRonda
	err := db.Where("user_id = ? AND idempotency_key = ?", userID, clave).
		Order("created_at desc").First(&ronda).Error
	if err != nil {
		return ronda, false
	}
	return ronda, true
}

// bingoRepetida responde un reintento de start que ya se habia aplicado.
func (h *CasinoHandler) bingoRepetida(c *gin.Context, ronda models.BingoRonda, reanudar bool) {
	estado := "repetida"
	if reanudar {
		estado = "reanudada"
	}
	RegistrarAccionJugador(h.db, c, ronda.UserID, "bingo_start_"+estado, "bingo_ronda",
		strconv.FormatUint(uint64(ronda.ID), 10),
		"monto="+strconv.FormatInt(ronda.Monto, 10)+" cartones="+strconv.Itoa(ronda.Cartones))
	c.JSON(http.StatusOK, gin.H{
		"repetida":     true,
		"ronda_id":     ronda.ID,
		"idempotencia": true,
		"estado":       estado,
		"resumen":      bingoResumen(ronda, saldoDe(h.db, ronda.UserID, 0), ultimaBolilla(ronda)),
	})
}

// bingoJuego carga la configuración de la variante y verifica que esté activa.
func (h *CasinoHandler) bingoJuego(c *gin.Context, codigo string) (models.CasinoGame, bool) {
	var juego models.CasinoGame
	if err := h.db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la variante no está configurada"})
		return juego, false
	}
	if !juego.Activo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la variante está desactivada"})
		return juego, false
	}
	return juego, true
}

// bingoTabla deja la tabla de pagos de la variante en el formato que espera el
// frontend.
func bingoTabla(v casino.BingoVariante) []gin.H {
	out := make([]gin.H, 0, len(v.Pagos))
	for _, p := range v.Pagos {
		out = append(out, gin.H{"nombre": p.Nombre, "mult": p.Mult})
	}
	return out
}

// bingoResumen arma la respuesta del estado de la ronda. `premio` es lo que se
// pagaría con las bolillas ya cantadas; `premio_pagado` es lo que ya se acreditó
// al cerrar.
func bingoResumen(ronda models.BingoRonda, saldo int64, bolilla int) gin.H {
	codigo := ronda.Variante
	if codigo == "" {
		codigo = models.CasinoBingo
	}
	v, ok := casino.BingoVariantePorCodigo(codigo)
	if !ok {
		v, _ = casino.BingoVariantePorCodigo(models.CasinoBingo)
		codigo = models.CasinoBingo
	}
	premio, lineas, lleno, tiers, patron := casino.BingoPremioVariante(ronda.MontoPorCarton, ronda.Grids, ronda.Cantadas, codigo)
	return gin.H{
		"ronda_id":          ronda.ID,
		"variante":          codigo,
		"nombre_variante":   v.Nombre,
		"monto":             ronda.Monto,
		"monto_por_carton":  ronda.MontoPorCarton,
		"cartones":          ronda.Cartones,
		"grids":             ronda.Grids,
		"orden":             ronda.Orden,
		"cantadas":          ronda.Cantadas,
		"bolilla":           bolilla,
		"cantidad_cantadas": v.TopeCantadas,
		"tope_cantadas":     v.TopeCantadas,
		"bolillas":          v.Bolillas,
		"filas":             v.Filas,
		"columnas":          v.Columnas,
		"centro_libre":      v.CentroLibre,
		"paytable":          bingoTabla(v),
		"estado":            ronda.Estado,
		"lineas":            lineas,
		"lleno":             lleno,
		"patron":            patron,
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
// bolillas (que el cliente no necesita mostrar: las revela `hit` de a una).
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
	// Reintento del mismo start: no vuelve a cobrar ni a generar cartones.
	if req.IdempotencyKey != "" {
		if ronda, ok := bingoPorClave(h.db, user.ID, req.IdempotencyKey); ok {
			h.bingoRepetida(c, ronda, false)
			return
		}
	}
	variante := req.Variante
	if variante == "" {
		variante = models.CasinoBingo
	}
	// Una sola jugada abierta por jugador. Si es de la misma variante se retoma
	// (sin cobrar de nuevo); si es de otra, se avisa en cuál quedó para no
	// cobrar otra apuesta ni mezclar rondas de modalidades distintas.
	if ronda, ok := bingoReanudar(h.db, user.ID, ""); ok {
		if ronda.Variante == variante {
			h.bingoRepetida(c, ronda, true)
			return
		}
		nombre := ronda.Variante
		if v, ok := casino.BingoVariantePorCodigo(ronda.Variante); ok {
			nombre = v.Nombre
		}
		c.JSON(http.StatusConflict, gin.H{
			"error":    "ya tenés una jugada abierta en " + nombre + "; cerrala o seguila ahí",
			"ronda_id": ronda.ID,
			"variante": ronda.Variante,
		})
		return
	}
	if _, ok := casino.BingoVariantePorCodigo(variante); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "variante de bingo invalida: " + variante})
		return
	}
	juego, ok := h.bingoJuego(c, variante)
	if !ok {
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, variante) {
		return
	}
	v, _ := casino.BingoVariantePorCodigo(variante)
	if req.Cartones < 1 {
		req.Cartones = 1
	}
	if req.Cartones > v.MaxCartones {
		req.Cartones = v.MaxCartones
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
	if !h.verificarLimite(c, user, variante, total) {
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
	orden, grids := casino.BingoSorteoVariante(clientSeed, user.CasinoSeed, nonce, req.Cartones, variante)

	ronda := models.BingoRonda{
		UserID:         user.ID,
		Variante:       variante,
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
		IdempotencyKey: req.IdempotencyKey,
		RequestID:      requestIDDe(c),
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
		// Dos peticiones con la misma clave llegaron juntas: la base de datos
		// dejo pasar una sola. Se devuelve la que quedo, no un error.
		if req.IdempotencyKey != "" {
			if repetida, ok := bingoPorClave(h.db, user.ID, req.IdempotencyKey); ok {
				h.bingoRepetida(c, repetida, false)
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la jugada"})
		return
	}

	RegistrarAccionJugador(h.db, c, user.ID, "bingo_start", "bingo_ronda",
		strconv.FormatUint(uint64(ronda.ID), 10),
		"variante="+variante+" monto="+strconv.FormatInt(total, 10)+" cartones="+strconv.Itoa(req.Cartones)+
			" semilla_hash="+ronda.ServerSeedHash)

	var actualizado models.User
	_ = h.db.First(&actualizado, user.ID)
	c.JSON(http.StatusCreated, bingoResumen(ronda, actualizado.Saldo, 0))
}

// BingoHit canta la siguiente bolilla. Cierra y paga cuando un cartón se llena
// o cuando se acaban las cantadas de la ronda.
//
// Es idempotente: el cliente avisa cuantas bolillas ya ve y, si el servidor va
// mas adelante (la respuesta anterior se perdio), le devuelve el estado en vez
// de cantar una bolilla de mas. Si ya se cerro, devuelve el resultado.
func (h *CasinoHandler) BingoHit(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var req bingoHitRequest
	_ = c.ShouldBindJSON(&req)
	if req.RequestID != "" {
		c.Set("request_id", req.RequestID)
	}

	var ronda models.BingoRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "la jugada no existe"})
		return
	}

	responderEstado := func(accion string) {
		RegistrarAccionJugador(h.db, c, userID, accion, "bingo_ronda",
			strconv.FormatUint(uint64(ronda.ID), 10),
			"bolilla="+strconv.Itoa(ultimaBolilla(ronda))+" cantadas="+strconv.Itoa(len(ronda.Cantadas))+
				" estado="+ronda.Estado)
		var user models.User
		_ = h.db.First(&user, userID)
		c.JSON(http.StatusOK, bingoResumen(ronda, user.Saldo, ultimaBolilla(ronda)))
	}

	if ronda.Estado != "abierta" {
		// Reintento de un hit que ya cerro la ronda: no es un error.
		responderEstado("bingo_hit_cerrada")
		return
	}
	codigo := ronda.Variante
	if codigo == "" {
		codigo = models.CasinoBingo
	}
	v, ok := casino.BingoVariantePorCodigo(codigo)
	if !ok {
		v, _ = casino.BingoVariantePorCodigo(models.CasinoBingo)
		codigo = models.CasinoBingo
	}
	if len(ronda.Cantadas) >= v.TopeCantadas {
		h.cerrarBingo(c, &ronda, userID, ultimaBolilla(ronda))
		return
	}
	if len(ronda.Orden) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "la jugada no tiene bolillas"})
		return
	}
	// Sincronizacion por posicion: un hit reenviado despues de un error de red
	// no puede hacer avanzar la ronda dos veces.
	if req.CantadasEsperadas != nil && *req.CantadasEsperadas != len(ronda.Cantadas) {
		responderEstado("bingo_hit_sincroniza")
		return
	}

	bolilla := ronda.Orden[len(ronda.Cantadas)]
	ronda.Cantadas = append(ronda.Cantadas, bolilla)

	_, _, lleno, _, _ := casino.BingoPremioVariante(ronda.MontoPorCarton, ronda.Grids, ronda.Cantadas, codigo)
	if lleno || len(ronda.Cantadas) >= v.TopeCantadas {
		h.cerrarBingo(c, &ronda, userID, bolilla)
		return
	}
	if err := h.db.Save(&ronda).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la bolilla"})
		return
	}

	RegistrarAccionJugador(h.db, c, userID, "bingo_hit", "bingo_ronda",
		strconv.FormatUint(uint64(ronda.ID), 10),
		"bolilla="+strconv.Itoa(bolilla)+" cantadas="+strconv.Itoa(len(ronda.Cantadas)))

	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, bingoResumen(ronda, user.Saldo, bolilla))
}

// BingoClose cobra el patrón logrado hasta el momento. Si la ronda ya se cobro
// (la respuesta anterior se perdio), devuelve el mismo resultado.
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
		bolilla := ultimaBolilla(ronda)
		RegistrarAccionJugador(h.db, c, userID, "bingo_close_repetida", "bingo_ronda",
			strconv.FormatUint(uint64(ronda.ID), 10),
			"premio="+strconv.FormatInt(ronda.Premio, 10)+" cantadas="+strconv.Itoa(len(ronda.Cantadas)))
		var user models.User
		_ = h.db.First(&user, userID)
		c.JSON(http.StatusOK, bingoResumen(ronda, user.Saldo, bolilla))
		return
	}
	bolilla := ultimaBolilla(ronda)
	h.cerrarBingo(c, &ronda, userID, bolilla)
}

// cerrarBingo acredita el premio, marca la ronda como pagada y registra el
// movimiento en el mismo saldo.
func (h *CasinoHandler) cerrarBingo(c *gin.Context, ronda *models.BingoRonda, userID uint, bolilla int) {
	codigo := ronda.Variante
	if codigo == "" {
		codigo = models.CasinoBingo
	}
	premio, lineas, _, _, _ := casino.BingoPremioVariante(ronda.MontoPorCarton, ronda.Grids, ronda.Cantadas, codigo)
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
			v, _ := casino.BingoVariantePorCodigo(codigo)
			mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: premio, Descripcion: "Premio casino " + v.Nombre}
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

	RegistrarAccionJugador(h.db, c, userID, "bingo_close", "bingo_ronda",
		strconv.FormatUint(uint64(ronda.ID), 10),
		"premio="+strconv.FormatInt(premio, 10)+" cantadas="+strconv.Itoa(len(ronda.Cantadas))+
			" multiplicador="+strconv.FormatFloat(ronda.Multiplicador, 'f', 2, 64))

	var user models.User
	_ = h.db.First(&user, userID)
	c.JSON(http.StatusOK, bingoResumen(*ronda, user.Saldo, bolilla))
}
