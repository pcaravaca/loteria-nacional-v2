package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// Ayudas compartidas de los juegos de mesa nuevos (domino, ajedrez, backgammon
// y tetris). Repiten el esquema de bingo/gemas: la apuesta se cobra al abrir y
// el pago se acredita al liquidar, con tope diario, KYC y GeoBlock.

// nuevosJuego carga la configuracion del juego y verifica que este activo.
func (h *CasinoHandler) nuevosJuego(c *gin.Context, codigo string) (models.CasinoGame, bool) {
	var juego models.CasinoGame
	if err := h.db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "el juego no esta configurado"})
		return juego, false
	}
	if !juego.Activo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el juego esta desactivado"})
		return juego, false
	}
	return juego, true
}

// nuevosApuesta valida el monto, el saldo y los limites antes de cobrar.
func (h *CasinoHandler) nuevosApuesta(c *gin.Context, user models.User, juego models.CasinoGame, monto int64) bool {
	if monto < juego.MinApuesta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la apuesta debe ser de al menos " + strconv.FormatInt(juego.MinApuesta, 10)})
		return false
	}
	apuestaMax := casino.ApuestaMaxima(juego.MaxApuesta, juego.MaxMultipl, juego.MaxPremio, juego.Codigo)
	if monto > apuestaMax {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el maximo es " + strconv.FormatInt(apuestaMax, 10), "maximo": apuestaMax})
		return false
	}
	if user.Saldo < monto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente"})
		return false
	}
	if !h.nuevosLimite(c, user, juego, monto) {
		return false
	}
	if excede, disponible, motivo := EvaluarGasto(h.db, user.ID, monto); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu limite " + motivo + " de gasto se excederia con esta jugada",
			"disponible": disponible, "motivo": motivo,
		})
		return false
	}
	return true
}

// nuevosLimite aplica el tope diario sobre la tabla propia del juego.
func (h *CasinoHandler) nuevosLimite(c *gin.Context, user models.User, juego models.CasinoGame, monto int64) bool {
	if juego.LimiteDiario <= 0 {
		return true
	}
	usado := h.nuevosApostadoHoy(user.ID, juego.Codigo, tiempo.InicioDia())
	if usado < 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo verificar el limite diario"})
		return false
	}
	if usado+monto > juego.LimiteDiario {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "limite diario alcanzado",
			"limite":     juego.LimiteDiario,
			"usado":      usado,
			"disponible": juego.LimiteDiario - usado,
		})
		return false
	}
	return true
}

// nuevosApostadoHoy suma lo apostado hoy en la tabla del juego.
func (h *CasinoHandler) nuevosApostadoHoy(userID uint, codigo string, desde time.Time) int64 {
	var total int64
	var err error
	switch codigo {
	case models.CasinoDomino:
		err = h.db.Model(&models.DominoRonda{}).Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&total).Error
	case models.CasinoAjedrez:
		err = h.db.Model(&models.AjedrezRonda{}).Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&total).Error
	case models.CasinoBackgammon:
		err = h.db.Model(&models.BackgammonRonda{}).Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&total).Error
	case models.CasinoTetris:
		err = h.db.Model(&models.TetrisRonda{}).Where("user_id = ? AND created_at >= ?", userID, desde).
			Select("COALESCE(SUM(monto),0)").Scan(&total).Error
	default:
		return 0
	}
	if err != nil {
		return -1
	}
	return total
}

// nuevosCobrar descuenta la apuesta, avanza el nonce y registra el movimiento.
func (h *CasinoHandler) nuevosCobrar(tx *gorm.DB, user models.User, juego models.CasinoGame, monto, nonce int64) error {
	if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"saldo":        gorm.Expr("saldo - ?", monto),
		"casino_nonce": nonce + 1,
	}).Error; err != nil {
		return err
	}
	mov := models.Movimiento{UserID: user.ID, Tipo: models.MovCasinoApuesta, Monto: -monto, Descripcion: "Casino " + juego.Nombre}
	return tx.Create(&mov).Error
}

// nuevosPagar acredita el premio y registra el movimiento.
func (h *CasinoHandler) nuevosPagar(tx *gorm.DB, userID uint, premio int64, juego string) error {
	if premio <= 0 {
		return nil
	}
	if err := tx.Model(&models.User{}).Where("id = ?", userID).
		Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
		return err
	}
	mov := models.Movimiento{UserID: userID, Tipo: models.MovCasinoPremio, Monto: premio, Descripcion: "Premio casino " + juego}
	return tx.Create(&mov).Error
}
