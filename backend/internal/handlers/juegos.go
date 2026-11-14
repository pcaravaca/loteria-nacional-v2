package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

type JuegosHandler struct {
	db *gorm.DB
}

func NewJuegosHandler(db *gorm.DB) *JuegosHandler {
	return &JuegosHandler{db: db}
}

func (h *JuegosHandler) ListLoterias(c *gin.Context) {
	q := h.db.Model(&models.Loteria{})
	if c.Query("tipo") != "" {
		q = q.Where("tipo = ?", c.Query("tipo"))
	}
	if c.Query("todas") != "1" {
		q = q.Where("activa = ?", true)
	}

	var loterias []models.Loteria
	if err := q.Order("tipo, nombre").Find(&loterias).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las loterias"})
		return
	}
	c.JSON(http.StatusOK, loterias)
}

func (h *JuegosHandler) GetLoteria(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var loteria models.Loteria
	if err := h.db.First(&loteria, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "loteria no encontrada"})
		return
	}
	c.JSON(http.StatusOK, loteria)
}

func (h *JuegosHandler) ListSorteos(c *gin.Context) {
	q := h.db.Model(&models.Sorteo{})
	if v := c.Query("loteria_id"); v != "" {
		q = q.Where("loteria_id = ?", v)
	}
	if v := c.Query("estado"); v != "" {
		q = q.Where("estado = ?", v)
	}

	var sorteos []models.Sorteo
	if err := q.Preload("Loteria").Order("fecha desc").Find(&sorteos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los sorteos"})
		return
	}
	c.JSON(http.StatusOK, sorteos)
}

func (h *JuegosHandler) GetSorteo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var sorteo models.Sorteo
	if err := h.db.Preload("Loteria").First(&sorteo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sorteo no encontrado"})
		return
	}
	c.JSON(http.StatusOK, sorteo)
}

func (h *JuegosHandler) ProximosSorteos(c *gin.Context) {
	var sorteos []models.Sorteo
	if err := h.db.Preload("Loteria").
		Where("estado = ? AND fecha >= ?", models.SorteoAbierto, time.Now()).
		Order("fecha asc").Limit(10).Find(&sorteos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los sorteos"})
		return
	}
	c.JSON(http.StatusOK, sorteos)
}
