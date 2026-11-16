package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// EdadMinima es la edad requerida para jugar.
const EdadMinima = 18

// LimitesHandler gestiona los limites voluntarios del jugador (juego
// responsable) y su historial queda en la auditoria.
type LimitesHandler struct {
	db *gorm.DB
}

func NewLimitesHandler(db *gorm.DB) *LimitesHandler { return &LimitesHandler{db: db} }

// CargarLimites devuelve los limites del usuario; si nunca los configuro,
// devuelve el valor cero (sin limites).
func CargarLimites(db *gorm.DB, userID uint) models.LimitesJugador {
	var l models.LimitesJugador
	if err := db.First(&l, userID).Error; err != nil {
		return models.LimitesJugador{UserID: userID}
	}
	return l
}

func (h *LimitesHandler) Obtener(c *gin.Context) {
	c.JSON(http.StatusOK, CargarLimites(h.db, c.GetUint("userID")))
}

type limitesRequest struct {
	DepositoDiario  *int64 `json:"deposito_diario"`
	DepositoSemanal *int64 `json:"deposito_semanal"`
	DepositoMensual *int64 `json:"deposito_mensual"`
	GastoDiario     *int64 `json:"gasto_diario"`
	GastoSemanal    *int64 `json:"gasto_semanal"`
	GastoMensual    *int64 `json:"gasto_mensual"`
}

func (h *LimitesHandler) Actualizar(c *gin.Context) {
	userID := c.GetUint("userID")
	var req limitesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}
	for nombre, v := range map[string]*int64{
		"deposito_diario": req.DepositoDiario, "deposito_semanal": req.DepositoSemanal,
		"deposito_mensual": req.DepositoMensual, "gasto_diario": req.GastoDiario,
		"gasto_semanal": req.GastoSemanal, "gasto_mensual": req.GastoMensual,
	} {
		if v != nil && *v < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": nombre + " no puede ser negativo"})
			return
		}
	}

	actual := CargarLimites(h.db, userID)
	before, _ := json.Marshal(actual)

	nueva := models.LimitesJugador{
		UserID:          userID,
		DepositoDiario:  limiteO(req.DepositoDiario, actual.DepositoDiario),
		DepositoSemanal: limiteO(req.DepositoSemanal, actual.DepositoSemanal),
		DepositoMensual: limiteO(req.DepositoMensual, actual.DepositoMensual),
		GastoDiario:     limiteO(req.GastoDiario, actual.GastoDiario),
		GastoSemanal:    limiteO(req.GastoSemanal, actual.GastoSemanal),
		GastoMensual:    limiteO(req.GastoMensual, actual.GastoMensual),
		ActualizadoEn:   time.Now(),
	}
	if err := h.db.Save(&nueva).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron guardar los limites"})
		return
	}

	after, _ := json.Marshal(nueva)
	RegistrarAuditoria(h.db, c, userID, "limites.actualizar", "usuario",
		strconv.FormatUint(uint64(userID), 10), string(before)+" -> "+string(after))

	c.JSON(http.StatusOK, nueva)
}

func limiteO(v *int64, actual int64) int64 {
	if v == nil {
		return actual
	}
	return *v
}

// ---------------------------------------------------------------- periodos

// gastoPeriodo suma lo apostado (loteria y casino) en el tramo indicado.
func gastoPeriodo(db *gorm.DB, userID uint, desde, hasta time.Time) int64 {
	var total int64
	_ = db.Model(&models.Movimiento{}).
		Where("user_id = ? AND tipo IN ? AND created_at BETWEEN ? AND ?",
			userID, []string{models.MovApuesta, models.MovCasinoApuesta}, desde, hasta).
		Select("COALESCE(SUM(-monto),0)").Scan(&total).Error
	return total
}

// depositoPeriodo suma los depositos (recargas) en el tramo indicado.
func depositoPeriodo(db *gorm.DB, userID uint, desde, hasta time.Time) int64 {
	var total int64
	_ = db.Model(&models.Movimiento{}).
		Where("user_id = ? AND tipo = ? AND created_at BETWEEN ? AND ?",
			userID, models.MovRecarga, desde, hasta).
		Select("COALESCE(SUM(monto),0)").Scan(&total).Error
	return total
}

// ---------------------------------------------------------------- evaluacion

type limitePar struct {
	Nombre string
	Limite int64
	Usado  int64
}

// ParesGasto arma los pares limite/usado vigentes para gastos de juego.
func ParesGasto(db *gorm.DB, userID uint, l models.LimitesJugador) []limitePar {
	ahora := time.Now().In(tiempo.CR)
	return []limitePar{
		{Nombre: "diario", Limite: l.GastoDiario, Usado: gastoPeriodo(db, userID, tiempo.InicioDia(), ahora)},
		{Nombre: "semanal", Limite: l.GastoSemanal, Usado: gastoPeriodo(db, userID, tiempo.InicioSemana(), ahora)},
		{Nombre: "mensual", Limite: l.GastoMensual, Usado: gastoPeriodo(db, userID, tiempo.InicioMes(), ahora)},
	}
}

// ParesDeposito arma los pares limite/usado vigentes para depositos.
func ParesDeposito(db *gorm.DB, userID uint, l models.LimitesJugador) []limitePar {
	ahora := time.Now().In(tiempo.CR)
	return []limitePar{
		{Nombre: "diario", Limite: l.DepositoDiario, Usado: depositoPeriodo(db, userID, tiempo.InicioDia(), ahora)},
		{Nombre: "semanal", Limite: l.DepositoSemanal, Usado: depositoPeriodo(db, userID, tiempo.InicioSemana(), ahora)},
		{Nombre: "mensual", Limite: l.DepositoMensual, Usado: depositoPeriodo(db, userID, tiempo.InicioMes(), ahora)},
	}
}

// EvaluarLimites indica si el intento cabe en todos los limites aplicados y
// cuanto queda del periodo mas exigente.
func EvaluarLimites(pares []limitePar, intento int64) (excede bool, disponible int64, motivo string) {
	aplicados := 0
	disponible = 0
	for _, p := range pares {
		if p.Limite <= 0 {
			continue
		}
		aplicados++
		resta := p.Limite - p.Usado
		if resta < disponible || aplicados == 1 {
			disponible = resta
		}
		if p.Usado+intento > p.Limite && !excede {
			excede = true
			motivo = p.Nombre
		}
	}
	if aplicados == 0 {
		disponible = 0
	}
	return excede, disponible, motivo
}

// EvaluarGasto aplica los limites de gasto del jugador a una jugada.
func EvaluarGasto(db *gorm.DB, userID uint, intento int64) (excede bool, disponible int64, motivo string) {
	l := CargarLimites(db, userID)
	return EvaluarLimites(ParesGasto(db, userID, l), intento)
}

// EvaluarDeposito aplica los limites de deposito del jugador a una recarga.
func EvaluarDeposito(db *gorm.DB, userID uint, intento int64) (excede bool, disponible int64, motivo string) {
	l := CargarLimites(db, userID)
	return EvaluarLimites(ParesDeposito(db, userID, l), intento)
}

// EdadEn calcula la edad cumplida en la fecha indicada.
func EdadEn(fechaNacimiento, hoy time.Time) int {
	edad := hoy.Year() - fechaNacimiento.Year()
	cumple := time.Date(hoy.Year(), fechaNacimiento.Month(), fechaNacimiento.Day(), 0, 0, 0, 0, hoy.Location())
	if hoy.Before(cumple) {
		edad--
	}
	return edad
}
