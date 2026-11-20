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
	// El panel pide el catalogo completo con todas=1; el resto solo ve las
	// loterias con acceso activo (global, paquete o jugador).
	verTodas := c.Query("todas") == "1" && rolOperador(c)
	if !verTodas {
		ids, err := LoteriasVisibles(h.db, c.GetUint("userID"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las loterias"})
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusOK, []models.Loteria{})
			return
		}
		q = q.Where("id IN ?", ids)
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
	if !rolOperador(c) && !ExigirAcceso(c, h.db, c.GetUint("userID"), models.ContenidoLoteria, loteria.Codigo) {
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
	if !rolOperador(c) {
		ids, err := LoteriasVisibles(h.db, c.GetUint("userID"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los sorteos"})
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusOK, []models.Sorteo{})
			return
		}
		q = q.Where("loteria_id IN ?", ids)
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
	q := h.db.Model(&models.Sorteo{}).
		Where("estado = ? AND fecha >= ?", models.SorteoAbierto, time.Now())
	if !rolOperador(c) {
		ids, err := LoteriasVisibles(h.db, c.GetUint("userID"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los sorteos"})
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusOK, []models.Sorteo{})
			return
		}
		q = q.Where("loteria_id IN ?", ids)
	}
	var sorteos []models.Sorteo
	if err := q.Preload("Loteria").Order("fecha asc").Limit(10).Find(&sorteos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los sorteos"})
		return
	}
	c.JSON(http.StatusOK, sorteos)
}
