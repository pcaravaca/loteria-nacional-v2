package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// ExportacionesHandler genera reportes CSV para la fiscalizacion. Todos los
// metodos se montan sobre el grupo de admin y quedan en la auditoria cuando
// el operador los descarga.
type ExportacionesHandler struct {
	db *gorm.DB
}

func NewExportacionesHandler(db *gorm.DB) *ExportacionesHandler {
	return &ExportacionesHandler{db: db}
}

// escribirCSV vuela el archivo con BOM para que Excel respete el UTF-8.
func escribirCSV(c *gin.Context, nombre string, headers []string, filas [][]string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, nombre))
	_, _ = c.Writer.WriteString("\ufeff")
	w := csv.NewWriter(c.Writer)
	_ = w.Write(headers)
	for _, f := range filas {
		_ = w.Write(f)
	}
	w.Flush()
}

func formatoTiempoCR(t time.Time) string {
	return t.In(tiempo.CR).Format("2006-01-02 15:04:05")
}

func numerosTexto(nums []int) string {
	partes := make([]string, 0, len(nums))
	for _, n := range nums {
		partes = append(partes, strconv.Itoa(n))
	}
	return strings.Join(partes, " ")
}

// Apuestas exporta el detalle de apuestas con filtros por sorteo, loteria,
// estado, usuario y rango de fechas.
func (h *ExportacionesHandler) Apuestas(c *gin.Context) {
	q := h.db.Model(&models.Apuesta{})
	if v := c.Query("sorteo_id"); v != "" {
		q = q.Where("sorteo_id = ?", v)
	}
	if v := c.Query("user_id"); v != "" {
		q = q.Where("user_id = ?", v)
	}
	if v := c.Query("estado"); v != "" {
		q = q.Where("estado = ?", v)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("created_at BETWEEN ? AND ?", desde, hasta)
	}

	var apuestas []models.Apuesta
	if err := q.Preload("Sorteo.Loteria").Order("created_at desc").
		Limit(10000).Find(&apuestas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo exportar"})
		return
	}

	loteriaID := c.Query("loteria_id")
	filas := make([][]string, 0, len(apuestas))
	for _, a := range apuestas {
		if loteriaID != "" {
			if a.Sorteo.Loteria.ID == 0 || strconv.FormatUint(uint64(a.Sorteo.Loteria.ID), 10) != loteriaID {
				continue
			}
		}
		loteria := ""
		if a.Sorteo.Loteria.ID != 0 {
			loteria = a.Sorteo.Loteria.Codigo
		}
		filas = append(filas, []string{
			strconv.FormatUint(uint64(a.ID), 10),
			strconv.FormatUint(uint64(a.UserID), 10),
			strconv.FormatUint(uint64(a.SorteoID), 10),
			loteria,
			numerosTexto(a.Numeros),
			strconv.FormatInt(a.Monto, 10),
			a.Estado,
			strconv.Itoa(a.Aciertos),
			strconv.FormatInt(a.Premio, 10),
			formatoTiempoCR(a.CreatedAt),
		})
	}
	escribirCSV(c, "apuestas.csv", []string{
		"id", "usuario_id", "sorteo_id", "loteria", "numeros", "monto",
		"estado", "aciertos", "premio", "fecha",
	}, filas)
}

