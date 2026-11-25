package impresion

import (
	"strings"
	"testing"
)

func TestGenerarReporteTermico(t *testing.T) {
	d := ReporteDatos{
		Periodo:       "2026-11-01 — 2026-11-25",
		Filas:         []FilaReporte{{Nombre: "Lotería Nacional", Apuestas: "3", Apostado: "₡4.000", Premios: "₡2.000"}},
		TotalApuestas: "3", TotalApostado: "₡4.000", TotalPremios: "₡2.000",
		Generado: "25/11/2026 06:30", AnchoMM: 58,
	}
	html, err := GenerarReporte(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, esperado := range []string{"width: 463px", "Lotería Nacional", "₡4.000", "2026-11-01", "58 mm"} {
		if !strings.Contains(html, esperado) {
			t.Fatalf("falta %q en el reporte", esperado)
		}
	}
	d.AnchoMM = 80
	html, err = GenerarReporte(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "width: 639px") || !strings.Contains(html, "80 mm") {
		t.Fatal("el reporte de 80 mm debe medir 639 px")
	}
}

func TestPeriodoReporte(t *testing.T) {
	if got := PeriodoReporte("", ""); got != "Histórico completo" {
		t.Fatalf("periodo vacío = %q", got)
	}
	if got := PeriodoReporte("2026-11-25", ""); got != "Desde 2026-11-25" {
		t.Fatalf("periodo desde = %q", got)
	}
	if got := PeriodoReporte("", "2026-11-25"); got != "Hasta 2026-11-25" {
		t.Fatalf("periodo hasta = %q", got)
	}
	if got := PeriodoReporte("2026-11-01", "2026-11-25"); got != "2026-11-01 — 2026-11-25" {
		t.Fatalf("periodo rango = %q", got)
	}
}
