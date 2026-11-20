package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// AdminAccesosHandler administra los paquetes de jugadores y los accesos por
// contenido (juego, loteria o rifa). Todo queda en la auditoria existente.
type AdminAccesosHandler struct {
	db *gorm.DB
}

func NewAdminAccesosHandler(db *gorm.DB) *AdminAccesosHandler {
	return &AdminAccesosHandler{db: db}
}

// ---------------------------------------------------------------- paquetes

type grupoRequest struct {
	Nombre      string `json:"nombre" binding:"required"`
	Descripcion string `json:"descripcion"`
}

type grupoSalida struct {
	models.GrupoJugadores
	Miembros int64 `json:"miembros"`
}

func (h *AdminAccesosHandler) grupoSalidaDe(g models.GrupoJugadores) grupoSalida {
	var n int64
	h.db.Model(&models.JugadorGrupo{}).Where("grupo_id = ?", g.ID).Count(&n)
	return grupoSalida{GrupoJugadores: g, Miembros: n}
}

// ListarGrupos devuelve los paquetes con la cantidad de miembros.
func (h *AdminAccesosHandler) ListarGrupos(c *gin.Context) {
	var grupos []models.GrupoJugadores
	if err := h.db.Order("nombre asc").Find(&grupos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los paquetes"})
		return
	}
	salida := make([]grupoSalida, 0, len(grupos))
	for _, g := range grupos {
		salida = append(salida, h.grupoSalidaDe(g))
	}
	c.JSON(http.StatusOK, salida)
}

// CrearGrupo crea un paquete vacio.
func (h *AdminAccesosHandler) CrearGrupo(c *gin.Context) {
	var req grupoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nombre requerido"})
		return
	}
	g := models.GrupoJugadores{Nombre: strings.TrimSpace(req.Nombre), Descripcion: strings.TrimSpace(req.Descripcion)}
	if g.Nombre == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nombre requerido"})
		return
	}
	if err := h.db.Create(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear el paquete"})
		return
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.grupo.crear", "grupo",
		strconv.FormatUint(uint64(g.ID), 10), g.Nombre)
	c.JSON(http.StatusCreated, h.grupoSalidaDe(g))
}

// ActualizarGrupo renombra un paquete o cambia su descripcion.
func (h *AdminAccesosHandler) ActualizarGrupo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var g models.GrupoJugadores
	if err := h.db.First(&g, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "paquete no encontrado"})
		return
	}
	var req grupoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nombre requerido"})
		return
	}
	g.Nombre = strings.TrimSpace(req.Nombre)
	g.Descripcion = strings.TrimSpace(req.Descripcion)
	if g.Nombre == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nombre requerido"})
		return
	}
	if err := h.db.Save(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el paquete"})
		return
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.grupo.actualizar", "grupo",
		strconv.FormatUint(id, 10), g.Nombre)
	c.JSON(http.StatusOK, h.grupoSalidaDe(g))
}

// EliminarGrupo borra el paquete, sus membresias y los accesos que apuntaban a
// el (una fila huérfana no tendria efecto y solo ensuciaria el panel).
func (h *AdminAccesosHandler) EliminarGrupo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var g models.GrupoJugadores
	if err := h.db.First(&g, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "paquete no encontrado"})
		return
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("grupo_id = ?", id).Delete(&models.JugadorGrupo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ambito = ? AND ambito_id = ?", models.AmbitoGrupo, id).
			Delete(&models.AccesoContenido{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.GrupoJugadores{}, id).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo eliminar el paquete"})
		return
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.grupo.eliminar", "grupo",
		strconv.FormatUint(id, 10), g.Nombre)
	c.JSON(http.StatusOK, gin.H{"id": id, "eliminado": true})
}

// ListarMiembros devuelve los jugadores del paquete.
func (h *AdminAccesosHandler) ListarMiembros(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var ids []uint
	if err := h.db.Model(&models.JugadorGrupo{}).Where("grupo_id = ?", id).
		Order("user_id asc").Pluck("user_id", &ids).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los miembros"})
		return
	}
	if len(ids) == 0 {
		c.JSON(http.StatusOK, []models.User{})
		return
	}
	var users []models.User
	if err := h.db.Where("id IN ?", ids).Order("nombre asc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los miembros"})
		return
	}
	c.JSON(http.StatusOK, users)
}

type asignarJugadoresRequest struct {
	UserID  *uint  `json:"user_id"`
	UserIDs []uint `json:"user_ids"`
}

