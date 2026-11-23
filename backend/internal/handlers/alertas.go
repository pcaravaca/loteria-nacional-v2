package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// AlertasHandler expone las alertas del panel de servicio al cliente.
type AlertasHandler struct {
	db *gorm.DB
}

func NewAlertasHandler(db *gorm.DB) *AlertasHandler { return &AlertasHandler{db: db} }

// Listar devuelve las alertas, por defecto las pendientes de revisar.
func (h *AlertasHandler) Listar(c *gin.Context) {
	q := h.db.Model(&models.AlertaCS{})
	if c.Query("revisadas") != "true" {
		q = q.Where("revisada = ?", false)
	}
	if severidad := c.Query("severidad"); severidad != "" {
		q = q.Where("severidad = ?", severidad)
	}
	if user := c.Query("user_id"); user != "" {
		q = q.Where("user_id = ?", user)
	}
	limite := 100
	if l, err := strconv.Atoi(c.Query("limite")); err == nil && l > 0 && l <= 500 {
		limite = l
	}

	var total int64
	_ = q.Count(&total).Error

	var alertas []models.AlertaCS
	if err := q.Order("created_at desc").Limit(limite).Find(&alertas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las alertas"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "alertas": alertas})
}

// Revisar marca una alerta como atendida.
func (h *AlertasHandler) Revisar(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	if err := h.db.Model(&models.AlertaCS{}).Where("id = ?", id).
		Update("revisada", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo marcar la alerta"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
