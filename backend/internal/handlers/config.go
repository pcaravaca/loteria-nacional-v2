package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

type ConfigHandler struct {
	db *gorm.DB
}

func NewConfigHandler(db *gorm.DB) *ConfigHandler { return &ConfigHandler{db: db} }

// marca devuelve la configuración de marca con valores por defecto.
func (h *ConfigHandler) marca() gin.H {
	out := gin.H{
		models.AjusteBrandNombre: "Lotería Nacional",
		models.AjusteBrandLogo:   "",
		models.AjusteBrandIcono:  "/logo.png",
	}
	for _, clave := range models.ClavesMarca {
		var a models.Ajuste
		if err := h.db.Where("clave = ?", clave).First(&a).Error; err == nil && a.Valor != "" {
			out[clave] = a.Valor
		}
	}
	return out
}

// Config es público: lo necesitan la web, la app y el login para mostrar el
// nombre y el logo del negocio que opera la plataforma.
func (h *ConfigHandler) Config(c *gin.Context) {
	c.JSON(http.StatusOK, h.marca())
}

type marcaRequest struct {
	Nombre string `json:"brand_nombre"`
	Logo   string `json:"brand_logo"`
	Icono  string `json:"brand_icono"`
}

// Guardar actualiza la marca. Solo el admin.
func (h *ConfigHandler) Guardar(c *gin.Context) {
	var req marcaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}
	valores := map[string]string{
		models.AjusteBrandNombre: req.Nombre,
		models.AjusteBrandLogo:   req.Logo,
		models.AjusteBrandIcono:  req.Icono,
	}
	for _, clave := range models.ClavesMarca {
		v := valores[clave]
		if v == "" {
			continue
		}
		var existente models.Ajuste
		err := h.db.Where("clave = ?", clave).First(&existente).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := h.db.Create(&models.Ajuste{Clave: clave, Valor: v}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la marca"})
				return
			}
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la marca"})
			return
		default:
			if err := h.db.Model(&models.Ajuste{}).Where("clave = ?", clave).
				Update("valor", v).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la marca"})
				return
			}
		}
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "marca.guardar", "ajuste", "-",
		"marca actualizada: "+req.Nombre)
	c.JSON(http.StatusOK, h.marca())
}
