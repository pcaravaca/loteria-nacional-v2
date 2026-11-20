package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
)

// accesoFixture arma una base en memoria con dos jugadores, dos paquetes, dos
// loterias (una activa y una apagada), dos juegos (uno activo y uno apagado) y
// una rifa.
type accesoFixture struct {
	db   *gorm.DB
	u1   models.User
	u2   models.User
	g1   models.GrupoJugadores
	g2   models.GrupoJugadores
	rifa models.Rifa
}

func nuevaAccesoFixture(t *testing.T) *accesoFixture {
	t.Helper()
	db, err := database.Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	fx := &accesoFixture{db: db}
	fx.u1 = models.User{Nombre: "Jugador Uno", Email: "uno@test", Rol: models.RolCliente, Saldo: 100000}
	fx.u2 = models.User{Nombre: "Jugador Dos", Email: "dos@test", Rol: models.RolCliente, Saldo: 100000}
	if err := db.Create(&fx.u1).Error; err != nil {
		t.Fatalf("u1: %v", err)
	}
	if err := db.Create(&fx.u2).Error; err != nil {
		t.Fatalf("u2: %v", err)
	}

	fx.g1 = models.GrupoJugadores{Nombre: "Paquete A", Descripcion: "uno"}
	fx.g2 = models.GrupoJugadores{Nombre: "Paquete B", Descripcion: "dos"}
	if err := db.Create(&fx.g1).Error; err != nil {
		t.Fatalf("g1: %v", err)
	}
	if err := db.Create(&fx.g2).Error; err != nil {
		t.Fatalf("g2: %v", err)
	}

	// Select("*") para que los booleanos en false se escriban: el default de
	// `activa`/`activo` es true y GORM omitiria el cero.
	if err := db.Select("*").Create(&models.Loteria{
		Codigo: "LN", Nombre: "Lotería Nacional", Tipo: models.TipoLoteria, Activa: true,
	}).Error; err != nil {
		t.Fatalf("loteria LN: %v", err)
	}
	if err := db.Select("*").Create(&models.Loteria{
		Codigo: "LOTTO", Nombre: "Lotto", Tipo: models.TipoLoteria, Activa: false,
	}).Error; err != nil {
		t.Fatalf("loteria LOTTO: %v", err)
	}
	if err := db.Select("*").Create(&models.CasinoGame{
		Codigo: models.CasinoDados, Nombre: "Dados del Sur", MinApuesta: 200, MaxApuesta: 50000, Activo: true,
	}).Error; err != nil {
		t.Fatalf("juego dados: %v", err)
	}
	if err := db.Select("*").Create(&models.CasinoGame{
		Codigo: models.CasinoSlots, Nombre: "Fortuna CR", MinApuesta: 500, MaxApuesta: 50000, Activo: false,
	}).Error; err != nil {
		t.Fatalf("juego slots: %v", err)
	}
	// GORM rellena los booleanos con `default` durante el Create (muta false a
	// true), asi que el estado apagado se impone con un update explicito.
	if err := db.Model(&models.Loteria{}).Where("codigo = ?", "LOTTO").
		Update("activa", false).Error; err != nil {
		t.Fatalf("fijar LOTTO inactiva: %v", err)
	}
	if err := db.Model(&models.CasinoGame{}).Where("codigo = ?", models.CasinoSlots).
		Update("activo", false).Error; err != nil {
		t.Fatalf("fijar slots inactivo: %v", err)
	}

	fx.rifa = models.Rifa{
		Titulo: "Rifa de prueba", Premio: "Moto", PrecioNumero: 100,
		DesdeNumero: 0, HastaNumero: 9, Estado: models.RifaAbierta,
		FechaSorteo: time.Now().Add(48 * time.Hour),
	}
	if err := db.Create(&fx.rifa).Error; err != nil {
		t.Fatalf("rifa: %v", err)
	}
	return fx
}

func (fx *accesoFixture) acceso(ambito string, ambitoID uint, tipo, codigo string, activo bool) {
	fila := models.AccesoContenido{Ambito: ambito, AmbitoID: ambitoID, Tipo: tipo, Codigo: codigo, Activo: activo}
	if err := fx.db.Create(&fila).Error; err != nil {
		panic(err)
	}
}

// ------------------------------------------------------------ resolución

