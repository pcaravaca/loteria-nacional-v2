package models

import "time"

// Ambitos de un acceso. Se resuelven de mas particular a mas general:
// jugador > paquete > global.
const (
	AmbitoGlobal  = "global"
	AmbitoGrupo   = "grupo"
	AmbitoUsuario = "usuario"
)

// Tipos de contenido cuyo acceso se puede encender o apagar.
const (
	ContenidoJuego   = "juego"
	ContenidoLoteria = "loteria"
	ContenidoRifa    = "rifa"
)

// GrupoJugadores es un paquete de jugadores: permite dar (o quitar) el acceso
// a un contenido para un grupo entero de una vez.
type GrupoJugadores struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Nombre      string    `gorm:"not null;size:120" json:"nombre"`
	Descripcion string    `json:"descripcion"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (GrupoJugadores) TableName() string { return "grupos_jugadores" }

// JugadorGrupo es la membresia de un jugador a un paquete. La PK compuesta
// (user_id, grupo_id) impide asignaciones duplicadas.
type JugadorGrupo struct {
	UserID  uint `gorm:"primaryKey" json:"user_id"`
	GrupoID uint `gorm:"primaryKey" json:"grupo_id"`
}

func (JugadorGrupo) TableName() string { return "jugador_grupo" }

// AccesoContenido enciende o apaga un contenido (juego, loteria o rifa) para
// un ambito: global, un paquete de jugadores o un jugador individual.
//
// AmbitoID es 0 para el ambito global y no es nullable a proposito: con NULL
// los motores tratan cada fila como distinta, y el indice unico
// (ambito, ambito_id, tipo, codigo) no podria garantizar que no haya dos filas
// globales para el mismo contenido.
//
// Codigo identifica el contenido dentro del tipo: el codigo del juego o de la
// loteria, y el numero de rifa (como texto) para las rifas, que no tienen
// codigo propio.
type AccesoContenido struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Ambito    string    `gorm:"not null;size:16;uniqueIndex:acceso_unico" json:"ambito"`
	AmbitoID  uint      `gorm:"not null;default:0;uniqueIndex:acceso_unico" json:"ambito_id"`
	Tipo      string    `gorm:"not null;size:16;uniqueIndex:acceso_unico" json:"tipo"`
	Codigo    string    `gorm:"not null;size:64;uniqueIndex:acceso_unico" json:"codigo"`
	Activo    bool      `gorm:"not null;default:false" json:"activo"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AccesoContenido) TableName() string { return "accesos_contenido" }
