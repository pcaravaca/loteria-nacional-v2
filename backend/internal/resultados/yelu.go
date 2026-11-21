package resultados

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const yeluBase = "https://www.yelu.cr/lottery/results/"

// Yelu (yelu.cr) es un espejo en texto de la JPS. Se usa como respaldo cuando
// la fuente principal (nacionalloteria.com) no responde. Cubre Nuevos Tiempos
// y 3 Monazos, los dos con tres turnos por dia.
type Yelu struct {
	client  *http.Client
	baseURL string
	zona    *time.Location
}

func NuevoYelu(client *http.Client) *Yelu {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	zona, err := time.LoadLocation(zonaCR)
	if err != nil {
		zona = time.FixedZone("CR", -6*3600)
	}
	return &Yelu{client: client, baseURL: yeluBase, zona: zona}
}

func (p *Yelu) Nombre() string { return "yelu" }

var yeluSlugs = map[string]string{
	JuegoNuevosTiempos: "nuevos-tiempos",
	JuegoTresMonazos:   "tres-monazos",
}

func (p *Yelu) Soporta(juego string) bool {
	_, ok := yeluSlugs[juego]
	return ok
}

var (
	reYeluFila   = regexp.MustCompile(`(?s)<tr><td title="Fecha del Sorteo">(.*?)</tr>`)
	reYeluFecha  = regexp.MustCompile(`(\d{1,2}) de ([A-Za-záéíóúñ]+) (\d{4})`)
	reYeluNum    = regexp.MustCompile(`class="lotto_no_r[^"]*">([^<]*)<`)
	reYeluSup    = regexp.MustCompile(`(?s)<sup>(.*?)</sup>`)
	reYeluSorteo = regexp.MustCompile(`#(\d+)`)
)

var mesesYelu = map[string]time.Month{
	"enero": time.January, "febrero": time.February, "marzo": time.March,
	"abril": time.April, "mayo": time.May, "junio": time.June,
	"julio": time.July, "agosto": time.August, "septiembre": time.September,
	"setiembre": time.September, "octubre": time.October,
	"noviembre": time.November, "diciembre": time.December,
}

// turnoYelu: la fuente etiqueta los turnos de NT/Monazos como DIA (el de las
// 12:55), MIDDAY (16:30) y NOCHE (19:30). Verificado contra sorteos ya
// importados: 20-nov DIA=07/MIDDAY=31/NOCHE=33 y 19-nov 26/68/85, que son
// exactamente nuestros turnos medio dia/tarde/noche.
func turnoYelu(sup string) string {
	switch strings.ToUpper(strings.TrimSpace(sup)) {
	case "DÍA", "DIA", "MEDIODÍA", "MEDIODIA":
		return TurnoMedioDia
	case "MIDDAY", "TARDE":
		return TurnoTarde
	case "NOCHE":
		return TurnoNoche
	}
	return ""
}

func (p *Yelu) Obtener(ctx context.Context, juego string, fecha time.Time) ([]Resultado, error) {
	slug, ok := yeluSlugs[juego]
	if !ok {
		return nil, fmt.Errorf("juego no soportado: %s", juego)
	}
	url := p.baseURL + slug
	doc, err := p.fetch(ctx, url)
	if err != nil {
		return nil, err
	}
	return extraerYelu(juego, doc, fecha, url), nil
}

// extraerYelu parsea la tabla de "Números Ganadores" de yelu.cr. Solo se queda
// con las filas de la fecha pedida y con turnos que reconoce.
func extraerYelu(juego, doc string, fecha time.Time, url string) []Resultado {
	normal := fecha.Format("2006-01-02")
	out := make([]Resultado, 0, 3)
	for _, m := range reYeluFila.FindAllStringSubmatch(doc, -1) {
		fila := m[1]
		f, ok := fechaYelu(fila)
		if !ok || f.Format("2006-01-02") != normal {
			continue
		}
		turno := turnoYelu(textoYelu(fila, reYeluSup))
		if turno == "" {
			// Turno que no reconozco: no lo uso (mejor no graduar que graduar mal).
			continue
		}
		numeros := numerosYelu(fila)
		if len(numeros) == 0 {
			continue
		}
		sorteo := ""
		if sm := reYeluSorteo.FindStringSubmatch(fila); sm != nil {
			sorteo = sm[1]
		}
		out = append(out, Resultado{
			Juego: juego, SorteoNum: sorteo, Fecha: fecha, Turno: turno,
			Numeros: numeros, Fuente: "yelu", URL: url,
		})
	}
	return out
}

func fechaYelu(h string) (time.Time, bool) {
	m := reYeluFecha.FindStringSubmatch(h)
	if m == nil {
		return time.Time{}, false
	}
	mes, ok := mesesYelu[strings.ToLower(strings.TrimSpace(m[2]))]
	if !ok {
		return time.Time{}, false
	}
	d, err1 := strconv.Atoi(m[1])
	y, err2 := strconv.Atoi(m[3])
	if err1 != nil || err2 != nil {
		return time.Time{}, false
	}
	return time.Date(y, mes, d, 0, 0, 0, 0, time.UTC), true
}

func textoYelu(h string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(h)
	if m == nil {
		return ""
	}
	return limpiarTexto(m[1])
}

func numerosYelu(h string) []int {
	out := make([]int, 0, 3)
	for _, m := range reYeluNum.FindAllStringSubmatch(h, -1) {
		v := strings.TrimSpace(m[1])
		if v == "" || strings.ContainsAny(v, "ABCDEFGHIJKLMNÑOPQRSTUVWXYZáéíóú:") {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

func (p *Yelu) fetch(ctx context.Context, url string) (string, error) {
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
