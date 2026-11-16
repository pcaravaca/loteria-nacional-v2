package handlers

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// AdminRifasHandler administra las rifas (crear, sortear, cancelar).
type AdminRifasHandler struct {
	db *gorm.DB
}

func NewAdminRifasHandler(db *gorm.DB) *AdminRifasHandler { return &AdminRifasHandler{db: db} }

// Crear crea una rifa con su stock de numeros.
func (h *AdminRifasHandler) Crear(c *gin.Context) {
	var req struct {
		Titulo        string `json:"titulo" binding:"required"`
		Descripcion   string `json:"descripcion"`
		Premio        string `json:"premio" binding:"required"`
		PremioMonto   int64  `json:"premio_monto"`
		PrecioNumero  int64  `json:"precio_numero" binding:"required"`
		DesdeNumero   int    `json:"desde_numero"`
		HastaNumero   int    `json:"hasta_numero"`
		FechaSorteo   string `json:"fecha_sorteo" binding:"required"`
		TipoSorteo    string `json:"tipo_sorteo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	if req.PrecioNumero <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el precio debe ser mayor a cero"})
		return
	}
	if req.DesdeNumero < 0 || req.HastaNumero <= req.DesdeNumero {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rango de numeros invalido"})
		return
	}
	total := req.HastaNumero - req.DesdeNumero + 1
	if total > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el rango no puede superar 5000 numeros"})
		return
	}
	fecha, err := time.Parse(time.RFC3339, req.FechaSorteo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha de sorteo invalida (usá RFC3339)"})
		return
	}
	if !fecha.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la fecha de sorteo debe ser futura"})
		return
	}
	tipo := req.TipoSorteo
	if tipo != models.RifaSorteoManual {
		tipo = models.RifaSorteoAzar
	}

	rifa := models.Rifa{
		Titulo: req.Titulo, Descripcion: req.Descripcion, Premio: req.Premio,
		PremioMonto: req.PremioMonto, PrecioNumero: req.PrecioNumero,
		DesdeNumero: req.DesdeNumero, HastaNumero: req.HastaNumero,
		FechaSorteo: fecha, Estado: models.RifaAbierta, TipoSorteo: tipo,
		CreadoPor: c.GetUint("userID"),
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&rifa).Error; err != nil {
			return err
		}
		numeros := make([]models.RifaNumero, 0, total)
		for n := req.DesdeNumero; n <= req.HastaNumero; n++ {
			numeros = append(numeros, models.RifaNumero{RifaID: rifa.ID, Numero: n})
			if len(numeros) == 200 {
				if err := tx.CreateInBatches(numeros, 200).Error; err != nil {
					return err
				}
				numeros = numeros[:0]
			}
		}
		if len(numeros) > 0 {
			return tx.CreateInBatches(numeros, 200).Error
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear la rifa"})
		return
	}

	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "rifa.crear", "rifa",
		strconv.FormatUint(uint64(rifa.ID), 10),
		fmt.Sprintf("%d numeros, precio %d, sorteo %s", total, rifa.PrecioNumero, rifa.FechaSorteo.Format(time.RFC3339)))

	c.JSON(http.StatusOK, rifa)
}

// Listar devuelve las rifas con totales de venta y recaudo (vista admin).
func (h *AdminRifasHandler) Listar(c *gin.Context) {
	var rifas []models.Rifa
	if err := h.db.Order("estado asc, fecha_sorteo asc").Find(&rifas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las rifas"})
		return
	}
	type salida struct {
		models.Rifa
		Vendidos  int                 `json:"vendidos"`
		Total     int                 `json:"total"`
		Recaudado int64               `json:"recaudado"`
		Ganadores []models.GanadorRifa `json:"ganadores,omitempty"`
	}
	out := make([]salida, 0, len(rifas))
	for _, r := range rifas {
		total := r.HastaNumero - r.DesdeNumero + 1
		var vendidos int64
		var recaudado int64
		h.db.Model(&models.RifaNumero{}).Where("rifa_id = ? AND user_id IS NOT NULL", r.ID).Count(&vendidos)
		h.db.Model(&models.RifaNumero{}).Where("rifa_id = ? AND user_id IS NOT NULL", r.ID).Select("COALESCE(SUM(monto),0)").Scan(&recaudado)
		s := salida{Rifa: r, Vendidos: int(vendidos), Total: total, Recaudado: recaudado}
		if r.Estado == models.RifaSorteada {
			s.Ganadores = rifasGanadores(h.db, r)
		}
		out = append(out, s)
	}
	c.JSON(http.StatusOK, out)
}

// Sortear cierra la rifa y define los numeros ganadores. Sin numeros en el
// cuerpo, se elige al azar entre los vendidos. Premia en efectivo si la rifa
// tiene premio_monto.
func (h *AdminRifasHandler) Sortear(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "la rifa ya fue sorteada o cancelada"})
		return
	}

	var vendidos []models.RifaNumero
	if err := h.db.Where("rifa_id = ? AND user_id IS NOT NULL", rifa.ID).Order("numero asc").Find(&vendidos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los numeros vendidos"})
		return
	}
	if len(vendidos) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no hay numeros vendidos para sortear"})
		return
	}

	var req struct {
		Numeros []int `json:"numeros"`
	}
	_ = c.ShouldBindJSON(&req)

	ganadores := make([]int, 0, len(req.Numeros))
	if len(req.Numeros) > 0 {
		vendidosPorNumero := map[int]models.RifaNumero{}
		for _, v := range vendidos {
			vendidosPorNumero[v.Numero] = v
		}
		vistos := map[int]bool{}
		for _, n := range req.Numeros {
			if vistos[n] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "los numeros ganadores no pueden repetirse"})
				return
			}
			vistos[n] = true
			if _, ok := vendidosPorNumero[n]; !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "el numero " + strconv.Itoa(n) + " no está vendido"})
				return
			}
			ganadores = append(ganadores, n)
		}
	} else {
		// Azar provably razonable: crypto/rand sobre los vendidos.
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(vendidos))))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo elegir el numero"})
			return
		}
		ganadores = []int{vendidos[idx.Int64()].Numero}
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		for _, num := range ganadores {
			var ganador models.RifaNumero
			if err := tx.Where("rifa_id = ? AND numero = ?", rifa.ID, num).First(&ganador).Error; err != nil {
				return err
			}
			if rifa.PremioMonto > 0 && ganador.UserID != nil {
				if err := tx.Model(&models.User{}).Where("id = ?", *ganador.UserID).
					Update("saldo", gorm.Expr("saldo + ?", rifa.PremioMonto)).Error; err != nil {
					return err
				}
				if err := tx.Create(&models.Movimiento{
					UserID: *ganador.UserID, Tipo: models.MovRifaPremio, Monto: rifa.PremioMonto,
					Descripcion: "Premio rifa " + rifa.Titulo,
				}).Error; err != nil {
					return err
				}
			}
		}
		rifa.Estado = models.RifaSorteada
		rifa.NumerosGanadores = ganadores
		return tx.Save(&rifa).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar el sorteo"})
		return
	}

	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "rifa.sortear", "rifa",
		strconv.FormatUint(uint64(rifa.ID), 10), "numeros="+joinInts(ganadores))

	c.JSON(http.StatusOK, gin.H{
		"rifa":      rifa,
		"ganadores": rifasGanadores(h.db, rifa),
	})
}

// Cancelar cierra la rifa y reembolsa todos los numeros vendidos.
func (h *AdminRifasHandler) Cancelar(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "la rifa ya fue sorteada o cancelada"})
		return
	}

	var vendidos []models.RifaNumero
	if err := h.db.Where("rifa_id = ? AND user_id IS NOT NULL", rifa.ID).Find(&vendidos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los numeros vendidos"})
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		for _, v := range vendidos {
			if v.UserID == nil {
				continue
			}
			if err := tx.Model(&models.User{}).Where("id = ?", *v.UserID).
				Update("saldo", gorm.Expr("saldo + ?", v.Monto)).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.Movimiento{
				UserID: *v.UserID, Tipo: models.MovRifaReembolso, Monto: v.Monto,
				Descripcion: "Reembolso rifa " + rifa.Titulo,
			}).Error; err != nil {
				return err
			}
		}
		rifa.Estado = models.RifaCancelada
		return tx.Save(&rifa).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cancelar la rifa"})
		return
	}

	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "rifa.cancelar", "rifa",
		strconv.FormatUint(uint64(rifa.ID), 10), fmt.Sprintf("reembolsados %d numeros", len(vendidos)))

	c.JSON(http.StatusOK, rifa)
}
