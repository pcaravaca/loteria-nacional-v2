package impresion

import (
	"strings"
	"testing"
)

func TestAnchoPXCalculado(t *testing.T) {
	if got := AnchoPXCalculado(58); got != 463 {
		t.Fatalf("58 mm -> %d px, quería 463", got)
	}
	if got := AnchoPXCalculado(80); got != 639 {
		t.Fatalf("80 mm -> %d px, quería 639", got)
	}
	if got := AnchoPXCalculado(0); got != 463 {
		t.Fatalf("ancho por defecto -> %d px, quería 463", got)
	}
}

func TestNormalizarAncho(t *testing.T) {
	if NormalizarAncho(80) != 80 || NormalizarAncho(58) != 58 {
		t.Fatal("58 y 80 deben respetarse")
	}
	for _, v := range []int{0, -1, 72, 120} {
		if got := NormalizarAncho(v); got != 58 {
			t.Fatalf("ancho %d -> %d, quería 58", v, got)
		}
	}
}

func TestFormatearNumeros(t *testing.T) {
	got := FormatearNumeros([]int{7, 15, 0}, 2)
	want := []string{"07", "15", "00"}
	if len(got) != len(want) {
		t.Fatalf("largo %d, quería %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %q, quería %q", i, got[i], want[i])
		}
	}
}

func TestGenerarEscapaContenido(t *testing.T) {
	html, err := Generar(Datos{
		Numero:  "000042",
		Sorteo:  `<script>alert("x")</script>`,
		Loteria: "LN",
		Numeros: []string{"07"},
		Monto:   "₡2.000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, `<script>alert`) {
		t.Fatal("el título del sorteo no fue escapado")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatal("no aparece el contenido escapado")
	}
	if !strings.Contains(html, "width: 463px") {
		t.Fatal("falta el ancho por defecto de 58 mm")
	}
	if strings.Contains(html, "%!") {
		t.Fatal("el HTML quedó con artefactos de formato")
	}
}

func TestGenerarAncho80(t *testing.T) {
	html, err := Generar(Datos{Numero: "1", AnchoPX: AnchoPXCalculado(80)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "width: 639px") {
		t.Fatal("el tique de 80 mm debe medir 639 px a 203 dpi")
	}
}
