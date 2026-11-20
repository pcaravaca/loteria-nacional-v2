package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Video poker interactivo: la apuesta se descuenta al repartir (para que el
// jugador no pueda mirar manos gratis) y la mano queda abierta hasta que cobra.

const pokerCaducidad = 10 * time.Minute

type pokerRequest struct {
	Juego      string `json:"juego" binding:"required"`
	Monto      int64  `json:"monto" binding:"required,min=1"`
	ClientSeed string `json:"client_seed"`
}

type pokerCobrarRequest struct {
	Retener []int `json:"retener"`
}

// PokerRepartir descuenta la apuesta y devuelve las cinco cartas iniciales.
func (h *CasinoHandler) PokerRepartir(c *gin.Context) {
	var req pokerRequest
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
	if _, ok := casino.BuscarPoker(req.Juego); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "juego no encontrado"})
		return
	}
	var juego models.CasinoGame
	if err := h.db.Where("codigo = ?", req.Juego).First(&juego).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "el juego no esta configurado"})
		return
	}
	if !juego.Activo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el juego esta desactivado"})
		return
	}
	if !ExigirAcceso(c, h.db, user.ID, models.ContenidoJuego, juego.Codigo) {
		return
	}
	if req.Monto < juego.MinApuesta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el minimo es " + strconv.FormatInt(juego.MinApuesta, 10)})
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
	if !h.verificarLimite(c, user, juego.Codigo, req.Monto) {
		return
	}
	if excede, disponible, motivo := EvaluarGasto(h.db, user.ID, req.Monto); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu limite " + motivo + " de gasto se excederia con esta mano",
			"disponible": disponible, "motivo": motivo,
		})
		return
	}

	h.cerrarManosCaducadas(user.ID)

	clientSeed := h.semillaCliente(c, req.ClientSeed)
	nonce := user.CasinoNonce
	draws := casino.Sortear(user.CasinoSeed, clientSeed, nonce, casino.Draws(juego.Codigo), 1<<31-1)
	iniciales, err := casino.PokerIniciales(juego.Codigo, draws)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ronda := models.CasinoRound{
		UserID:         user.ID,
		GameCodigo:     juego.Codigo,
		Monto:          req.Monto,
		ClientSeed:     clientSeed,
		ServerSeedHash: casino.HashSemilla(user.CasinoSeed),
		Nonce:          nonce,
		Resultado:      map[string]any{"fase": "abierta", "cartas": iniciales},
	}
	mano := models.PokerMano{
		UserID:         user.ID,
		Juego:          juego.Codigo,
		Monto:          req.Monto,
		Iniciales:      iniciales,
		Estado:         "abierta",
		ClientSeed:     clientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		Nonce:          nonce,
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ronda).Error; err != nil {
			return err
		}
		mano.RondaID = ronda.ID
		if err := tx.Create(&mano).Error; err != nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la mano"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, user.ID)
	c.JSON(http.StatusCreated, gin.H{
		"mano_id":          mano.ID,
		"ronda_id":         ronda.ID,
		"cartas":           iniciales,
		"saldo":            actualizado.Saldo,
		"server_seed_hash": ronda.ServerSeedHash,
		"nonce":            nonce,
	})
}

// PokerCobrar aplica los holds elegidos, cierra la mano y paga.
func (h *CasinoHandler) PokerCobrar(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var mano models.PokerMano
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&mano).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mano no encontrada"})
		return
	}
	if mano.Estado != "abierta" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la mano ya esta cerrada"})
		return
	}
	var req pokerCobrarRequest
	_ = c.ShouldBindJSON(&req)
	if time.Since(mano.CreatedAt) > pokerCaducidad {
		// abandono: se resuelve con el reparto automatico
		req.Retener = nil
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	draws := casino.Sortear(user.CasinoSeed, mano.ClientSeed, mano.Nonce, casino.Draws(mano.Juego), 1<<31-1)
	out, err := casino.PokerReparte(mano.Juego, draws, req.Retener)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	premio := casino.Premio(mano.Monto, out)
	estado := "pagada"
	if time.Since(mano.CreatedAt) > pokerCaducidad {
		estado = "caducada"
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var ronda models.CasinoRound
		if err := tx.First(&ronda, mano.RondaID).Error; err != nil {
			return err
		}
		ronda.Multiplicador = out.Multiplicador
		ronda.Premio = premio
		ronda.Resultado = out.Data
		if err := tx.Save(&ronda).Error; err != nil {
			return err
		}
		mano.Estado = estado
		if err := tx.Save(&mano).Error; err != nil {
			return err
		}
		if premio > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).
				Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
				return err
			}
			mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: premio, Descripcion: "Premio casino " + mano.Juego}
			if err := tx.Create(&mov).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar la mano"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, userID)
	c.JSON(http.StatusOK, gin.H{
		"mano_id":  mano.ID,
		"ronda_id": mano.RondaID,
		"estado":   estado,
		"premio":   premio,
		"resultado": out.Data,
		"saldo":    actualizado.Saldo,
	})
}

// cerrarManosCaducadas resuelve las manos abiertas hace más de diez minutos.
func (h *CasinoHandler) cerrarManosCaducadas(userID uint) {
	var manos []models.PokerMano
	corte := time.Now().Add(-pokerCaducidad)
	if err := h.db.Where("user_id = ? AND estado = ? AND created_at < ?", userID, "abierta", corte).Find(&manos).Error; err != nil {
		return
	}
	for _, mano := range manos {
		var user models.User
		if err := h.db.First(&user, userID).Error; err != nil {
			continue
		}
		draws := casino.Sortear(user.CasinoSeed, mano.ClientSeed, mano.Nonce, casino.Draws(mano.Juego), 1<<31-1)
		out, err := casino.PokerReparte(mano.Juego, draws, nil)
		if err != nil {
			continue
		}
		premio := casino.Premio(mano.Monto, out)
		_ = h.db.Transaction(func(tx *gorm.DB) error {
			var ronda models.CasinoRound
			if err := tx.First(&ronda, mano.RondaID).Error; err != nil {
				return err
			}
			ronda.Multiplicador = out.Multiplicador
			ronda.Premio = premio
			ronda.Resultado = out.Data
			if err := tx.Save(&ronda).Error; err != nil {
				return err
			}
			mano.Estado = "caducada"
			if err := tx.Save(&mano).Error; err != nil {
				return err
			}
			if premio > 0 {
				if err := tx.Model(&models.User{}).Where("id = ?", userID).
					Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
					return err
				}
				mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: premio, Descripcion: "Premio casino " + mano.Juego}
				return tx.Create(&mov).Error
			}
			return nil
		})
	}
}
