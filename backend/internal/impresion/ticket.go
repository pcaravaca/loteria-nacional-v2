// Package impresion genera el HTML de los comprobantes térmicos.
//
// El mismo HTML se usa en los dos caminos de impresión:
//   - PC con POS-80: el frontend lo abre en una ventana y llama a window.print().
//   - Sunmi V2: el puente AndroidPOS lo renderiza en un WebView y manda el bitmap
//     a la impresora térmica integrada (ver loteria-nacional-android).
//
// No depende de la base de datos ni del framework HTTP: es puro HTML escapado,
// así que se puede probar con tests unitarios sin levantar el servidor.
package impresion

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

const ticketTemplate = `<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
@page { margin: 0; size: auto; }
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  font-family: 'Courier New', Courier, monospace;
  font-size: 12px;
  width: {{.AnchoPX}}px;
  padding: 3mm 4mm;
  color: #000;
}
.centro { text-align: center; }
.header { text-align: center; border-bottom: 1px dashed #000; padding-bottom: 4px; margin-bottom: 6px; }
.header h1 { font-size: 16px; letter-spacing: 1px; }
.header p { font-size: 10px; }
.fila { display: flex; justify-content: space-between; font-size: 11px; padding: 2px 0; }
.fila b { font-weight: bold; }
.linea { border-top: 1px dashed #000; margin: 6px 0; padding-top: 5px; }
.numeros { display: flex; flex-wrap: wrap; gap: 3px; justify-content: center; margin: 8px 0; }
.numero { border: 1px solid #000; border-radius: 3px; padding: 2px 6px; font-size: 16px; font-weight: bold; }
.total { border-top: 1px solid #000; margin-top: 6px; padding-top: 4px; font-size: 14px; font-weight: bold; }
.footer { text-align: center; font-size: 9px; border-top: 1px dashed #000; margin-top: 8px; padding-top: 5px; }
</style></head>
<body>
<div class="header">
  <h1>LOTERÍA NACIONAL</h1>
  <p>Comprobante de apuesta</p>
</div>
<div class="fila"><span>Tique <b>#{{.Numero}}</b></span><span>{{.Fecha}}</span></div>
<div class="linea"></div>
<div class="fila"><span>Sorteo</span><b>{{.Sorteo}}</b></div>
<div class="fila"><span>Lotería</span><span>{{.Loteria}}</span></div>
<div class="fila"><span>Jugada</span><span>{{.Jugada}}</span></div>
<div class="numeros">{{range .Numeros}}<span class="numero">{{.}}</span>{{end}}</div>
<div class="fila"><span>Apostado</span><b>{{.Monto}}</b></div>
{{if .Premio}}<div class="fila"><span>Premio</span><b>{{.Premio}}</b></div>{{end}}
<div class="total"><span>Estado: {{.Estado}}</span><span>{{.Aciertos}} aciertos</span></div>
<div class="footer">
  <p>Conserve este comprobante para consultar el resultado.</p>
  <p>El premio se acredita a su saldo. No válido para cobro en efectivo.</p>
</div>
</body></html>`

// Datos del comprobante. Todos los campos de texto se escapan con html/template.
type Datos struct {
	Numero   string
	Fecha    string
	Sorteo   string
	Loteria  string
	Jugada   string
	Numeros  []string
	Monto    string
	Premio   string
	Estado   string
	Aciertos int
	AnchoPX  int
}

var plantilla = template.Must(template.New("tique").Parse(ticketTemplate))

// AnchoPXCalculado convierte milímetros de papel a píxeles de impresión a 203 dpi,
// el mismo criterio que POSMaster usa para sus tiques térmicos.
func AnchoPXCalculado(anchoMM int) int {
	if anchoMM <= 0 {
		anchoMM = 58
	}
	return int(float64(anchoMM) * 203 / 25.4)
}

// NormalizarAncho acepta sólo los formatos de papel que sabemos imprimir.
func NormalizarAncho(anchoMM int) int {
	if anchoMM == 80 {
		return 80
	}
	return 58
}

// FormatearNumeros rellena con ceros a la izquierda según la lotería (02, 07, 15).
func FormatearNumeros(numeros []int, longitud int) []string {
	if longitud <= 0 {
		longitud = 2
	}
	out := make([]string, 0, len(numeros))
	for _, n := range numeros {
		out = append(out, fmt.Sprintf("%0*d", longitud, n))
	}
	return out
}

// Generar devuelve el HTML completo del comprobante.
func Generar(d Datos) (string, error) {
	if d.AnchoPX <= 0 {
		d.AnchoPX = AnchoPXCalculado(58)
	}
	if d.Estado == "" {
		d.Estado = "pendiente"
	}
	var b strings.Builder
	if err := plantilla.Execute(&b, d); err != nil {
		return "", err
	}
	return b.String(), nil
}

// FechaLocal formatea en hora de Costa Rica; si la zona no está disponible,
// cae a UTC en vez de fallar al imprimir.
func FechaLocal(t time.Time) string {
	loc, err := time.LoadLocation("America/Costa_Rica")
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("02/01/2006 15:04")
}