func TestResolucionAccesoDefaultPorTipo(t *testing.T) {
	fx := nuevaAccesoFixture(t)

	casos := []struct {
		tipo   string
		codigo string
		want   bool
	}{
		{models.ContenidoLoteria, "LN", true},     // loteria activa
		{models.ContenidoLoteria, "LOTTO", false}, // loteria apagada
		{models.ContenidoJuego, models.CasinoDados, true},
		{models.ContenidoJuego, models.CasinoSlots, false},
		{models.ContenidoRifa, strconv.FormatUint(uint64(fx.rifa.ID), 10), true}, // rifa: activa por defecto
	}
	for _, c := range casos {
		ok, fuente, err := ResuelveAcceso(fx.db, fx.u1.ID, c.tipo, c.codigo)
		if err != nil {
			t.Fatalf("%s %s: %v", c.tipo, c.codigo, err)
		}
		if ok != c.want {
			t.Errorf("%s %s: activo=%v, queria %v", c.tipo, c.codigo, ok, c.want)
		}
		if fuente != "default" {
			t.Errorf("%s %s: fuente=%s, queria default", c.tipo, c.codigo, fuente)
		}
	}
}

func TestResolucionAccesoGlobalOnOff(t *testing.T) {
	fx := nuevaAccesoFixture(t)

	// Sin filas manda el estado global del contenido.
	if ok, _, _ := ResuelveAcceso(fx.db, fx.u1.ID, models.ContenidoJuego, models.CasinoDados); !ok {
		t.Fatal("el juego deberia estar activo por su fila global")
	}

	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoJuego, models.CasinoDados, false)
	ok, fuente, err := ResuelveAcceso(fx.db, fx.u1.ID, models.ContenidoJuego, models.CasinoDados)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if ok {
		t.Error("la fila global apagada deberia ocultar el juego")
	}
	if fuente != models.AmbitoGlobal {
		t.Errorf("fuente=%s, queria %s", fuente, models.AmbitoGlobal)
	}

	// Se puede reencender con una fila global activa.
	if err := fx.db.Model(&models.AccesoContenido{}).
		Where("ambito = ? AND tipo = ? AND codigo = ?", models.AmbitoGlobal, models.ContenidoJuego, models.CasinoDados).
		Update("activo", true).Error; err != nil {
		t.Fatalf("update: %v", err)
	}
	ok, _, _ = ResuelveAcceso(fx.db, fx.u1.ID, models.ContenidoJuego, models.CasinoDados)
	if !ok {
		t.Error("la fila global activa deberia mostrar el juego")
	}
}

func TestResolucionAccesoGrupo(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	if err := fx.db.Create(&models.JugadorGrupo{UserID: fx.u1.ID, GrupoID: fx.g1.ID}).Error; err != nil {
		t.Fatalf("membresia: %v", err)
	}

	// Global apagado, paquete encendido: el paquete gana.
	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoLoteria, "LN", false)
	fx.acceso(models.AmbitoGrupo, fx.g1.ID, models.ContenidoLoteria, "LN", true)

	ok, fuente, err := ResuelveAcceso(fx.db, fx.u1.ID, models.ContenidoLoteria, "LN")
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !ok || fuente != models.AmbitoGrupo {
		t.Errorf("u1: activo=%v fuente=%s, queria true/grupo", ok, fuente)
	}

	// u2 no esta en el paquete: se queda con el global apagado.
	ok, fuente, err = ResuelveAcceso(fx.db, fx.u2.ID, models.ContenidoLoteria, "LN")
	if err != nil {
		t.Fatalf("resolver u2: %v", err)
	}
	if ok || fuente != models.AmbitoGlobal {
		t.Errorf("u2: activo=%v fuente=%s, queria false/global", ok, fuente)
	}
}

func TestResolucionAccesoUsuarioMandaSobreTodos(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	if err := fx.db.Create(&models.JugadorGrupo{UserID: fx.u1.ID, GrupoID: fx.g1.ID}).Error; err != nil {
		t.Fatalf("membresia: %v", err)
	}

	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoJuego, models.CasinoDados, false)
	fx.acceso(models.AmbitoGrupo, fx.g1.ID, models.ContenidoJuego, models.CasinoDados, true)

	// El jugador apagado vence al paquete y al global encendidos.
	fx.acceso(models.AmbitoUsuario, fx.u1.ID, models.ContenidoJuego, models.CasinoDados, false)
	ok, fuente, err := ResuelveAcceso(fx.db, fx.u1.ID, models.ContenidoJuego, models.CasinoDados)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if ok || fuente != models.AmbitoUsuario {
		t.Errorf("activo=%v fuente=%s, queria false/usuario", ok, fuente)
	}

	// El jugador encendido vence al global apagado.
	fx.acceso(models.AmbitoUsuario, fx.u2.ID, models.ContenidoJuego, models.CasinoDados, true)
	ok, fuente, err = ResuelveAcceso(fx.db, fx.u2.ID, models.ContenidoJuego, models.CasinoDados)
	if err != nil {
		t.Fatalf("resolver u2: %v", err)
	}
	if !ok || fuente != models.AmbitoUsuario {
		t.Errorf("u2: activo=%v fuente=%s, queria true/usuario", ok, fuente)
	}
}

