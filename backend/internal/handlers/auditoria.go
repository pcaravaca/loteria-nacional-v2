package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// RegistrarAuditoria deja constancia de una accion relevante (admin, KYC,
// limites, configuracion). El registro no debe tumbar la operacion principal:
// si la escritura falla, se ignora silenciosamente.
func RegistrarAuditoria(db *gorm.DB, c *gin.Context, actorID uint, accion, entidad, entidadID, detalle string) {
	nombre := ""
	if actorID != 0 {
		var u models.User
		if err := db.Select("nombre").First(&u, actorID).Error; err == nil {
			nombre = u.Nombre
		}
	}
	ip := ""
	if c != nil {
		ip = c.ClientIP()
	}
	_ = db.Create(&models.AuditLog{
		ActorID:   actorID,
		ActorName: nombre,
		Accion:    accion,
		Entidad:   entidad,
		EntidadID: entidadID,
		Detalle:   detalle,
		IP:        ip,
	}).Error
}

// AuditoriaHandler expone el registro para la fiscalizacion.
type AuditoriaHandler struct {
	db *gorm.DB
}

func NewAuditoriaHandler(db *gorm.DB) *AuditoriaHandler {
	return &AuditoriaHandler{db: db}
}

// Listar devuelve las entradas con filtros y paginacion sencilla.
func (h *AuditoriaHandler) Listar(c *gin.Context) {
	q := h.db.Model(&models.AuditLog{})

	if actor := c.Query("actor"); actor != "" {
		q = q.Where("actor_id = ?", actor)
	}
	if entidad := c.Query("entidad"); entidad != "" {
		q = q.Where("entidad = ?", entidad)
	}
	if accion := c.Query("accion"); accion != "" {
		q = q.Where("accion = ?", accion)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("created_at BETWEEN ? AND ?", desde, hasta)
	}

	limite := 100
	if l, err := strconv.Atoi(c.Query("limite")); err == nil && l > 0 && l <= 1000 {
		limite = l
	}
	var total int64
	_ = q.Count(&total).Error

	var entradas []models.AuditLog
	if err := q.Order("created_at desc").Limit(limite).Find(&entradas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar la auditoria"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "entradas": entradas})
}

// rangoFechas interpreta desde/hasta (YYYY-MM-DD) como un tramo de Costa Rica.
// Devuelve falso si no hay filtros de fecha o si alguno es invalido.
func rangoFechas(c *gin.Context) (time.Time, time.Time, bool) {
	desde := c.Query("desde")
	hasta := c.Query("hasta")
	if desde == "" && hasta == "" {
		return time.Time{}, time.Time{}, false
	}

	ini := time.Date(2000, 1, 1, 0, 0, 0, 0, tiempo.CR)
	fin := time.Now().In(tiempo.CR).Add(24 * time.Hour)

	if desde != "" {
		t, err := time.ParseInLocation("2006-01-02", desde, tiempo.CR)
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
		ini = t
	}
	if hasta != "" {
		t, err := time.ParseInLocation("2006-01-02", hasta, tiempo.CR)
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
		fin = t.Add(24 * time.Hour)
	}
	return ini, fin, true
}
