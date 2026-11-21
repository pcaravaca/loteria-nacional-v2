package resultados

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// fuenteFalsa devuelve lo que el test quiere, sin tocar la red.
type fuenteFalsa struct {
	juego      string
	nombre     string
	resultados map[string][]Resultado // clave: fecha YYYY-MM-DD
	err        error
}

func (f *fuenteFalsa) Nombre() string {
	if f.nombre != "" {
		return f.nombre
	}
	return "falsa"
}
func (f *fuenteFalsa) Soporta(j string) bool { return j == f.juego }
func (f *fuenteFalsa) Obtener(_ context.Context, juego string, fecha time.Time) ([]Resultado, error) {
	if f.err != nil {
		return nil, f.err
	}
	rs := f.resultados[fecha.Format("2006-01-02")]
	for i := range rs {
		rs[i].Fuente = f.Nombre()
	}
	return rs, nil
}

func abrirDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := db.AutoMigrate(&models.Loteria{}, &models.Sorteo{}, &models.Apuesta{}, &models.User{}, &models.Movimiento{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestImportadorAplicaYPaga(t *testing.T) {
	db := abrirDB(t)
	loteria := models.Loteria{Codigo: "LN", Nombre: "Lotería Nacional", LongitudNum: 2,
		MaxNumeros: 3, Multiplicador: 50, Activa: true, TablaPremios: `{"1":2}`}
	if err := db.Create(&loteria).Error; err != nil {
		t.Fatalf("loteria: %v", err)
	}
	zona := time.FixedZone("CR", -6*3600)
	fecha := time.Date(2026, 11, 15, 19, 0, 0, 0, zona)
	sorteo := models.Sorteo{LoteriaID: loteria.ID, Titulo: "Lotería Nacional · Sorteo",
		Fecha: fecha, Estado: models.SorteoAbierto}
	if err := db.Create(&sorteo).Error; err != nil {
		t.Fatalf("sorteo: %v", err)
	}
	user := models.User{Nombre: "Prueba", Email: "p@prueba.test", Rol: models.RolCliente, Saldo: 1000}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("user: %v", err)
	}
	apuesta := models.Apuesta{UserID: user.ID, SorteoID: sorteo.ID, Numeros: []int{76},
		Monto: 100, Estado: models.ApuestaPendiente}
	if err := db.Create(&apuesta).Error; err != nil {
		t.Fatalf("apuesta: %v", err)
	}

	fuente := &fuenteFalsa{juego: "LN", resultados: map[string][]Resultado{
		"2026-11-15": {{Juego: "LN", SorteoNum: "4927", Numeros: []int{76, 46, 10}}},
	}}
	imp := NuevoImportador(db, []Proveedor{fuente}, nil)
	res := imp.ImportarLoteria(context.Background(), loteria)

	if res.Aplicados != 1 {
		t.Fatalf("aplicados=%d errores=%v", res.Aplicados, res.Errores)
	}
	var actualizado models.Sorteo
	if err := db.Preload("Loteria").First(&actualizado, sorteo.ID).Error; err != nil {
		t.Fatalf("sorteo: %v", err)
	}
	if actualizado.Estado != models.SorteoPagado {
		t.Errorf("estado=%s, quería pagado", actualizado.Estado)
	}
	if len(actualizado.NumerosGanadores) != 3 || actualizado.NumerosGanadores[0] != 76 {
		t.Errorf("ganadores=%v", actualizado.NumerosGanadores)
	}
	var ap models.Apuesta
	if err := db.First(&ap, apuesta.ID).Error; err != nil {
		t.Fatalf("apuesta: %v", err)
	}
	if ap.Estado != models.ApuestaPagada || ap.Aciertos != 1 || ap.Premio != 5000 {
		t.Errorf("apuesta=%+v, quería pagada aciertos=1 premio=5000", ap)
	}
	var u models.User
	if err := db.First(&u, user.ID).Error; err != nil {
		t.Fatalf("user: %v", err)
	}
	if u.Saldo != 6000 {
		t.Errorf("saldo=%d, quería 6000", u.Saldo)
	}
}

func TestImportadorRespetaElTurno(t *testing.T) {
	db := abrirDB(t)
	loteria := models.Loteria{Codigo: "NT", Nombre: "Nuevos Tiempos", LongitudNum: 2,
		MaxNumeros: 1, Multiplicador: 70, Activa: true}
	if err := db.Create(&loteria).Error; err != nil {
		t.Fatalf("loteria: %v", err)
	}
	zona := time.FixedZone("CR", -6*3600)
	fecha := time.Date(2026, 11, 18, 17, 0, 0, 0, zona)
	sorteo := models.Sorteo{LoteriaID: loteria.ID, Titulo: "Nuevos Tiempos · Sorteo de la tarde",
		Fecha: fecha, Estado: models.SorteoAbierto}
	if err := db.Create(&sorteo).Error; err != nil {
		t.Fatalf("sorteo: %v", err)
	}
	fuente := &fuenteFalsa{juego: "NT", resultados: map[string][]Resultado{
		"2026-11-18": {{Juego: "NT", Turno: TurnoTarde, Numeros: []int{48}}},
	}}
	imp := NuevoImportador(db, []Proveedor{fuente}, nil)
	res := imp.ImportarLoteria(context.Background(), loteria)
	if res.Aplicados != 1 {
		t.Fatalf("aplicados=%d omitidos=%d errores=%v", res.Aplicados, res.Omitidos, res.Errores)
	}
}

func TestMismoTurno(t *testing.T) {
	casos := []struct {
		titulo, turno string
		want          bool
	}{
		{"Nuevos Tiempos · Sorteo de la tarde", TurnoTarde, true},
		{"Nuevos Tiempos · Sorteo de el medio día", TurnoMedioDia, true},
		{"Nuevos Tiempos · Sorteo de la noche", TurnoNoche, true},
		{"Nuevos Tiempos · Sorteo de la tarde", TurnoNoche, false},
		{"Lotería Nacional · Sorteo", "", true},
		{"Lotería Nacional · Sorteo", TurnoTarde, false},
	}
	for _, c := range casos {
		if got := mismoTurno(c.titulo, c.turno); got != c.want {
			t.Errorf("mismoTurno(%q,%q)=%v, quería %v", c.titulo, c.turno, got, c.want)
		}
	}
}

// Si la fuente principal no publica una fecha, el importador prueba la
// siguiente antes de dar el sorteo por perdido.
func TestImportadorUsaElRespaldo(t *testing.T) {
	db := abrirDB(t)
	loteria := models.Loteria{Codigo: "NT1", Nombre: "Nuevos Tiempos", LongitudNum: 2,
		MaxNumeros: 1, Multiplicador: 70, Activa: true}
	db.Create(&loteria)
	zona := time.FixedZone("CR", -6*3600)
	fecha := time.Date(2026, 11, 20, 12, 55, 0, 0, zona)
	sorteo := models.Sorteo{LoteriaID: loteria.ID, Titulo: "Nuevos Tiempos · Sorteo de el medio día",
		Fecha: fecha, Estado: models.SorteoAbierto}
	db.Create(&sorteo)

	primaria := &fuenteFalsa{juego: "NT1", nombre: "principal", resultados: map[string][]Resultado{}}
	respaldo := &fuenteFalsa{juego: "NT1", nombre: "respaldo", resultados: map[string][]Resultado{
		"2026-11-20": {{Juego: "NT1", Turno: TurnoMedioDia, Numeros: []int{7}}},
	}}
	imp := NuevoImportador(db, []Proveedor{primaria, respaldo}, nil)
	res := imp.ImportarLoteria(context.Background(), loteria)
	if res.Aplicados != 1 {
		t.Fatalf("aplicados=%d errores=%v", res.Aplicados, res.Errores)
	}
	if res.Fuente != "respaldo" {
		t.Errorf("fuente=%s, quería respaldo", res.Fuente)
	}
	var s models.Sorteo
	db.First(&s, sorteo.ID)
	if s.Estado != models.SorteoPagado || len(s.NumerosGanadores) != 1 || s.NumerosGanadores[0] != 7 {
		t.Errorf("sorteo=%+v", s)
	}
}

// Si la fuente principal falla (red caida), tambien se prueba el respaldo.
func TestImportadorRespaldoCuandoElPrimarioFalla(t *testing.T) {
	db := abrirDB(t)
	loteria := models.Loteria{Codigo: "NT2", Nombre: "Nuevos Tiempos", LongitudNum: 2,
		MaxNumeros: 1, Multiplicador: 70, Activa: true}
	db.Create(&loteria)
	zona := time.FixedZone("CR", -6*3600)
	fecha := time.Date(2026, 11, 20, 19, 30, 0, 0, zona)
	sorteo := models.Sorteo{LoteriaID: loteria.ID, Titulo: "Nuevos Tiempos · Sorteo de la noche",
		Fecha: fecha, Estado: models.SorteoAbierto}
	db.Create(&sorteo)

	primaria := &fuenteFalsa{juego: "NT2", nombre: "principal", err: errors.New("404")}
	respaldo := &fuenteFalsa{juego: "NT2", nombre: "respaldo", resultados: map[string][]Resultado{
		"2026-11-20": {{Juego: "NT2", Turno: TurnoNoche, Numeros: []int{33}}},
	}}
	imp := NuevoImportador(db, []Proveedor{primaria, respaldo}, nil)
	res := imp.ImportarLoteria(context.Background(), loteria)
	if res.Aplicados != 1 {
		t.Fatalf("aplicados=%d errores=%v", res.Aplicados, res.Errores)
	}
	if res.Fuente != "respaldo" {
		t.Errorf("fuente=%s, quería respaldo", res.Fuente)
	}
}


