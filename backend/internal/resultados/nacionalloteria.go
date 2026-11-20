package resultados

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	nacionBase = "https://www.nacionalloteria.com/costarica/"
	zonaCR     = "America/Costa_Rica"
)

// NacionAlloteria es el espejo en texto de los resultados de la JPS. No es la
// fuente oficial (jps.go.cr publica PDF escaneados), pero es la unica con
// historial legible por máquina.
type NacionAlloteria struct {
	client  *http.Client
	baseURL string
	zona    *time.Location
}

func NuevoNacionAlloteria(client *http.Client) *NacionAlloteria {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	zona, err := time.LoadLocation(zonaCR)
	if err != nil {
		zona = time.FixedZone("CR", -6*3600)
	}
	return &NacionAlloteria{client: client, baseURL: nacionBase, zona: zona}
}

func (p *NacionAlloteria) Nombre() string { return "nacionalloteria" }

var paginas = map[string]string{
	JuegoLoteriaNacional: "loteria-nacional.php",
	JuegoChances:         "chances.php",
	JuegoNuevosTiempos:   "nuevos-tiempos.php",
	JuegoLotto:           "lotto.php",
	JuegoLottoRevancha:   "lotto.php",
	JuegoTresMonazos:     "3-monazos.php",
}

func (p *NacionAlloteria) Soporta(juego string) bool {
	_, ok := paginas[juego]
	return ok
}

func (p *NacionAlloteria) Obtener(ctx context.Context, juego string, fecha time.Time) ([]Resultado, error) {
	pagina, ok := paginas[juego]
	if !ok {
		return nil, fmt.Errorf("juego no soportado: %s", juego)
	}
	url := p.baseURL + pagina + "?del-dia=" + fecha.Format("2006-01-02")
	doc, err := p.fetch(ctx, url)
	if err != nil {
		return nil, err
	}
	// La fuente solo publica el sorteo si la fecha coincide con un sorteo; si
	// no hay datos para ese día devuelve vacío sin error.
	bloques := separarBloques(doc)
	if len(bloques) == 0 {
		return nil, nil
	}
	b := bloques[0]
	f, ok := fechaDelBloque(b.html)
	if !ok || f.Format("2006-01-02") != fecha.Format("2006-01-02") {
		// La fuente no publico esa fecha.
		return nil, nil
	}
	return extraer(juego, b, doc, f.In(p.zona), url), nil
}

type bloque struct {
	html  string
	titulo string
}

// separarBloques parte el listado en un bloque por sorteo (un <h2> cada uno).
// Se hace a mano porque Go no soporta lookahead en sus expresiones regulares.
func separarBloques(doc string) []bloque {
	var out []bloque
	rest := doc
	for {
		i := strings.Index(rest, "<h2>")
		if i < 0 {
			break
		}
		rest = rest[i:]
		end := len(rest)
		if k := strings.Index(rest, "</section>"); k >= 0 && k < end {
			end = k
		}
		if j := strings.Index(rest[4:], "<h2>"); j >= 0 && 4+j < end {
			end = 4 + j
		}
		b := rest[:end]
		titulo := ""
		if h := strings.Index(b, "</h2>"); h > 0 {
			titulo = limpiarTexto(b[:h])
		}
		out = append(out, bloque{html: b, titulo: titulo})
		rest = rest[end:]
	}
	return out
}

var reFecha = regexp.MustCompile(`datetime="(\d{4}-\d{2}-\d{2})"`)

func fechaDelBloque(h string) (time.Time, bool) {
	m := reFecha.FindStringSubmatch(h)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", m[1], time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func (p *NacionAlloteria) fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LoteriaCR/1.0)")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("leer %s: %w", url, err)
	}
	return string(b), nil
}

func extraer(juego string, b bloque, doc string, fecha time.Time, url string) []Resultado {
	switch juego {
	case JuegoLoteriaNacional:
		num := primerNumero(b.titulo, "Loteria Nacional")
		ns, series := premiosDetalle(doc)
		if len(ns) == 0 {
			ns = numerosPorPremio(b.html)
			series = seriesPorPremio(b.html)
		}
		if num == "" || len(ns) == 0 {
			return nil
		}
		return []Resultado{{
			Juego: JuegoLoteriaNacional, SorteoNum: num, Fecha: fecha,
			Numeros: ns, Series: series,
			Fuente: "nacionalloteria", URL: url,
		}}
	case JuegoChances:
		num := primerNumero(b.titulo, "Chances")
		ns, series := premiosDetalle(doc)
		if len(ns) == 0 {
			ns = numerosPorPremio(b.html)
			series = seriesPorPremio(b.html)
		}
		if num == "" || len(ns) == 0 {
			return nil
		}
		return []Resultado{{
			Juego: JuegoChances, SorteoNum: num, Fecha: fecha,
			Numeros: ns, Series: series,
			Fuente: "nacionalloteria", URL: url,
		}}
	case JuegoNuevosTiempos:
		num := primerNumero(b.titulo, "Nuevos Tiempos")
		rs := make([]Resultado, 0, 3)
		for _, turno := range []struct{ etiqueta, codigo string }{
			{"Medio d", TurnoMedioDia}, {"Tarde", TurnoTarde}, {"Noche", TurnoNoche},
		} {
			if n, ok := numeroDeTurnoNT(b.html, turno.etiqueta); ok {
				rs = append(rs, Resultado{
					Juego: JuegoNuevosTiempos, SorteoNum: num, Fecha: fecha, Turno: turno.codigo,
					Numeros: []int{n}, Fuente: "nacionalloteria", URL: url,
				})
			}
		}
		return rs
	case JuegoTresMonazos:
		num := primerNumero(b.titulo, "3 Monazos")
		rs := make([]Resultado, 0, 3)
		for _, turno := range []struct{ etiqueta, codigo string }{
			{"Medio D", TurnoMedioDia}, {"Tarde", TurnoTarde}, {"Noche", TurnoNoche},
		} {
			if ns, ok := numerosDeTurnoMonazos(b.html, turno.etiqueta); ok {
				rs = append(rs, Resultado{
					Juego: JuegoTresMonazos, SorteoNum: num, Fecha: fecha, Turno: turno.codigo,
					Numeros: ns, Fuente: "nacionalloteria", URL: url,
				})
			}
		}
		return rs
	case JuegoLotto, JuegoLottoRevancha:
		num := primerNumero(b.titulo, "Lotto JPS")
		titulo := "Premios del Lotto JPS"
		codigo := JuegoLotto
		if juego == JuegoLottoRevancha {
			titulo = "Premios del Lotto Revancha"
			codigo = JuegoLottoRevancha
		}
		ns, ok := numerosSeccion(doc, titulo)
		if !ok {
			return nil
		}
		return []Resultado{{
			Juego: codigo, SorteoNum: num, Fecha: fecha, Numeros: ns,
			Fuente: "nacionalloteria", URL: url,
		}}
	}
	return nil
}