// Sorteos exporta el detalle de sorteos y sus numeros ganadores.
func (h *ExportacionesHandler) Sorteos(c *gin.Context) {
	q := h.db.Model(&models.Sorteo{})
	if v := c.Query("estado"); v != "" {
		q = q.Where("estado = ?", v)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("fecha BETWEEN ? AND ?", desde, hasta)
	}

	var sorteos []models.Sorteo
	if err := q.Preload("Loteria").Order("fecha desc").
		Limit(10000).Find(&sorteos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo exportar"})
		return
	}

	filas := make([][]string, 0, len(sorteos))
	for _, s := range sorteos {
		loteria := ""
		if s.Loteria.ID != 0 {
			loteria = s.Loteria.Codigo
		}
		filas = append(filas, []string{
			strconv.FormatUint(uint64(s.ID), 10),
			strconv.FormatUint(uint64(s.LoteriaID), 10),
			loteria,
			s.Titulo,
			formatoTiempoCR(s.Fecha),
			s.Estado,
			numerosTexto(s.NumerosGanadores),
		})
	}
	escribirCSV(c, "sorteos.csv", []string{
		"id", "loteria_id", "loteria", "titulo", "fecha", "estado", "numeros_ganadores",
	}, filas)
}

// RondasCasino exporta las jugadas del mini casino (sin blackjack).
func (h *ExportacionesHandler) RondasCasino(c *gin.Context) {
	q := h.db.Model(&models.CasinoRound{})
	if v := c.Query("game"); v != "" {
		q = q.Where("game_codigo = ?", v)
	}
	if v := c.Query("user_id"); v != "" {
		q = q.Where("user_id = ?", v)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("created_at BETWEEN ? AND ?", desde, hasta)
	}

	var rondas []models.CasinoRound
	if err := q.Order("created_at desc").Limit(20000).Find(&rondas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo exportar"})
		return
	}

	filas := make([][]string, 0, len(rondas))
	for _, r := range rondas {
		filas = append(filas, []string{
			strconv.FormatUint(uint64(r.ID), 10),
			strconv.FormatUint(uint64(r.UserID), 10),
			r.GameCodigo,
			strconv.FormatInt(r.Monto, 10),
			strconv.FormatInt(r.Multiplicador, 10),
			strconv.FormatInt(r.Premio, 10),
			strconv.FormatInt(r.Nonce, 10),
			r.ClientSeed,
			r.ServerSeedHash,
			formatoTiempoCR(r.CreatedAt),
		})
	}
	escribirCSV(c, "rondas_casino.csv", []string{
		"id", "usuario_id", "juego", "monto", "multiplicador", "premio",
		"nonce", "client_seed", "server_seed_hash", "fecha",
	}, filas)
}

// BlackjackManos exporta las manos de blackjack (multiples turnos).
func (h *ExportacionesHandler) BlackjackManos(c *gin.Context) {
	q := h.db.Model(&models.BlackjackMano{})
	if v := c.Query("user_id"); v != "" {
		q = q.Where("user_id = ?", v)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("created_at BETWEEN ? AND ?", desde, hasta)
	}

	var manos []models.BlackjackMano
	if err := q.Order("created_at desc").Limit(10000).Find(&manos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo exportar"})
		return
	}

	filas := make([][]string, 0, len(manos))
	for _, m := range manos {
		filas = append(filas, []string{
			strconv.FormatUint(uint64(m.ID), 10),
			strconv.FormatUint(uint64(m.UserID), 10),
			strconv.FormatInt(m.Monto, 10),
			m.Estado,
			fmt.Sprintf("%.2f", m.Multiplicador),
			strconv.FormatInt(m.Premio, 10),
			strconv.FormatInt(m.Seguro, 10),
			strconv.FormatInt(m.Nonce, 10),
			formatoTiempoCR(m.CreatedAt),
		})
	}
	escribirCSV(c, "blackjack.csv", []string{
		"id", "usuario_id", "monto", "estado", "multiplicador", "premio",
		"seguro", "nonce", "fecha",
	}, filas)
}