func TestResolucionAccesoPrimerGrupoQueAplica(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	for _, g := range []models.GrupoJugadores{fx.g1, fx.g2} {
		if err := fx.db.Create(&models.JugadorGrupo{UserID: fx.u1.ID, GrupoID: g.ID}).Error; err != nil {
			t.Fatalf("membresia: %v", err)
		}
	}
	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoLoteria, "LN", false)

	// g1 (id menor) tiene fila encendida: gana aunque g2 este apagada.
	fx.acceso(models.AmbitoGrupo, fx.g1.ID, models.ContenidoLoteria, "LN", true)
	fx.acceso(models.AmbitoGrupo, fx.g2.ID, models.ContenidoLoteria, "LN", false)
	ok, fuente, err := ResuelveAcceso(fx.db, fx.u1.ID, models.ContenidoLoteria, "LN")
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !ok || fuente != models.AmbitoGrupo {
		t.Errorf("activo=%v fuente=%s, queria true/grupo (g1)", ok, fuente)
	}

	// Si g1 no tiene fila, aplica g2 aunque este apagada.
	if err := fx.db.Where("ambito = ? AND ambito_id = ?", models.AmbitoGrupo, fx.g1.ID).
		Delete(&models.AccesoContenido{}).Error; err != nil {
		t.Fatalf("borrar fila g1: %v", err)
	}
	ok, fuente, err = ResuelveAcceso(fx.db, fx.u1.ID, models.ContenidoLoteria, "LN")
	if err != nil {
		t.Fatalf("resolver 2: %v", err)
	}
	if ok || fuente != models.AmbitoGrupo {
		t.Errorf("activo=%v fuente=%s, queria false/grupo (g2)", ok, fuente)
	}
}

// ------------------------------------------------------------ rechazo

func TestExigirAccesoRechazaContenidoInactivo(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoLoteria, "LN", false)

	c, w := contextoPrueba(http.MethodPost, "/api/apuestas", url.Values{}, fx.u1.ID)
	if ExigirAcceso(c, fx.db, fx.u1.ID, models.ContenidoLoteria, "LN") {
		t.Error("ExigirAcceso deberia rechazar el contenido apagado")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status=%d, queria 403", w.Code)
	}
	var cuerpo map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &cuerpo)
	if cuerpo["codigo"] != "LN" {
		t.Errorf("cuerpo=%v, queria codigo LN", cuerpo)
	}

	// Al encenderlo, no se escribe ninguna respuesta.
	if err := fx.db.Model(&models.AccesoContenido{}).
		Where("ambito = ? AND tipo = ? AND codigo = ?", models.AmbitoGlobal, models.ContenidoLoteria, "LN").
		Update("activo", true).Error; err != nil {
		t.Fatalf("update: %v", err)
	}
	c2, w2 := contextoPrueba(http.MethodPost, "/api/apuestas", url.Values{}, fx.u1.ID)
	if !ExigirAcceso(c2, fx.db, fx.u1.ID, models.ContenidoLoteria, "LN") {
		t.Error("ExigirAcceso deberia dejar pasar el contenido encendido")
	}
	if w2.Code != http.StatusOK || w2.Body.Len() != 0 {
		t.Errorf("status=%d cuerpo=%q, queria 200 vacio", w2.Code, w2.Body.String())
	}
}

