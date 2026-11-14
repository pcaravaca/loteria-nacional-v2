package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []models.User
	if err := h.db.Order("created_at desc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los usuarios"})
		return
	}
	c.JSON(http.StatusOK, users)
}

type ajustarSaldoRequest struct {
	Monto       int64  `json:"monto" binding:"required"`
	Descripcion string `json:"descripcion"`
}

func (h *AdminHandler) AjustarSaldo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	var req ajustarSaldoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "monto requerido"})
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	if user.Saldo+req.Monto < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el ajuste dejaria el saldo en negativo"})
		return
	}

	tipo := models.MovAjuste
	if req.Monto > 0 {
		tipo = models.MovRecarga
	} else if req.Monto < 0 {
		tipo = models.MovRetiro
	}

	desc := req.Descripcion
	if desc == "" {
		desc = "Ajuste de saldo"
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", id).
			Update("saldo", gorm.Expr("saldo + ?", req.Monto)).Error; err != nil {
			return err
		}
		return tx.Create(&models.Movimiento{
			UserID: uint(id), Tipo: tipo, Monto: req.Monto, Descripcion: desc,
		}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo aplicar el ajuste"})
		return
	}

	var actualizado models.User
	_ = h.db.First(&actualizado, id)
	c.JSON(http.StatusOK, actualizado)
}

type loteriaRequest struct {
	Codigo        string `json:"codigo" binding:"required"`
	Nombre        string `json:"nombre" binding:"required"`
	Tipo          string `json:"tipo" binding:"required,oneof=loteria quiniela"`
	Descripcion   string `json:"descripcion"`
	CostoNumero   int64  `json:"costo_numero"`
	LongitudNum   int    `json:"longitud_num"`
	MaxNumeros    int    `json:"max_numeros"`
	Multiplicador int64  `json:"multiplicador"`
	Activa        *bool  `json:"activa"`
}

func (h *AdminHandler) CreateLoteria(c *gin.Context) {
	var req loteriaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	l := models.Loteria{
		Codigo:        req.Codigo,
		Nombre:        req.Nombre,
		Tipo:          req.Tipo,
		Descripcion:   req.Descripcion,
		CostoNumero:   orDefault(req.CostoNumero, 1000),
		LongitudNum:   orDefaultInt(req.LongitudNum, 2),
		MaxNumeros:    orDefaultInt(req.MaxNumeros, 1),
		Multiplicador: orDefault(req.Multiplicador, 50),
		Activa:        true,
	}
	if req.Activa != nil {
		l.Activa = *req.Activa
	}

	if err := h.db.Create(&l).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "no se pudo crear (codigo ya existente?)"})
		return
	}
	c.JSON(http.StatusCreated, l)
}

func (h *AdminHandler) UpdateLoteria(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	var req loteriaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}

	updates := map[string]any{
		"codigo":        req.Codigo,
		"nombre":        req.Nombre,
		"tipo":          req.Tipo,
		"descripcion":   req.Descripcion,
		"costo_numero":  req.CostoNumero,
		"longitud_num":  req.LongitudNum,
		"max_numeros":   req.MaxNumeros,
		"multiplicador": req.Multiplicador,
	}
	if req.Activa != nil {
		updates["activa"] = *req.Activa
	}

	if err := h.db.Model(&models.Loteria{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar"})
		return
	}
	var l models.Loteria
	_ = h.db.First(&l, id)
	c.JSON(http.StatusOK, l)
}

type sorteoRequest struct {
	LoteriaID uint   `json:"loteria_id" binding:"required"`
	Titulo    string `json:"titulo" binding:"required"`
	Fecha     string `json:"fecha" binding:"required"`
}

func (h *AdminHandler) CreateSorteo(c *gin.Context) {
	var req sorteoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	fecha, err := time.Parse(time.RFC3339, req.Fecha)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha invalida (usar RFC3339)"})
		return
	}

	var loteria models.Loteria
	if err := h.db.First(&loteria, req.LoteriaID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "loteria no encontrada"})
		return
	}

	s := models.Sorteo{
		LoteriaID: req.LoteriaID,
		Titulo:    req.Titulo,
		Fecha:     fecha,
		Estado:    models.SorteoAbierto,
	}
	if err := h.db.Create(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear el sorteo"})
		return
	}
	c.JSON(http.StatusCreated, s)
}

type cerrarSorteoRequest struct {
	NumerosGanadores []int `json:"numeros_ganadores" binding:"required,min=1,dive,min=0"`
}

// CerrarSorteo fija los números ganadores, evalúa las apuestas pendientes y
// paga los premios. Lo pueden agente y administrador.
func (h *AdminHandler) CerrarSorteo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	var req cerrarSorteoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "numeros ganadores requeridos"})
		return
	}

	var sorteo models.Sorteo
	if err := h.db.Preload("Loteria").First(&sorteo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sorteo no encontrado"})
		return
	}
	if sorteo.Estado != models.SorteoAbierto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el sorteo ya fue cerrado"})
		return
	}

	if len(req.NumerosGanadores) != uniqCount(req.NumerosGanadores) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "los numeros ganadores no pueden repetirse"})
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		sorteo.NumerosGanadores = req.NumerosGanadores
		sorteo.Estado = models.SorteoPagado
		if err := tx.Save(&sorteo).Error; err != nil {
			return err
		}

		var apuestas []models.Apuesta
		if err := tx.Where("sorteo_id = ? AND estado = ?", sorteo.ID, models.ApuestaPendiente).
			Find(&apuestas).Error; err != nil {
			return err
		}

		for _, a := range apuestas {
			aciertos := intersect(a.Numeros, req.NumerosGanadores)
			estado := models.ApuestaPerdedora
			var premio int64

			switch {
			case aciertos == len(a.Numeros) && aciertos > 0:
				estado = models.ApuestaPagada
				premio = a.Monto * sorteo.Loteria.Multiplicador
			case aciertos > 0:
				estado = models.ApuestaGanadora
				premio = a.Monto * 2
			}

			if err := tx.Model(&models.Apuesta{}).Where("id = ?", a.ID).Updates(map[string]any{
				"estado": estado, "aciertos": aciertos, "premio": premio,
			}).Error; err != nil {
				return err
			}

			if premio > 0 {
				if err := tx.Model(&models.User{}).Where("id = ?", a.UserID).
					Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
					return err
				}
				if err := tx.Create(&models.Movimiento{
					UserID: a.UserID, Tipo: models.MovPremio, Monto: premio,
					Descripcion: "Premio " + sorteo.Titulo,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar el sorteo"})
		return
	}

	var actualizado models.Sorteo
	_ = h.db.Preload("Loteria").First(&actualizado, id)
	c.JSON(http.StatusOK, actualizado)
}

func intersect(a, b []int) int {
	set := make(map[int]bool, len(a))
	for _, n := range a {
		set[n] = true
	}
	total := 0
	for _, n := range b {
		if set[n] {
			total++
		}
	}
	return total
}

func orDefault(v, def int64) int64 {
	if v == 0 {
		return def
	}
	return v
}

func orDefaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}
