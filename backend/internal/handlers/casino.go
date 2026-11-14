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
