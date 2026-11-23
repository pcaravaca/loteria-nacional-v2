package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// requestIDDe devuelve el identificador de la peticion (middleware.RequestID).
// Vacio si el middleware no corrio; nunca debe tumbar una jugada.
func requestIDDe(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// RegistrarAccionJugador deja constancia de una accion del jugador (abrir una
// jugada, cantar, cobrar, rendirse). A diferencia de RegistrarAuditoria, que es
// para el administrador, este registro es la materia prima del panel de CS:
// arma la linea de tiempo de lo que el jugador hizo, con el request id.
func RegistrarAccionJugador(db *gorm.DB, c *gin.Context, userID uint, accion, entidad, entidadID, detalle string) {
	if db == nil {
		return
	}
	ip := ""
	if c != nil {
		ip = c.ClientIP()
	}
	_ = db.Create(&models.AuditLog{
		ActorID:   userID,
		Accion:    accion,
		Entidad:   entidad,
		EntidadID: entidadID,
		Detalle:   detalle,
		IP:        ip,
		RequestID: requestIDDe(c),
	}).Error
}

// saldoDe devuelve el saldo actual del usuario. Si no se puede leer, devuelve
// el valor de respaldo: nunca debe impedir responder al jugador.
func saldoDe(db *gorm.DB, userID uint, respaldo int64) int64 {
	var u models.User
	if err := db.Select("saldo").First(&u, userID).Error; err == nil {
		return u.Saldo
	}
	return respaldo
}

// bingoReanudar busca la ultima ronda abierta del jugador. Devuelve falso si no
// hay ninguna: el front arranca una jugada nueva.
func bingoReanudar(db *gorm.DB, userID uint) (models.BingoRonda, bool) {
	var ronda models.BingoRonda
	err := db.Where("user_id = ? AND estado = ?", userID, "abierta").
		Order("created_at desc").First(&ronda).Error
	if err != nil {
		return ronda, false
	}
	return ronda, true
}

// BingoReanudarHandler devuelve la ronda abierta del jugador para que la app
// la retome si se corto la conexion a mitad de jugada. Sin ronda abierta,
// responde 204 y el front sigue como antes.
func (h *CasinoHandler) BingoReanudarHandler(c *gin.Context) {
	userID := c.GetUint("userID")
	ronda, ok := bingoReanudar(h.db, userID)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"reanudar": false, "ronda": nil})
		return
	}
	bolilla := 0
	if len(ronda.Cantadas) > 0 {
		bolilla = ronda.Cantadas[len(ronda.Cantadas)-1]
	}
	c.JSON(http.StatusOK, gin.H{
		"ronda":    bingoResumen(ronda, saldoDe(h.db, userID, 0), bolilla),
		"reanudar": true,
	})
}

// idEntero parsea un id de ruta. Devuelve falso si no es un numero valido.
func idEntero(c *gin.Context, nombre string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(nombre), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return 0, false
	}
	return id, true
}
