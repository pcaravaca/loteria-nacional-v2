package resultados

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// servirFixture monta un servidor que responde cualquier ruta con el HTML del
// fixture, igual que lo que devolvería nacionalloteria para ?del-dia=...
func servirFixture(t *testing.T, nombre string) *httptest.Server {
	t.Helper()
	b, err := os.ReadFile("testdata/" + nombre + ".html")
	if err != nil {
		t.Fatalf("fixture %s: %v", nombre, err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	}))
}

func proveedorFixture(client *http.Client) *NacionAlloteria {
	p := NuevoNacionAlloteria(client)
	p.baseURL = "" // se pisa en cada test
	return p
}

func TestParsearLoteriaNacional(t *testing.T) {
	srv := servirFixture(t, "ln")
	defer srv.Close()
	p := NuevoNacionAlloteria(srv.Client())
	p.baseURL = srv.URL + "/"

	rs, err := p.Obtener(context.Background(), JuegoLoteriaNacional, fechaCR(t, "2026-11-15"))
	if err != nil {
		t.Fatalf("Obtener: %v", err)
	}
	if len(rs) == 0 {
		t.Fatal("no se parseó ningún resultado")
	}
	r := rs[0]
	if r.SorteoNum != "4927" {
		t.Errorf("sorteo=%q, quería 4927", r.SorteoNum)
	}
	if len(r.Numeros) != 3 || r.Numeros[0] != 76 {
		t.Errorf("numeros=%v, quería [76 ...]", r.Numeros)
	}
	if len(r.Series) == 0 {
		t.Errorf("se esperaba la serie del primer premio")
	}
}

func TestParsearChances(t *testing.T) {
	srv := servirFixture(t, "ca")
	defer srv.Close()
	p := NuevoNacionAlloteria(srv.Client())
	p.baseURL = srv.URL + "/"

	rs, err := p.Obtener(context.Background(), JuegoChances, fechaCR(t, "2026-11-17"))
	if err != nil {
		t.Fatalf("Obtener: %v", err)
	}
	if len(rs) == 0 {
		t.Fatal("no se parseó ningún resultado")
	}
	if rs[0].SorteoNum != "7087" || rs[0].Numeros[0] != 61 {
		t.Errorf("resultado inesperado: %+v", rs[0])
	}
}

func TestParsearNuevosTiempos(t *testing.T) {
	srv := servirFixture(t, "nt")
	defer srv.Close()
	p := NuevoNacionAlloteria(srv.Client())
	p.baseURL = srv.URL + "/"

	rs, err := p.Obtener(context.Background(), JuegoNuevosTiempos, fechaCR(t, "2026-11-18"))
	if err != nil {
		t.Fatalf("Obtener: %v", err)
	}
	if len(rs) != 3 {
		t.Fatalf("turnos parseados=%d, quería 3", len(rs))
	}
	porTurno := map[string]int{}
	for _, r := range rs {
		if len(r.Numeros) != 1 {
			t.Errorf("turno %s sin numero: %+v", r.Turno, r)
		}
		porTurno[r.Turno] = r.Numeros[0]
	}
	if porTurno[TurnoMedioDia] != 60 || porTurnoTarde(porTurno) != 48 || porTurno[TurnoNoche] != 34 {
		t.Errorf("turnos=%v, quería medio_dia=60 tarde=48 noche=34", porTurno)
	}
}

func porTurnoTarde(m map[string]int) int { return m[TurnoTarde] }

func TestParsearTresMonazos(t *testing.T) {
	srv := servirFixture(t, "monazos")
	defer srv.Close()
	p := NuevoNacionAlloteria(srv.Client())
	p.baseURL = srv.URL + "/"

	rs, err := p.Obtener(context.Background(), JuegoTresMonazos, fechaCR(t, "2026-11-18"))
	if err != nil {
		t.Fatalf("Obtener: %v", err)
	}
	if len(rs) != 3 {
		t.Fatalf("turnos parseados=%d, quería 3", len(rs))
	}
	var medio *Resultado
	for i := range rs {
		if rs[i].Turno == TurnoMedioDia {
			medio = &rs[i]
		}
	}
	if medio == nil {
		t.Fatal("falta el turno medio dia")
	}
	if len(medio.Numeros) != 3 || medio.Numeros[0] != 4 || medio.Numeros[1] != 9 || medio.Numeros[2] != 4 {
		t.Errorf("monazos medio dia=%v, quería [4 9 4]", medio.Numeros)
	}
}

func TestParsearLottoYRevancha(t *testing.T) {
	srv := servirFixture(t, "lotto")
	defer srv.Close()
	p := NuevoNacionAlloteria(srv.Client())
	p.baseURL = srv.URL + "/"

	lotto, err := p.Obtener(context.Background(), JuegoLotto, fechaCR(t, "2026-11-18"))
	if err != nil || len(lotto) != 1 {
		t.Fatalf("lotto: n=%d err=%v", len(lotto), err)
	}
	if len(lotto[0].Numeros) != 5 {
		t.Errorf("lotto numeros=%v", lotto[0].Numeros)
	}
	revancha, err := p.Obtener(context.Background(), JuegoLottoRevancha, fechaCR(t, "2026-11-18"))
	if err != nil || len(revancha) != 1 {
		t.Fatalf("revancha: n=%d err=%v", len(revancha), err)
	}
	if revancha[0].Numeros[0] != 35 {
		t.Errorf("revancha numeros=%v, quería empezar en 35", revancha[0].Numeros)
	}
}

func TestFechaSinSorteoDevuelveVacio(t *testing.T) {
	srv := servirFixture(t, "ln")
	defer srv.Close()
	p := NuevoNacionAlloteria(srv.Client())
	p.baseURL = srv.URL + "/"

	// El fixture solo trae el 15/11; pedir otro día debe dar vacío, sin error.
	rs, err := p.Obtener(context.Background(), JuegoLoteriaNacional, fechaCR(t, "2026-11-08"))
	if err != nil {
		t.Fatalf("Obtener: %v", err)
	}
	if len(rs) != 0 {
		t.Errorf("resultados=%d, quería 0", len(rs))
	}
}

func fechaCR(t *testing.T, s string) time.Time {
	t.Helper()
	f, err := time.ParseInLocation("2006-01-02", s, time.FixedZone("CR", -6*3600))
	if err != nil {
		t.Fatalf("fecha %s: %v", s, err)
	}
	return f
}
