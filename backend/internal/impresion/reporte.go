package impresion

import (
	"html/template"
	"strings"
	"time"
)

// Plantilla de reporte para papel térmico: una sola columna de datos, sin
// colores ni imágenes, con filas compactas y separadores punteados. La misma
// se usa en la Sunmi V2 (58 mm) y en la POS-80 (80 mm) cambiando sólo el ancho.
const reporteTemplate = `<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
@page { margin: 0; size: auto; }
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: 'Courier New', Courier, monospace; font-size: 10px; width: {{.AnchoPX}}px; padding: 3mm 4mm; color: #000; }
.header { text-align: center; border-bottom: 1px dashed #000; padding-bottom: 4px; margin-bottom: 5px; }
.header h1 { font-size: 14px; letter-spacing: 1px; }
.header p { font-size: 9px; }
.periodo { text-align: center; font-size: 10px; margin-bottom: 4px; }
.fila { display: flex; justify-content: space-between; gap: 4px; padding: 2px 0; border-bottom: 1px dotted #999; }
.fila span:first-child { flex: 1; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.fila span:not(:first-child) { text-align: right; white-space: nowrap; }
.cabecera { font-weight: bold; border-bottom: 1px solid #000; }
.total { display: flex; justify-content: space-between; gap: 4px; border-top: 1px solid #000; margin-top: 5px; padding-top: 4px; font-size: 12px; font-weight: bold; }
.total span:first-child { flex: 1; }
.total span:not(:first-child) { text-align: right; white-space: nowrap; flex: 0 0 auto; }
.footer { text-align: center; font-size: 8px; border-top: 1px dashed #000; margin-top: 6px; padding-top: 4px; }
</style></head>
<body>
<div class="header"><h1>LOTERÍA NACIONAL</h1><p>Reporte de apuestas</p></div>
<div class="periodo">{{.Periodo}}</div>
<div class="fila cabecera"><span>Lotería</span><span>Apuestas</span><span>Apostado</span><span>Premios</span></div>
{{range .Filas}}<div class="fila"><span>{{.Nombre}}</span><span>{{.Apuestas}}</span><span>{{.Apostado}}</span><span>{{.Premios}}</span></div>{{end}}
<div class="total"><span>TOTAL</span><span>{{.TotalApuestas}}</span><span>{{.TotalApostado}}</span><span>{{.TotalPremios}}</span></div>
<div class="footer"><p>Generado {{.Generado}}</p><p>Reporte térmico · {{.AnchoMM}} mm</p></div>
</body></html>`

var plantillaReporte = template.Must(template.New("reporte").Parse(reporteTemplate))

type FilaReporte struct {
	Nombre   string
	Apuestas string
	Apostado string
	Premios  string
}

type ReporteDatos struct {
	Periodo       string
	Filas         []FilaReporte
	TotalApuestas string
	TotalApostado string
	TotalPremios  string
	Generado      string
	AnchoMM       int
	AnchoPX       int
}

// GenerarReporte devuelve el HTML compacto del reporte para térmica.
func GenerarReporte(d ReporteDatos) (string, error) {
	if d.AnchoPX <= 0 {
		d.AnchoPX = AnchoPXCalculado(d.AnchoMM)
	}
	if d.AnchoMM != 80 {
		d.AnchoMM = 58
	}
	var b strings.Builder
	if err := plantillaReporte.Execute(&b, d); err != nil {
		return "", err
	}
	return b.String(), nil
}

// PeriodoReporte arma el rótulo del rango de fechas; con fechas vacías muestra
// "Histórico" para no dejar la línea en blanco en el papel.
func PeriodoReporte(desde, hasta string) string {
	if desde == "" && hasta == "" {
		return "Histórico completo"
	}
	if desde == "" {
		return "Hasta " + hasta
	}
	if hasta == "" {
		return "Desde " + desde
	}
	return desde + " — " + hasta
}

// HoraReporte usa la hora local de Costa Rica para el pie del comprobante.
func HoraReporte() string {
	return FechaLocal(time.Now())
}