// Movimientos exporta los movimientos de saldo.
func (h *ExportacionesHandler) Movimientos(c *gin.Context) {
	q := h.db.Model(&models.Movimiento{})
	if v := c.Query("user_id"); v != "" {
		q = q.Where("user_id = ?", v)
	}
	if v := c.Query("tipo"); v != "" {
		q = q.Where("tipo = ?", v)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("created_at BETWEEN ? AND ?", desde, hasta)
	}

	var movs []models.Movimiento
	if err := q.Order("created_at desc").Limit(20000).Find(&movs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo exportar"})
		return
	}

	filas := make([][]string, 0, len(movs))
	for _, m := range movs {
		filas = append(filas, []string{
			strconv.FormatUint(uint64(m.ID), 10),
			strconv.FormatUint(uint64(m.UserID), 10),
			m.Tipo,
			strconv.FormatInt(m.Monto, 10),
			m.Descripcion,
			formatoTiempoCR(m.CreatedAt),
		})
	}
	escribirCSV(c, "movimientos.csv", []string{
		"id", "usuario_id", "tipo", "monto", "descripcion", "fecha",
	}, filas)
}

// Usuarios exporta el padron de jugadores (sin claves ni hashes).
func (h *ExportacionesHandler) Usuarios(c *gin.Context) {
	q := h.db.Model(&models.User{})
	if v := c.Query("rol"); v != "" {
		q = q.Where("rol = ?", v)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("created_at BETWEEN ? AND ?", desde, hasta)
	}

	var users []models.User
	if err := q.Order("created_at desc").Limit(20000).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo exportar"})
		return
	}

	filas := make([][]string, 0, len(users))
	for _, u := range users {
		documento := u.TipoDocumento + " " + u.NumeroDocumento
		verificado := "no"
		if u.DocumentoVerificado {
			verificado = "si"
		}
		fechaNac := ""
		if u.FechaNacimiento != nil {
			fechaNac = u.FechaNacimiento.In(tiempo.CR).Format("2006-01-02")
		}
		autoexclusion := ""
		if u.AutoexcluidoHasta != nil {
			autoexclusion = u.AutoexcluidoHasta.In(tiempo.CR).Format("2006-01-02 15:04:05")
		}
		filas = append(filas, []string{
			strconv.FormatUint(uint64(u.ID), 10),
			u.Nombre,
			u.Email,
			u.Rol,
			u.Telefono,
			strconv.FormatInt(u.Saldo, 10),
			strconv.FormatInt(u.Bono, 10),
			documento,
			verificado,
			fechaNac,
			autoexclusion,
			formatoTiempoCR(u.CreatedAt),
		})
	}
	escribirCSV(c, "usuarios.csv", []string{
		"id", "nombre", "email", "rol", "telefono", "saldo", "bono",
		"documento", "verificado", "fecha_nacimiento", "autoexclusion", "registrado",
	}, filas)
}

// Auditoria exporta el registro de acciones administrativas.
func (h *ExportacionesHandler) Auditoria(c *gin.Context) {
	q := h.db.Model(&models.AuditLog{})
	if actor := c.Query("actor"); actor != "" {
		q = q.Where("actor_id = ?", actor)
	}
	if entidad := c.Query("entidad"); entidad != "" {
		q = q.Where("entidad = ?", entidad)
	}
	if accion := c.Query("accion"); accion != "" {
		q = q.Where("accion = ?", accion)
	}
	if desde, hasta, ok := rangoFechas(c); ok {
		q = q.Where("created_at BETWEEN ? AND ?", desde, hasta)
	}

	var entradas []models.AuditLog
	if err := q.Order("created_at desc").Limit(20000).Find(&entradas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo exportar"})
		return
	}

	filas := make([][]string, 0, len(entradas))
	for _, e := range entradas {
		filas = append(filas, []string{
			strconv.FormatUint(uint64(e.ID), 10),
			formatoTiempoCR(e.CreatedAt),
			strconv.FormatUint(uint64(e.ActorID), 10),
			e.ActorName,
			e.Accion,
			e.Entidad,
			e.EntidadID,
			e.Detalle,
			e.IP,
		})
	}
	escribirCSV(c, "auditoria.csv", []string{
		"id", "fecha", "actor_id", "actor", "accion", "entidad", "entidad_id", "detalle", "ip",
	}, filas)
}