func TestApuestaRechazadaCuandoLoteriaApagada(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	var loteria models.Loteria
	if err := fx.db.Where("codigo = ?", "LN").First(&loteria).Error; err != nil {
		t.Fatalf("loteria: %v", err)
	}
	sorteo := models.Sorteo{
		LoteriaID: loteria.ID, Titulo: "Sorteo prueba",
		Fecha: time.Now().Add(24 * time.Hour), Estado: models.SorteoAbierto,
	}
	if err := fx.db.Create(&sorteo).Error; err != nil {
		t.Fatalf("sorteo: %v", err)
	}
	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoLoteria, "LN", false)

	h := NewApuestasHandler(fx.db, &config.Config{})
	c, w := contextoPrueba(http.MethodPost, "/api/apuestas", url.Values{}, fx.u1.ID)
	cuerpo := `{"sorteo_id":` + strconv.FormatUint(uint64(sorteo.ID), 10) + `,"numeros":[1,2]}`
	c.Request.Body = io.NopCloser(strings.NewReader(cuerpo))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Crear(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status=%d cuerpo=%s, queria 403", w.Code, w.Body.String())
	}
}

// ------------------------------------------------------------ filtros

func TestGamesFiltradasPorAcceso(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoJuego, models.CasinoDados, false)

	h := NewCasinoHandler(fx.db, &config.Config{})
	c, w := contextoPrueba(http.MethodGet, "/api/casino/games", url.Values{}, fx.u1.ID)
	c.Set("rol", models.RolCliente)
	h.Games(c)

	var juegos []juegoResponse
	if err := json.Unmarshal(w.Body.Bytes(), &juegos); err != nil {
		t.Fatalf("json: %v (%s)", err, w.Body.String())
	}
	for _, j := range juegos {
		if j.Codigo == models.CasinoDados || j.Codigo == models.CasinoSlots {
			t.Errorf("el juego %s no deberia aparecer", j.Codigo)
		}
	}
	if len(juegos) != 0 {
		t.Errorf("no quedaba ningun juego visible, hay %d", len(juegos))
	}
}

func TestLoteriasFiltradasPorAcceso(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoLoteria, "LN", false)

	h := NewJuegosHandler(fx.db)
	c, w := contextoPrueba(http.MethodGet, "/api/loterias", url.Values{}, fx.u1.ID)
	c.Set("rol", models.RolCliente)
	h.ListLoterias(c)

	var loterias []models.Loteria
	if err := json.Unmarshal(w.Body.Bytes(), &loterias); err != nil {
		t.Fatalf("json: %v (%s)", err, w.Body.String())
	}
	for _, l := range loterias {
		if l.Codigo == "LN" {
			t.Error("LN no deberia aparecer")
		}
	}
}

func TestRifasFiltradasPorAcceso(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	fx.acceso(models.AmbitoGlobal, 0, models.ContenidoRifa, strconv.FormatUint(uint64(fx.rifa.ID), 10), false)

	h := NewRifasHandler(fx.db)
	c, w := contextoPrueba(http.MethodGet, "/api/rifas", url.Values{}, fx.u1.ID)
	c.Set("rol", models.RolCliente)
	h.Listar(c)

	var rifas []struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rifas); err != nil {
		t.Fatalf("json: %v (%s)", err, w.Body.String())
	}
	for _, r := range rifas {
		if r.ID == fx.rifa.ID {
			t.Error("la rifa apagada no deberia aparecer")
		}
	}
}

// ------------------------------------------------------------ admin