// AsignarJugadores suma jugadores al paquete. Acepta user_id solo o user_ids.
func (h *AdminAccesosHandler) AsignarJugadores(c *gin.Context) {
	grupoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var g models.GrupoJugadores
	if err := h.db.First(&g, grupoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "paquete no encontrado"})
		return
	}
	var req asignarJugadoresRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}
	ids := append([]uint{}, req.UserIDs...)
	if req.UserID != nil {
		ids = append(ids, *req.UserID)
	}
	vistos := map[uint]bool{}
	unicos := make([]uint, 0, len(ids))
	for _, uid := range ids {
		if uid == 0 || vistos[uid] {
			continue
		}
		vistos[uid] = true
		unicos = append(unicos, uid)
	}
	if len(unicos) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "indica al menos un jugador"})
		return
	}
	var existen int64
	h.db.Model(&models.User{}).Where("id IN ?", unicos).Count(&existen)
	if int(existen) != len(unicos) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hay jugadores que no existen"})
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		for _, uid := range unicos {
			fila := models.JugadorGrupo{UserID: uid, GrupoID: uint(grupoID)}
			if err := tx.Where(fila).FirstOrCreate(&fila).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron asignar los jugadores"})
		return
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.grupo.asignar", "grupo",
		strconv.FormatUint(grupoID, 10), fmt.Sprintf("%d jugadores: %v", len(unicos), unicos))
	c.JSON(http.StatusOK, gin.H{"grupo_id": grupoID, "asignados": len(unicos)})
}

// QuitarJugador saca a un jugador del paquete.
func (h *AdminAccesosHandler) QuitarJugador(c *gin.Context) {
	grupoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jugador invalido"})
		return
	}
	if err := h.db.Where("grupo_id = ? AND user_id = ?", grupoID, userID).
		Delete(&models.JugadorGrupo{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo quitar el jugador"})
		return
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.grupo.quitar", "grupo",
		strconv.FormatUint(grupoID, 10), "jugador="+strconv.FormatUint(userID, 10))
	c.JSON(http.StatusOK, gin.H{"grupo_id": grupoID, "user_id": userID, "quitado": true})
}

// ---------------------------------------------------------------- accesos

type accesoRequest struct {
	Ambito   string `json:"ambito" binding:"required,oneof=global grupo usuario"`
	AmbitoID *uint  `json:"ambito_id"`
	Tipo     string `json:"tipo" binding:"required,oneof=juego loteria rifa"`
	Codigo   string `json:"codigo" binding:"required"`
	Activo   *bool  `json:"activo"`
}

type accesoSalida struct {
	models.AccesoContenido
	AmbitoNombre    string `json:"ambito_nombre"`
	ContenidoNombre string `json:"contenido_nombre"`
}

func (h *AdminAccesosHandler) validarContenido(tipo, codigo string) error {
	switch tipo {
	case models.ContenidoJuego:
		var n int64
		h.db.Model(&models.CasinoGame{}).Where("codigo = ?", codigo).Count(&n)
		if n == 0 {
			return fmt.Errorf("el juego no existe")
		}
	case models.ContenidoLoteria:
		var n int64
		h.db.Model(&models.Loteria{}).Where("codigo = ?", codigo).Count(&n)
		if n == 0 {
			return fmt.Errorf("la loteria no existe")
		}
	case models.ContenidoRifa:
		id, err := strconv.ParseUint(codigo, 10, 64)
		if err != nil {
			return fmt.Errorf("el codigo de rifa debe ser el numero de rifa")
		}
		var n int64
		h.db.Model(&models.Rifa{}).Where("id = ?", id).Count(&n)
		if n == 0 {
			return fmt.Errorf("la rifa no existe")
		}
	}
	return nil
}

// validarAmbito devuelve el ambito_id efectivo y verifica que exista.
func (h *AdminAccesosHandler) validarAmbito(req accesoRequest) (uint, error) {
	switch req.Ambito {
	case models.AmbitoGlobal:
		return 0, nil
	case models.AmbitoGrupo:
		if req.AmbitoID == nil {
			return 0, fmt.Errorf("ambito_id requerido para el ambito grupo")
		}
		var n int64
		h.db.Model(&models.GrupoJugadores{}).Where("id = ?", *req.AmbitoID).Count(&n)
		if n == 0 {
			return 0, fmt.Errorf("el paquete no existe")
		}
		return *req.AmbitoID, nil
	case models.AmbitoUsuario:
		if req.AmbitoID == nil {
			return 0, fmt.Errorf("ambito_id requerido para el ambito usuario")
		}
		var n int64
		h.db.Model(&models.User{}).Where("id = ?", *req.AmbitoID).Count(&n)
		if n == 0 {
			return 0, fmt.Errorf("el jugador no existe")
		}
		return *req.AmbitoID, nil
	}
	return 0, fmt.Errorf("ambito invalido")
}

func (h *AdminAccesosHandler) completarAcceso(f models.AccesoContenido) accesoSalida {
	out := accesoSalida{AccesoContenido: f}
	switch f.Ambito {
	case models.AmbitoGrupo:
		var g models.GrupoJugadores
		if err := h.db.Select("nombre").First(&g, f.AmbitoID).Error; err == nil {
			out.AmbitoNombre = g.Nombre
		}
	case models.AmbitoUsuario:
		var u models.User
		if err := h.db.Select("nombre").First(&u, f.AmbitoID).Error; err == nil {
			out.AmbitoNombre = u.Nombre
		}
	default:
		out.AmbitoNombre = "Todos"
	}
	switch f.Tipo {
	case models.ContenidoJuego:
		var j models.CasinoGame
		if err := h.db.Select("nombre").First(&j, "codigo = ?", f.Codigo).Error; err == nil {
			out.ContenidoNombre = j.Nombre
		}
	case models.ContenidoLoteria:
		var l models.Loteria
		if err := h.db.Select("nombre").First(&l, "codigo = ?", f.Codigo).Error; err == nil {
			out.ContenidoNombre = l.Nombre
		}
	case models.ContenidoRifa:
		if id, err := strconv.ParseUint(f.Codigo, 10, 64); err == nil {
			var r models.Rifa
			if err := h.db.Select("titulo").First(&r, id).Error; err == nil {
				out.ContenidoNombre = r.Titulo
			}
		}
	}
	return out
}

