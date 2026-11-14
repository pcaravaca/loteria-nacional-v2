package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// CasinoHandler expone el mini casino. El resultado y el pago se calculan aqui;
// el cliente solo anima lo que recibe.
type CasinoHandler struct {
	db *gorm.DB
}

func NewCasinoHandler(db *gorm.DB) *CasinoHandler {
	return &CasinoHandler{db: db}
}

// Games devuelve el catalogo de juegos activos.
func (h *CasinoHandler) Games(c *gin.Context) {
	var juegos []models.CasinoGame
	if err := h.db.Order("orden").Find(&juegos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los juegos"})
		return
	}
	c.JSON(http.StatusOK, juegos)
}

type semillaResponse struct {
	Hash        string `json:"hash"`
	Nonce       int64  `json:"nonce"`
	ClientSeed  string `json:"client_seed"`
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
	Juego       string         `json:"game" binding:"required"`
	Monto       int64          `json:"monto" binding:"required,min=1"`
	ClientSeed  string         `json:"client_seed"`
	Params      casino.Params  `json:"params"`
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
	if req.Monto < juego.MinApuesta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el minimo es " + strconv.FormatInt(juego.MinApuesta, 10)})
		return
	}
	if req.Monto > juego.MaxApuesta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el maximo es " + strconv.FormatInt(juego.MaxApuesta, 10)})
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
	if user.Saldo < req.Monto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente", "saldo": user.Saldo, "monto": req.Monto})
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
	premio := req.Monto * out.Multiplicador

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
			"saldo":       gorm.Expr("saldo - ?", req.Monto),
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
		"saldo":          actualizado.Saldo,
		"premio":         premio,
		"multiplicador":  out.Multiplicador,
		"resultado":      out.Data,
		"server_seed_hash": ronda.ServerSeedHash,
		"nonce":          ronda.Nonce,
		"ronda_id":       ronda.ID,
	})
}

// Rounds devuelve el historial de jugadas del usuario.
func (h *CasinoHandler) Rounds(c *gin.Context) {
	userID := c.GetUint("userID")
	var rondas []models.CasinoRound
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(100).Find(&rondas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las jugadas"})
		return
	}
	c.JSON(http.StatusOK, rondas)
}

// Verify revela la semilla del servidor de una ronda y reproduce el sorteo,
// para que el jugador pueda auditar el resultado.
func (h *CasinoHandler) Verify(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	var ronda models.CasinoRound
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ronda no encontrada"})
		return
	}

	var user models.User
	_ = h.db.First(&user, userID)

	draws := casino.Sortear(user.CasinoSeed, ronda.ClientSeed, ronda.Nonce, casino.Draws(ronda.GameCodigo), 1<<31-1)
	c.JSON(http.StatusOK, gin.H{
		"ronda_id":         ronda.ID,
		"juego":            ronda.GameCodigo,
		"monto":            ronda.Monto,
		"multiplicador":    ronda.Multiplicador,
		"premio":           ronda.Premio,
		"client_seed":      ronda.ClientSeed,
		"server_seed_hash": ronda.ServerSeedHash,
		"server_seed":      user.CasinoSeed,
		"nonce":            ronda.Nonce,
		"sorteo":           draws,
		"resultado":        ronda.Resultado,
		"verificado":       casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash,
	})
}

// Stats devuelve totales por juego (solo admin).
func (h *CasinoHandler) Stats(c *gin.Context) {
	type fila struct {
		Juego    string `json:"juego"`
		Jugadas  int64  `json:"jugadas"`
		Apostado int64  `json:"apostado"`
		Premios  int64  `json:"premios"`
		Beneficio int64 `json:"beneficio"`
	}
	var filas []fila
	if err := h.db.Model(&models.CasinoRound{}).
		Select("game_codigo as juego, count(*) as jugadas, sum(monto) as apostado, sum(premio) as premios, sum(monto - premio) as beneficio").
		Group("game_codigo").Scan(&filas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las estadisticas"})
		return
	}
	c.JSON(http.StatusOK, filas)
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
	if req.Monto < 500 || req.Monto > 50000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la apuesta debe estar entre 500 y 50000"})
		return
	}
	if user.Saldo < req.Monto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente"})
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

	var actualizado models.User
	_ = h.db.First(&actualizado, user.ID)
	c.JSON(http.StatusCreated, gin.H{
		"mano_id":           bj.ID,
		"mano":              bj.Mano,
		"crupier":           bj.Crupier,
		"estado":            bj.Estado,
		"premio":            bj.Premio,
		"saldo":             actualizado.Saldo,
		"server_seed_hash":  bj.ServerSeedHash,
		"nonce":             bj.Nonce,
	})
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
		c.JSON(http.StatusOK, gin.H{"mano_id": bj.ID, "mano": mano, "crupier": bj.Crupier, "estado": "abierta", "premio": 0, "saldo": user.Saldo})
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

	var actualizado models.User
	_ = h.db.First(&actualizado, userID)
	c.JSON(http.StatusOK, gin.H{
		"mano_id":       bj.ID,
		"mano":          bj.Mano,
		"crupier":       bj.Crupier,
		"estado":        bj.Estado,
		"premio":        bj.Premio,
		"multiplicador": bj.Multiplicador,
		"saldo":         actualizado.Saldo,
	})
}