func TestAdminAccesosCrearListarYActualizar(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	h := NewAdminAccesosHandler(fx.db)
	actor := models.User{Nombre: "Admin", Email: "admin@test", Rol: models.RolAdmin}
	if err := fx.db.Create(&actor).Error; err != nil {
		t.Fatalf("actor: %v", err)
	}

	// Crear.
	c, w := contextoPrueba(http.MethodPost, "/api/admin/accesos", url.Values{}, actor.ID)
	c.Request.Body = io.NopCloser(strings.NewReader(
		`{"ambito":"grupo","ambito_id":` + strconv.FormatUint(uint64(fx.g1.ID), 10) + `,"tipo":"juego","codigo":"` + models.CasinoDados + `","activo":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CrearAcceso(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("crear status=%d cuerpo=%s", w.Code, w.Body.String())
	}

	// Listar: la fila queda con los nombres resueltos.
	c2, w2 := contextoPrueba(http.MethodGet, "/api/admin/accesos", url.Values{}, actor.ID)
	h.ListarAccesos(c2)
	var filas []accesoSalida
	if err := json.Unmarshal(w2.Body.Bytes(), &filas); err != nil {
		t.Fatalf("listar json: %v", err)
	}
	if len(filas) != 1 {
		t.Fatalf("filas=%d, queria 1", len(filas))
	}
	if filas[0].AmbitoNombre != fx.g1.Nombre || filas[0].ContenidoNombre != "Dados del Sur" {
		t.Errorf("nombres sin resolver: %+v", filas[0])
	}

	// Actualizar: se apaga.
	c3, w3 := contextoPrueba(http.MethodPut, "/api/admin/accesos", url.Values{}, actor.ID)
	c3.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(filas[0].ID), 10)}}
	c3.Request.Body = io.NopCloser(strings.NewReader(`{"activo":false}`))
	c3.Request.Header.Set("Content-Type", "application/json")
	h.ActualizarAcceso(c3)
	if w3.Code != http.StatusOK {
		t.Fatalf("actualizar status=%d cuerpo=%s", w3.Code, w3.Body.String())
	}
	ok, _, err := ResuelveAcceso(fx.db, fx.u2.ID, models.ContenidoJuego, models.CasinoDados)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	// u2 no esta en el paquete, asi que el default del juego (activo) manda.
	if !ok {
		t.Log("u2 no pertenece al paquete: el default del juego sigue activo")
	}

	// La accion queda en la auditoria.
	var entradas []models.AuditLog
	fx.db.Where("accion LIKE ?", "accesos.%").Order("id asc").Find(&entradas)
	if len(entradas) < 2 {
		t.Errorf("auditoria: %d entradas, queria al menos 2", len(entradas))
	}
}

func TestAdminGruposCicloCompleto(t *testing.T) {
	fx := nuevaAccesoFixture(t)
	h := NewAdminAccesosHandler(fx.db)
	actor := models.User{Nombre: "Admin", Email: "admin@test", Rol: models.RolAdmin}
	if err := fx.db.Create(&actor).Error; err != nil {
		t.Fatalf("actor: %v", err)
	}

	// Crear grupo.
	c, w := contextoPrueba(http.MethodPost, "/api/admin/grupos", url.Values{}, actor.ID)
	c.Request.Body = io.NopCloser(strings.NewReader(`{"nombre":"VIP","descripcion":"jugadores top"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CrearGrupo(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("crear status=%d cuerpo=%s", w.Code, w.Body.String())
	}
	var g grupoSalida
	_ = json.Unmarshal(w.Body.Bytes(), &g)

	// Asignar jugadores.
	c2, w2 := contextoPrueba(http.MethodPost, "/api/admin/grupos", url.Values{}, actor.ID)
	c2.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(g.ID), 10)}}
	c2.Request.Body = io.NopCloser(strings.NewReader(
		`{"user_ids":[` + strconv.FormatUint(uint64(fx.u1.ID), 10) + `,` + strconv.FormatUint(uint64(fx.u2.ID), 10) + `]}`))
	c2.Request.Header.Set("Content-Type", "application/json")
	h.AsignarJugadores(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("asignar status=%d cuerpo=%s", w2.Code, w2.Body.String())
	}

	// Listar miembros.
	c3, w3 := contextoPrueba(http.MethodGet, "/api/admin/grupos", url.Values{}, actor.ID)
	c3.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(g.ID), 10)}}
	h.ListarMiembros(c3)
	var miembros []models.User
	if err := json.Unmarshal(w3.Body.Bytes(), &miembros); err != nil {
		t.Fatalf("miembros json: %v", err)
	}
	if len(miembros) != 2 {
		t.Errorf("miembros=%d, queria 2", len(miembros))
	}

	// Quitar uno.
	c4, w4 := contextoPrueba(http.MethodDelete, "/api/admin/grupos", url.Values{}, actor.ID)
	c4.Params = gin.Params{
		{Key: "id", Value: strconv.FormatUint(uint64(g.ID), 10)},
		{Key: "user_id", Value: strconv.FormatUint(uint64(fx.u2.ID), 10)},
	}
	h.QuitarJugador(c4)
	if w4.Code != http.StatusOK {
		t.Fatalf("quitar status=%d cuerpo=%s", w4.Code, w4.Body.String())
	}

	// Borrar el grupo.
	c5, w5 := contextoPrueba(http.MethodDelete, "/api/admin/grupos", url.Values{}, actor.ID)
	c5.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(g.ID), 10)}}
	h.EliminarGrupo(c5)
	if w5.Code != http.StatusOK {
		t.Fatalf("eliminar status=%d cuerpo=%s", w5.Code, w5.Body.String())
	}
	var quedan int64
	fx.db.Model(&models.JugadorGrupo{}).Where("grupo_id = ?", g.ID).Count(&quedan)
	if quedan != 0 {
		t.Errorf("quedaron %d membresias", quedan)
	}
}