// ListarAccesos devuelve las filas de acceso con los nombres resueltos.
func (h *AdminAccesosHandler) ListarAccesos(c *gin.Context) {
	q := h.db.Model(&models.AccesoContenido{})
	if v := c.Query("ambito"); v != "" {
		q = q.Where("ambito = ?", v)
	}
	if v := c.Query("tipo"); v != "" {
		q = q.Where("tipo = ?", v)
	}
	if v := c.Query("codigo"); v != "" {
		q = q.Where("codigo = ?", v)
	}
	var filas []models.AccesoContenido
	if err := q.Order("ambito asc, tipo asc, codigo asc, ambito_id asc").Find(&filas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los accesos"})
		return
	}
	salida := make([]accesoSalida, 0, len(filas))
	for _, f := range filas {
		salida = append(salida, h.completarAcceso(f))
	}
	c.JSON(http.StatusOK, salida)
}

// CrearAcceso crea o actualiza la fila de un contenido en un ambito (es un
// upsert: el unique de (ambito, ambito_id, tipo, codigo) deja una sola fila).
func (h *AdminAccesosHandler) CrearAcceso(c *gin.Context) {
	var req accesoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}
	ambitoID, err := h.validarAmbito(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	codigo := strings.TrimSpace(req.Codigo)
	if err := h.validarContenido(req.Tipo, codigo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	activo := true
	if req.Activo != nil {
		activo = *req.Activo
	}

	var existente models.AccesoContenido
	err = h.db.Where("ambito = ? AND ambito_id = ? AND tipo = ? AND codigo = ?",
		req.Ambito, ambitoID, req.Tipo, codigo).First(&existente).Error
	if err == nil {
		if err := h.db.Model(&models.AccesoContenido{}).Where("id = ?", existente.ID).
			Update("activo", activo).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el acceso"})
			return
		}
		existente.Activo = activo
		RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.actualizar", "acceso",
			strconv.FormatUint(uint64(existente.ID), 10), h.detalleAcceso(existente))
		c.JSON(http.StatusOK, h.completarAcceso(existente))
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el acceso"})
		return
	}

	fila := models.AccesoContenido{
		Ambito: req.Ambito, AmbitoID: ambitoID, Tipo: req.Tipo, Codigo: codigo, Activo: activo,
	}
	if err := h.db.Create(&fila).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear el acceso"})
		return
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.crear", "acceso",
		strconv.FormatUint(uint64(fila.ID), 10), h.detalleAcceso(fila))
	c.JSON(http.StatusCreated, h.completarAcceso(fila))
}

// ActualizarAcceso cambia el estado de una fila de acceso.
func (h *AdminAccesosHandler) ActualizarAcceso(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var fila models.AccesoContenido
	if err := h.db.First(&fila, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "acceso no encontrado"})
		return
	}
	var req struct {
		Activo *bool `json:"activo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}
	if req.Activo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no hay cambios que aplicar"})
		return
	}
	if err := h.db.Model(&models.AccesoContenido{}).Where("id = ?", id).
		Update("activo", *req.Activo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar el acceso"})
		return
	}
	fila.Activo = *req.Activo
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.actualizar", "acceso",
		strconv.FormatUint(id, 10), h.detalleAcceso(fila))
	c.JSON(http.StatusOK, h.completarAcceso(fila))
}

// EliminarAcceso borra la fila: el contenido vuelve a regirse por el ambito
// siguiente en la cadena (grupo, global o el estado por defecto).
func (h *AdminAccesosHandler) EliminarAcceso(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var fila models.AccesoContenido
	if err := h.db.First(&fila, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "acceso no encontrado"})
		return
	}
	if err := h.db.Delete(&models.AccesoContenido{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo eliminar el acceso"})
		return
	}
	RegistrarAuditoria(h.db, c, c.GetUint("userID"), "accesos.eliminar", "acceso",
		strconv.FormatUint(id, 10), h.detalleAcceso(fila))
	c.JSON(http.StatusOK, gin.H{"id": id, "eliminado": true})
}

func (h *AdminAccesosHandler) detalleAcceso(f models.AccesoContenido) string {
	return fmt.Sprintf("ambito=%s id=%d tipo=%s codigo=%s activo=%t",
		f.Ambito, f.AmbitoID, f.Tipo, f.Codigo, f.Activo)
}