var reEtiquetaNumero = regexp.MustCompile(`(?is)<span class="label label-normal[^"]*">\s*([^<]+?)\s*</span>\s*(?:</td>\s*<td[^>]*>\s*)?<span class="label label-numero">\s*([0-9][0-9 ,]*)\s*</span>`)

func primerNumero(titulo, prefijo string) string {
	t := limpiarTexto(titulo)
	i := strings.Index(strings.ToLower(t), strings.ToLower(prefijo))
	if i < 0 {
		return ""
	}
	rest := strings.TrimSpace(t[i+len(prefijo):])
	// "4927 - Domingo..." / "24185|24187|24186 de ..."
	f := strings.FieldsFunc(rest, func(r rune) bool {
		return !(r >= '0' && r <= '9') && r != '|'
	})
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

var rePremio = regexp.MustCompile(`(?is)(Primer|Segundo|Tercer)\s*Premio</span>\s*<span class="label label-numero">\s*([0-9]+)\s*</span>`)

func numerosPorPremio(h string) []int {
	ms := rePremio.FindAllStringSubmatch(h, -1)
	out := make([]int, 0, len(ms))
	for _, m := range ms {
		n, err := strconv.Atoi(strings.TrimSpace(m[2]))
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

var reSerie = regexp.MustCompile(`(?is)Serie</span>\s*<span class="label label-numero">\s*([0-9]+)`)

func seriesPorPremio(h string) []string {
	ms := reSerie.FindAllStringSubmatch(h, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m[1])
	}
	return out
}

// numeroDeTurnoNT lee la fila de la tabla de Nuevos Tiempos (medio día/tarde/noche).
func numeroDeTurnoNT(h, etiqueta string) (int, bool) {
	re := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(etiqueta) + `[^<]*</span></td><td[^>]*>\s*<span class="label label-numero">\s*([0-9]+)\s*</span>`)
	m := re.FindStringSubmatch(h)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

// numerosDeTurnoMonazos lee las tres cifras de un turno de 3 Monazos ("4 9 4").
func numerosDeTurnoMonazos(h, etiqueta string) ([]int, bool) {
	re := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(etiqueta) + `[^<]*</span>\s*<span class="label label-numero">\s*([0-9 ]+?)\s*</span>`)
	m := re.FindStringSubmatch(h)
	if m == nil {
		return nil, false
	}
	parts := strings.Fields(m[1])
	if len(parts) == 0 {
		return nil, false
	}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

func numerosSeccion(h, titulo string) ([]int, bool) {
	re := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(titulo) + `.*?</h3>\s*<div[^>]*>(.*?)</div>`)
	m := re.FindStringSubmatch(h)
	if m == nil {
		return nil, false
	}
	reNum := regexp.MustCompile(`(?is)<span class="label label-numero">\s*([0-9]+)\s*</span>`)
	ms := reNum.FindAllStringSubmatch(m[1], -1)
	if len(ms) == 0 {
		return nil, false
	}
	out := make([]int, 0, len(ms))
	for _, x := range ms {
		n, err := strconv.Atoi(x[1])
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

// rePremioDetalle lee la tabla "Resultado del sorteo": cada fila trae el
// nombre del premio, el numero y la serie. Ahi estan los tres premios grandes.
var rePremioDetalle = regexp.MustCompile(`(?is)(?:Primer|Segundo|Tercer)\s*Premio[^<]*</td>\s*<td[^>]*>\s*(\d+)\s*</td>\s*<td[^>]*>\s*([0-9][0-9,\s]*)`)

func premiosDetalle(doc string) ([]int, []string) {
	ms := rePremioDetalle.FindAllStringSubmatch(doc, -1)
	if len(ms) == 0 {
		return nil, nil
	}
	numeros := make([]int, 0, len(ms))
	series := make([]string, 0, len(ms))
	for _, m := range ms {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		s := strings.TrimSpace(strings.TrimRight(m[2], ", -"))
		numeros = append(numeros, n)
		series = append(series, s)
	}
	if len(numeros) == 0 {
		return nil, nil
	}
	return numeros, series
}

func limpiarTexto(s string) string {
	s = html.UnescapeString(s)
	reTags := regexp.MustCompile(`<[^>]+>`)
	s = reTags.ReplaceAllString(s, " ")
	reEspacios := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(reEspacios.ReplaceAllString(s, " "))
}
