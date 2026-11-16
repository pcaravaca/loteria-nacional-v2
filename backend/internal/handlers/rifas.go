package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// RifasHandler expone las rifas al publico y permite comprar numeros.
type RifasHandler struct {
	db *gorm.DB
}

func NewRifasHandler(db *gorm.DB) *RifasHandler { return &RifasHandler{db: db} }

// ganadores devuelve los ganadores de una rifa ya sorteada.
func rifasGanadores(db *gorm.DB, r models.Rifa) []models.GanadorRifa {
	out := []models.GanadorRifa{}
	if len(r.NumerosGanadores) == 0 {
		return out
	}
	var nums []models.RifaNumero
	if err := db.Where("rifa_id = ? AND numero IN ? AND user_id IS NOT NULL", r.ID, r.NumerosGanadores).
		Order("numero asc").Find(&nums).Error; err != nil {
		return out
	}
	for _, n := range nums {
		var u models.User
		if err := db.First(&u, *n.UserID).Error; err != nil {
			continue
		}
		out = append(out, models.GanadorRifa{Numero: n.Numero, Nombre: u.Nombre})
	}
	return out
}

// Listar devuelve todas las rifas con el progreso de venta.
func (h *RifasHandler) Listar(c *gin.Context) {
	var rifas []models.Rifa
	if err := h.db.Order("estado asc, fecha_sorteo asc").Find(&rifas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las rifas"})
		return
	}
	type salida struct {
		models.Rifa
		Vendidos  int                `json:"vendidos"`
		Total     int                `json:"total"`
		Ganadores []models.GanadorRifa `json:"ganadores,omitempty"`
	}
	out := make([]salida, 0, len(rifas))
	for _, r := range rifas {
		total := r.HastaNumero - r.DesdeNumero + 1
		var vendidos int64
		h.db.Model(&models.RifaNumero{}).Where("rifa_id = ? AND user_id IS NOT NULL", r.ID).Count(&vendidos)
		s := salida{Rifa: r, Vendidos: int(vendidos), Total: total}
		if r.Estado == models.RifaSorteada {
			s.Ganadores = rifasGanadores(h.db, r)
		}
		out = append(out, s)
	}
	c.JSON(http.StatusOK, out)
}

// Detalle devuelve la rifa y su mapa de numeros (disponible/vendido/mio).
func (h *RifasHandler) Detalle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var rifa models.Rifa
	if err := h.db.First(&rifa, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rifa no encontrada"})
		return
	}
	var nums []models.RifaNumero
	if err := h.db.Where("rifa_id = ?", rifa.ID).Order("numero asc").Find(&nums).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los numeros"})
		return
	}
	mio := c.GetUint("userID")
	type numero struct {
		Numero  int  `json:"numero"`
		Vendido bool `json:"vendido"`
		Mio     bool `json:"mio"`
	}
	out := make([]numero, 0, len(nums))
	for _, n := range nums {
		out = append(out, numero{
			Numero:  n.Numero,
			Vendido: n.UserID != nil,
			Mio:     mio != 0 && n.UserID != nil && *n.UserID == mio,
		})
	}
	c.JSON(http.StatusOK, gin.H{"rifa": rifa, "numeros": out})
}

