package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
)

type ApuestasHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewApuestasHandler(db *gorm.DB, cfg *config.Config) *ApuestasHandler {
	return &ApuestasHandler{db: db, cfg: cfg}
}

type crearApuestaRequest struct {
	SorteoID uint  `json:"sorteo_id" binding:"required"`
	Numeros  []int `json:"numeros" binding:"required,min=1,dive,min=0"`
}

func (h *ApuestasHandler) Crear(c *gin.Context) {
	var req crearApuestaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	if len(req.Numeros) != uniqCount(req.Numeros) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "los numeros no pueden repetirse"})
		return
	}

	userID := c.GetUint("userID")

	var sorteo models.Sorteo
	if err := h.db.Preload("Loteria").First(&sorteo, req.SorteoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sorteo no encontrado"})
		return
	}
	if sorteo.Estado != models.SorteoAbierto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el sorteo no esta abierto"})
		return
	}
	// El acceso por paquete o por jugador manda sobre el sorteo: si la loteria
	// esta apagada para esta cuenta, no se puede apostar aunque el sorteo siga
	// abierto (403 claro, no alcanza con ocultarlo en la web).
	if !ExigirAcceso(c, h.db, userID, models.ContenidoLoteria, sorteo.Loteria.Codigo) {
		return
	}

	max := 1
	for i := 0; i < sorteo.Loteria.LongitudNum; i++ {
		max *= 10
	}
	for _, n := range req.Numeros {
		if n >= max {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cada numero debe tener a lo sumo " + strconv.Itoa(sorteo.Loteria.LongitudNum) + " digitos"})
			return
		}
	}
	if len(req.Numeros) > sorteo.Loteria.MaxNumeros {
		c.JSON(http.StatusBadRequest, gin.H{"error": "puedes jugar a lo sumo " + strconv.Itoa(sorteo.Loteria.MaxNumeros) + " numeros"})
		return
	}

	monto := sorteo.Loteria.CostoNumero * int64(len(req.Numeros))

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	if !GuardJugada(c, h.cfg, user) {
		return
	}
	if user.Saldo < monto {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente", "monto": monto, "saldo": user.Saldo})
		return
	}
	// Limites voluntarios de gasto del jugador (juego responsable).
	if excede, disponible, motivo := EvaluarGasto(h.db, userID, monto); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu limite " + motivo + " de gasto se excederia con esta apuesta",
			"disponible": disponible, "motivo": motivo,
		})
		return
	}

	var idApuesta uint
	err := h.db.Transaction(func(tx *gorm.DB) error {
		apuesta := models.Apuesta{
			UserID:   userID,
			SorteoID: sorteo.ID,
			Numeros:  req.Numeros,
			Monto:    monto,
			Estado:   models.ApuestaPendiente,
		}
		if err := tx.Create(&apuesta).Error; err != nil {
			return err
		}
		idApuesta = apuesta.ID
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("saldo", gorm.Expr("saldo - ?", monto)).Error; err != nil {
			return err
		}
		mov := models.Movimiento{
			UserID:      userID,
			Tipo:        models.MovApuesta,
			Monto:       -monto,
			Descripcion: "Apuesta " + sorteo.Titulo,
		}
		return tx.Create(&mov).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar la apuesta"})
		return
	}

	RegistrarAccionJugador(h.db, c, userID, "apuesta_crear", "apuesta",
		strconv.FormatUint(uint64(idApuesta), 10),
		"sorteo="+strconv.FormatUint(uint64(sorteo.ID), 10)+" monto="+strconv.FormatInt(monto, 10)+
			" numeros="+joinInts(req.Numeros))

	var actualizado models.User
	_ = h.db.First(&actualizado, userID)

	c.JSON(http.StatusCreated, gin.H{
		"monto": monto,
		"saldo": actualizado.Saldo,
	})
}

func (h *ApuestasHandler) Mias(c *gin.Context) {
	userID := c.GetUint("userID")
	pagina, porPagina := leerPaginacion(c)

	var total int64
	if err := h.db.Model(&models.Apuesta{}).
		Where("user_id = ?", userID).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las apuestas"})
		return
	}

	var apuestas []models.Apuesta
	err := h.db.Preload("Sorteo.Loteria").
		Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(porPagina).
		Offset((pagina - 1) * porPagina).
		Find(&apuestas).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las apuestas"})
		return
	}
	respuestaPagina(c, apuestas, total, pagina, porPagina)
}

func uniqCount(nums []int) int {
	seen := map[int]bool{}
	for _, n := range nums {
		seen[n] = true
	}
	return len(seen)
}
