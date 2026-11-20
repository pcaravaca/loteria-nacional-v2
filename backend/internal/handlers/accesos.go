package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// ResuelveAcceso decide si un contenido esta activo para un usuario.
//
// Prioridad: jugador > paquete (el primero que aplique, por id de paquete) >
// global. Si no hay ninguna fila para el contenido, el valor por defecto es el
// estado global del contenido: casino_juegos.activo para juegos, loterias.activa
// para loterias y activo para rifas (las rifas no tienen interruptor propio).
//
// El segundo valor devuelto indica de donde salio la decision: "usuario",
// "grupo", "global" o "default".
func ResuelveAcceso(db *gorm.DB, userID uint, tipo, codigo string) (bool, string, error) {
	// 1) Override del jugador: manda sobre cualquier otra cosa.
	var fila models.AccesoContenido
	err := db.Where("ambito = ? AND ambito_id = ? AND tipo = ? AND codigo = ?",
		models.AmbitoUsuario, userID, tipo, codigo).First(&fila).Error
	if err == nil {
		return fila.Activo, models.AmbitoUsuario, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", err
	}

	// 2) Paquetes del jugador: gana el primero que tenga fila, en orden de id.
	sub := db.Model(&models.JugadorGrupo{}).Select("grupo_id").Where("user_id = ?", userID)
	var grupo models.AccesoContenido
	err = db.Where("ambito = ? AND tipo = ? AND codigo = ? AND ambito_id IN (?)",
		models.AmbitoGrupo, tipo, codigo, sub).
		Order("ambito_id asc").First(&grupo).Error
	if err == nil {
		return grupo.Activo, models.AmbitoGrupo, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", err
	}

	// 3) Ambito global.
	var global models.AccesoContenido
	err = db.Where("ambito = ? AND ambito_id = 0 AND tipo = ? AND codigo = ?",
		models.AmbitoGlobal, tipo, codigo).First(&global).Error
	if err == nil {
		return global.Activo, models.AmbitoGlobal, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", err
	}

	// 4) Linea base segun el tipo de contenido.
	switch tipo {
	case models.ContenidoJuego:
		var juego models.CasinoGame
		if err := db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
			return false, "default", err
		}
		return juego.Activo, "default", nil
	case models.ContenidoLoteria:
		var loteria models.Loteria
		if err := db.Where("codigo = ?", codigo).First(&loteria).Error; err != nil {
			return false, "default", err
		}
		return loteria.Activa, "default", nil
	default:
		// Las rifas no tienen interruptor global: si nadie dijo nada, se ve.
		return true, "default", nil
	}
}

// ExigirAcceso responde 403 y devuelve false cuando el contenido no esta
// activo para el usuario. Se usa en los handlers de jugada para que la
// restriccion no dependa solo de la interfaz.
func ExigirAcceso(c *gin.Context, db *gorm.DB, userID uint, tipo, codigo string) bool {
	activo, ambito, err := ResuelveAcceso(db, userID, tipo, codigo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo verificar el acceso"})
		return false
	}
	if !activo {
		c.JSON(http.StatusForbidden, gin.H{
			"error":  "este contenido no está disponible para tu cuenta",
			"tipo":   tipo,
			"codigo": codigo,
			"ambito": ambito,
		})
		return false
	}
	return true
}

// GruposDelUsuario devuelve los ids de paquete del usuario, en orden
// ascendente (el orden en que se evaluan).
func GruposDelUsuario(db *gorm.DB, userID uint) ([]uint, error) {
	var ids []uint
	err := db.Model(&models.JugadorGrupo{}).Where("user_id = ?", userID).
		Order("grupo_id asc").Pluck("grupo_id", &ids).Error
	return ids, err
}

// rolOperador indica si el rol en el contexto es admin o agente: esos roles
// administran el contenido y por lo tanto ven el catalogo completo.
func rolOperador(c *gin.Context) bool {
	rol, _ := c.Get("rol")
	return rol == models.RolAdmin || rol == models.RolAgente
}

// LoteriasVisibles devuelve los ids de loteria que el usuario puede ver.
func LoteriasVisibles(db *gorm.DB, userID uint) ([]uint, error) {
	var loterias []models.Loteria
	if err := db.Find(&loterias).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(loterias))
	for _, l := range loterias {
		ok, _, err := ResuelveAcceso(db, userID, models.ContenidoLoteria, l.Codigo)
		if err != nil {
			return nil, err
		}
		if ok {
			ids = append(ids, l.ID)
		}
	}
	return ids, nil
}

// RifasVisibles devuelve los ids de rifa que el usuario puede ver.
func RifasVisibles(db *gorm.DB, userID uint) ([]uint, error) {
	var rifas []models.Rifa
	if err := db.Find(&rifas).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rifas))
	for _, r := range rifas {
		ok, _, err := ResuelveAcceso(db, userID, models.ContenidoRifa, strconv.FormatUint(uint64(r.ID), 10))
		if err != nil {
			return nil, err
		}
		if ok {
			ids = append(ids, r.ID)
		}
	}
	return ids, nil
}

// JuegosVisibles devuelve el mapa codigo -> visible para el casino.
func JuegosVisibles(db *gorm.DB, userID uint) (map[string]bool, error) {
	var juegos []models.CasinoGame
	if err := db.Find(&juegos).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(juegos))
	for _, j := range juegos {
		ok, _, err := ResuelveAcceso(db, userID, models.ContenidoJuego, j.Codigo)
		if err != nil {
			return nil, err
		}
		out[j.Codigo] = ok
	}
	return out, nil
}