// Comprar asigna numeros de una rifa abierta al usuario autenticado.
func (h *RifasHandler) Comprar(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Numeros []int `json:"numeros"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Numeros) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "indicá al menos un numero"})
		return
	}
	if len(req.Numeros) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "máximo 50 numeros por compra"})
		return
	}
	vistos := map[int]bool{}
	for _, n := range req.Numeros {
		if vistos[n] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "los numeros no pueden repetirse"})
			return
		}
		vistos[n] = true
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var rifa models.Rifa
	if err := h.db.First(&rifa, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rifa no encontrada"})
		return
	}
	if rifa.Estado != models.RifaAbierta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la rifa no está abierta"})
		return
	}
	for _, n := range req.Numeros {
		if n < rifa.DesdeNumero || n > rifa.HastaNumero {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hay numeros fuera del rango de la rifa"})
			return
		}
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}

	total := rifa.PrecioNumero * int64(len(req.Numeros))
	if excede, disponible, motivo := EvaluarGasto(h.db, userID, total); excede {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      "tu límite " + motivo + " de gasto se excedería con esta compra",
			"disponible": disponible, "motivo": motivo,
		})
		return
	}
	if user.Saldo < total {
		c.JSON(http.StatusBadRequest, gin.H{"error": "saldo insuficiente", "saldo": user.Saldo, "monto": total})
		return
	}

	var existentes []models.RifaNumero
	if err := h.db.Where("rifa_id = ? AND numero IN ?", rifa.ID, req.Numeros).Find(&existentes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron validar los numeros"})
		return
	}
	if len(existentes) != len(req.Numeros) {
		c.JSON(http.StatusConflict, gin.H{"error": "algun numero no existe en la rifa"})
		return
	}
	for _, n := range existentes {
		if n.UserID != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "algun numero ya fue vendido"})
			return
		}
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		ahora := time.Now()
		for _, n := range existentes {
			if err := tx.Model(&models.RifaNumero{}).Where("id = ?", n.ID).Updates(map[string]any{
				"user_id": userID, "monto": rifa.PrecioNumero, "comprado_en": ahora,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("saldo", gorm.Expr("saldo - ?", total)).Error; err != nil {
			return err
		}
		return tx.Create(&models.Movimiento{
			UserID: userID, Tipo: models.MovRifaCompra, Monto: -total,
			Descripcion: "Rifa " + rifa.Titulo,
		}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar la compra"})
		return
	}

	RegistrarAuditoria(h.db, c, userID, "rifa.comprar", "rifa",
		strconv.FormatUint(uint64(rifa.ID), 10),
		"numeros="+joinInts(req.Numeros)+" total="+strconv.FormatInt(total, 10))

	var actualizado models.User
	_ = h.db.First(&actualizado, userID)
	c.JSON(http.StatusOK, gin.H{
		"saldo":   actualizado.Saldo,
		"rifa_id": rifa.ID,
		"numeros": req.Numeros,
		"total":   total,
	})
}

// Mias devuelve los numeros comprados por el usuario, agrupados por rifa.
func (h *RifasHandler) Mias(c *gin.Context) {
	userID := c.GetUint("userID")
	var nums []models.RifaNumero
	if err := h.db.Where("user_id = ?", userID).Preload("Rifa").Order("comprado_en desc").Find(&nums).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar tus numeros"})
		return
	}
	type item struct {
		RifaID   uint   `json:"rifa_id"`
		Titulo   string `json:"titulo"`
		Premio   string `json:"premio"`
		Estado   string `json:"estado"`
		Numero   int    `json:"numero"`
		Monto    int64  `json:"monto"`
		Fecha    string `json:"fecha_sorteo"`
		Ganador  bool   `json:"ganador"`
	}
	out := make([]item, 0, len(nums))
	for _, n := range nums {
		ganador := false
		for _, g := range n.Rifa.NumerosGanadores {
			if g == n.Numero {
				ganador = true
				break
			}
		}
		out = append(out, item{
			RifaID: n.RifaID, Titulo: n.Rifa.Titulo, Premio: n.Rifa.Premio, Estado: n.Rifa.Estado,
			Numero: n.Numero, Monto: n.Monto, Fecha: n.Rifa.FechaSorteo.Format(time.RFC3339), Ganador: ganador,
		})
	}
	c.JSON(http.StatusOK, out)
}

func joinInts(nums []int) string {
	partes := make([]string, 0, len(nums))
	for _, n := range nums {
		partes = append(partes, strconv.Itoa(n))
	}
	return strings.Join(partes, ",")
}
