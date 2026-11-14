package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

type ReportesHandler struct {
	db *gorm.DB
}

func NewReportesHandler(db *gorm.DB) *ReportesHandler {
	return &ReportesHandler{db: db}
}

type DashboardResponse struct {
	Saldo          int64              `json:"saldo"`
	Bono           int64              `json:"bono"`
	Pendiente      int64              `json:"pendiente"`
	Premios        int64              `json:"premios"`
	TotalApostado  int64              `json:"total_apostado"`
	ApuestasActivas int64             `json:"apuestas_activas"`
	Ultimas        []models.Apuesta   `json:"ultimas_apuestas"`
}

func (h *ReportesHandler) Dashboard(c *gin.Context) {
	userID := c.GetUint("userID")

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}

	var pendiente, premios, totalApostado int64
	var activas int64

	_ = h.db.Model(&models.Apuesta{}).
		Where("user_id = ? AND estado = ?", userID, models.ApuestaPendiente).
		Select("COALESCE(SUM(monto),0)").Scan(&pendiente).Error
	_ = h.db.Model(&models.Apuesta{}).
		Where("user_id = ? AND estado = ?", userID, models.ApuestaPendiente).
		Count(&activas).Error
	_ = h.db.Model(&models.Apuesta{}).
		Where("user_id = ? AND estado IN ?", userID, []string{models.ApuestaGanadora, models.ApuestaPagada}).
		Select("COALESCE(SUM(premio),0)").Scan(&premios).Error
	_ = h.db.Model(&models.Apuesta{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(monto),0)").Scan(&totalApostado).Error

	var ultimas []models.Apuesta
	_ = h.db.Preload("Sorteo.Loteria").
		Where("user_id = ?", userID).
		Order("created_at desc").Limit(5).Find(&ultimas).Error

	c.JSON(http.StatusOK, DashboardResponse{
		Saldo:           user.Saldo,
		Bono:            user.Bono,
		Pendiente:       pendiente,
		Premios:         premios,
		TotalApostado:   totalApostado,
		ApuestasActivas: activas,
		Ultimas:         ultimas,
	})
}

func (h *ReportesHandler) Movimientos(c *gin.Context) {
	userID := c.GetUint("userID")
	var movs []models.Movimiento
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(100).Find(&movs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los movimientos"})
		return
	}
	c.JSON(http.StatusOK, movs)
}

func (h *ReportesHandler) Resultados(c *gin.Context) {
	var sorteos []models.Sorteo
	if err := h.db.Preload("Loteria").
		Where("estado IN ?", []string{models.SorteoCerrado, models.SorteoPagado}).
		Order("fecha desc").Limit(50).Find(&sorteos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los resultados"})
		return
	}

	userID := c.GetUint("userID")
	var mias []models.Apuesta
	_ = h.db.Where("user_id = ?", userID).Find(&mias).Error

	aciertosPorSorteo := map[uint]models.Apuesta{}
	for _, a := range mias {
		if _, ok := aciertosPorSorteo[a.SorteoID]; !ok {
			aciertosPorSorteo[a.SorteoID] = a
		}
	}

	type fila struct {
		Sorteo   models.Sorteo `json:"sorteo"`
		Aciertos int           `json:"aciertos"`
		Premio   int64         `json:"premio"`
		Estado   string        `json:"estado_apuesta"`
	}

	filas := make([]fila, 0, len(sorteos))
	for _, s := range sorteos {
		a, ok := aciertosPorSorteo[s.ID]
		if !ok {
			continue
		}
		filas = append(filas, fila{Sorteo: s, Aciertos: a.Aciertos, Premio: a.Premio, Estado: a.Estado})
	}

	c.JSON(http.StatusOK, gin.H{"resultados": filas, "sorteos": sorteos})
}

func (h *ReportesHandler) Resumen(c *gin.Context) {
	rol, _ := c.Get("rol")

	type porLoteria struct {
		LoteriaID   uint   `json:"loteria_id"`
		Nombre      string `json:"nombre"`
		Tipo        string `json:"tipo"`
		Apostado    int64  `json:"apostado"`
		Premios     int64  `json:"premios"`
		Apuestas    int64  `json:"apuestas"`
		Sorteos     int64  `json:"sorteos"`
	}

	var filas []porLoteria
	q := h.db.Table("apuestas a").
		Select("l.id AS loteria_id, l.nombre AS nombre, l.tipo AS tipo, COALESCE(SUM(a.monto),0) AS apostado, COALESCE(SUM(a.premio),0) AS premios, COUNT(a.id) AS apuestas").
		Joins("JOIN sorteos s ON s.id = a.sorteo_id").
		Joins("JOIN loterias l ON l.id = s.loteria_id").
		Group("l.id, l.nombre, l.tipo").
		Order("apostado desc")

	if rol == models.RolCliente {
		q = q.Where("a.user_id = ?", c.GetUint("userID"))
	}

	if err := q.Scan(&filas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar el resumen"})
		return
	}

	var totalApostado, totalPremios, totalApuestas int64
	var loteriasActivas, usuariosTotales int64
	_ = h.db.Model(&models.Apuesta{}).Select("COALESCE(SUM(monto),0)").Scan(&totalApostado).Error
	_ = h.db.Model(&models.Apuesta{}).Select("COALESCE(SUM(premio),0)").Scan(&totalPremios).Error
	_ = h.db.Model(&models.Apuesta{}).Count(&totalApuestas).Error
	_ = h.db.Model(&models.Loteria{}).Where("activa = ?", true).Count(&loteriasActivas).Error
	_ = h.db.Model(&models.User{}).Count(&usuariosTotales).Error

	c.JSON(http.StatusOK, gin.H{
		"por_loteria":     filas,
		"total_apostado":  totalApostado,
		"total_premios":   totalPremios,
		"total_apuestas":  totalApuestas,
		"loterias_activas": loteriasActivas,
		"usuarios":        usuariosTotales,
	})
}
