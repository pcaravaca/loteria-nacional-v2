package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/models"
	"loteria-backend/internal/resultados"
)

// IniciarImportador corre el importador de resultados cada intervalo mientras
// la API viva, y una primera pasada al arrancar. Antes de cada pasada
// regenera el calendario (incluyendo los días hacia atrás) para que los
// sorteos que hayan quedado sin crear puedan recibir su resultado. Con
// auto=false no hace nada (el admin puede forzarlo a mano).
func IniciarImportador(imp *resultados.Importador, auto bool, intervalo time.Duration, diasAtras int) {
	if !auto || imp == nil || intervalo <= 0 {
		return
	}
	correr := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		resumen := imp.ImportarPendientesConCalendario(ctx, diasAtras)
		cancel()
		for _, r := range resumen {
			if r.Aplicados > 0 || len(r.Errores) > 0 {
				log.Printf("importacion %s: aplicados=%d omitidos=%d errores=%v",
					r.Loteria, r.Aplicados, r.Omitidos, r.Errores)
			}
		}
	}
	go func() {
		correr()
		ticker := time.NewTicker(intervalo)
		defer ticker.Stop()
		for range ticker.C {
			correr()
		}
	}()
}

// ImportarResultadosHandler fuerza la importacion de resultados ahora.
func (h *AdminHandler) ImportarResultados(c *gin.Context) {
	imp := resultados.NuevoImportador(h.db, []resultados.Proveedor{resultados.NuevoNacionAlloteria(nil)}, nil)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()
	resumen := imp.ImportarPendientes(ctx)
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "resultados.importar", "sorteo", "-",
		"importacion manual de resultados")
	c.JSON(http.StatusOK, gin.H{"resultados": resumen})
}

// GenerarCalendarioHandler crea los sorteos futuros que falten.
func (h *AdminHandler) GenerarCalendario(c *gin.Context) {
	n, err := resultados.GenerarSorteosVentana(h.db, nil, 14, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar el calendario"})
		return
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "sorteo.calendario", "sorteo", "-",
		"sorteos generados: "+strconv.Itoa(n))
	c.JSON(http.StatusOK, gin.H{"generados": n})
}

// EstadoResultadosHandler resume el estado de los sorteos por loteria.
func (h *AdminHandler) EstadoResultados(c *gin.Context) {
	var loterias []models.Loteria
	if err := h.db.Order("codigo asc").Find(&loterias).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo leer el estado"})
		return
	}
	type fila struct {
		Loteria  string `json:"loteria"`
		Nombre   string `json:"nombre"`
		Abiertos int64  `json:"abiertos"`
		Pagados  int64  `json:"pagados"`
		Ultimo   string `json:"ultimo_sorteo"`
	}
	out := make([]fila, 0, len(loterias))
	zona := time.FixedZone("CR", -6*3600)
	for _, l := range loterias {
		var abiertos, pagados int64
		h.db.Model(&models.Sorteo{}).Where("loteria_id = ? AND estado = ?", l.ID, models.SorteoAbierto).Count(&abiertos)
		h.db.Model(&models.Sorteo{}).Where("loteria_id = ? AND estado = ?", l.ID, models.SorteoPagado).Count(&pagados)
		var ultimo models.Sorteo
		if err := h.db.Where("loteria_id = ? AND estado = ?", l.ID, models.SorteoPagado).
			Order("fecha desc").First(&ultimo).Error; err == nil {
			out = append(out, fila{
				Loteria: l.Codigo, Nombre: l.Nombre, Abiertos: abiertos, Pagados: pagados,
				Ultimo: ultimo.Titulo + " · " + ultimo.Fecha.In(zona).Format("2006-01-02 15:04"),
			})
			continue
		}
		out = append(out, fila{Loteria: l.Codigo, Nombre: l.Nombre, Abiertos: abiertos, Pagados: pagados})
	}
	c.JSON(http.StatusOK, gin.H{"loterias": out})
}
