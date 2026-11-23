package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// CsHandler alimenta el panel de servicio al cliente: reconstruye lo que un
// jugador hizo, con el request id que correlaciona cada peticion.
type CsHandler struct {
	db *gorm.DB
}

func NewCsHandler(db *gorm.DB) *CsHandler { return &CsHandler{db: db} }

// LineaDeTiempo devuelve las acciones registradas de un jugador (user_id) o de
// una peticion exacta (request_id). Sin filtros no responde: el registro es
// grande y CS siempre busca un caso concreto.
func (h *CsHandler) LineaDeTiempo(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 64)
	requestID := c.Query("request_id")
	if userID == 0 && requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "indicá user_id o request_id"})
		return
	}

	q := h.db.Model(&models.AuditLog{})
	if userID != 0 {
		q = q.Where("actor_id = ?", userID)
	}
	if requestID != "" {
		q = q.Where("request_id = ?", requestID)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("created_at BETWEEN ? AND ?", desde, hasta)
	}
	if accion := c.Query("accion"); accion != "" {
		q = q.Where("accion = ?", accion)
	}

	limite := 200
	if l, err := strconv.Atoi(c.Query("limite")); err == nil && l > 0 && l <= 1000 {
		limite = l
	}

	var total int64
	_ = q.Count(&total).Error

	var entradas []models.AuditLog
	if err := q.Order("created_at asc").Limit(limite).Find(&entradas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar la linea de tiempo"})
		return
	}

	out := gin.H{
		"total":      total,
		"entradas":   entradas,
		"request_id": requestID,
	}
	if userID != 0 {
		var u models.User
		if err := h.db.First(&u, userID).Error; err == nil {
			out["jugador"] = gin.H{
				"id": u.ID, "nombre": u.Nombre, "email": u.Email,
				"rol": u.Rol, "saldo": u.Saldo, "bono": u.Bono,
				"telefono": u.Telefono, "verificado": u.DocumentoVerificado,
			}
		}
	}
	c.JSON(http.StatusOK, out)
}
