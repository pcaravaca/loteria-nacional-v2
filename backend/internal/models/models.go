package models

import "time"

// Rol de usuario dentro de la plataforma.
const (
	RolCliente = "cliente"
	RolAgente  = "agente"
	RolAdmin   = "admin"
)

// Tipo de juego.
const (
	TipoLoteria  = "loteria"
	TipoQuiniela = "quiniela"
)

// Estado de un sorteo.
const (
	SorteoAbierto  = "abierto"
	SorteoCerrado  = "cerrado"
	SorteoPagado   = "pagado"
)

// Estado de una apuesta.
const (
	ApuestaPendiente = "pendiente"
	ApuestaGanadora  = "ganadora"
	ApuestaPerdedora = "perdedora"
	ApuestaPagada    = "pagada"
)

// Movimiento de saldo.
const (
	MovRecarga = "recarga"
	MovApuesta = "apuesta"
	MovPremio  = "premio"
	MovBono    = "bono"
	MovRetiro    = "retiro"
	MovAjuste    = "ajuste"
	MovCasinoApuesta = "casino_apuesta"
	MovCasinoPremio  = "casino_premio"
)

// Nombres de tabla explícitos (en español, plural) para no depender de la
// pluralización automática de GORM.
func (User) TableName() string        { return "usuarios" }
func (Loteria) TableName() string     { return "loterias" }
func (Sorteo) TableName() string      { return "sorteos" }
func (Apuesta) TableName() string     { return "apuestas" }
func (Movimiento) TableName() string  { return "movimientos" }

// User es un jugador, agente o administrador.
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Nombre       string    `gorm:"not null" json:"nombre"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"column:password_hash" json:"-"`
	Rol          string    `gorm:"not null;default:cliente" json:"rol"`
	Telefono     string    `json:"telefono"`
	Saldo        int64     `gorm:"not null;default:0" json:"saldo"`
	Bono         int64     `gorm:"not null;default:0" json:"bono"`
	// Semilla del casino (provably fair): el hash se publica antes de jugar y
	// la semilla se revela al verificar la ronda.
	CasinoSeed  string `gorm:"size:64" json:"-"`
	CasinoNonce int64  `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Loteria es un juego (Lotería Nacional, Quiniela, etc.).
type Loteria struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Codigo        string    `gorm:"uniqueIndex;not null" json:"codigo"`
	Nombre        string    `gorm:"not null" json:"nombre"`
	Tipo          string    `gorm:"not null;default:loteria" json:"tipo"`
	Descripcion   string    `json:"descripcion"`
	CostoNumero   int64     `gorm:"not null;default:1000" json:"costo_numero"`
	LongitudNum   int       `gorm:"not null;default:2" json:"longitud_num"`
	MaxNumeros    int       `gorm:"not null;default:1" json:"max_numeros"`
	Multiplicador int64     `gorm:"not null;default:50" json:"multiplicador"`
	Activa        bool      `gorm:"not null;default:true" json:"activa"`
	CreatedAt     time.Time `json:"created_at"`
}

// Sorteo es un concurso con fecha y, al cerrarse, números ganadores.
type Sorteo struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	LoteriaID       uint      `gorm:"not null;index" json:"loteria_id"`
	Titulo          string    `gorm:"not null" json:"titulo"`
	Fecha           time.Time `gorm:"not null;index" json:"fecha"`
	Estado          string    `gorm:"not null;default:abierto" json:"estado"`
	NumerosGanadores []int    `gorm:"-" json:"numeros_ganadores"`
	NumerosRaw      string    `gorm:"column:numeros_raw" json:"-"`
	CreatedAt       time.Time `json:"created_at"`

	Loteria Loteria `gorm:"foreignKey:LoteriaID" json:"loteria,omitempty"`
}

// Apuesta es un ticket de juego de un usuario para un sorteo.
type Apuesta struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	SorteoID  uint      `gorm:"not null;index" json:"sorteo_id"`
	Numeros   []int     `gorm:"-" json:"numeros"`
	NumerosRaw string   `gorm:"column:numeros_raw" json:"-"`
	Monto     int64     `gorm:"not null" json:"monto"`
	Estado    string    `gorm:"not null;default:pendiente" json:"estado"`
	Aciertos  int       `gorm:"not null;default:0" json:"aciertos"`
	Premio    int64     `gorm:"not null;default:0" json:"premio"`
	CreatedAt time.Time `json:"created_at"`

	Sorteo Sorteo `gorm:"foreignKey:SorteoID" json:"sorteo,omitempty"`
}

// Movimiento registra cada cambio de saldo de un usuario.
type Movimiento struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	Tipo        string    `gorm:"not null" json:"tipo"`
	Monto       int64     `gorm:"not null" json:"monto"`
	Descripcion string    `json:"descripcion"`
	CreatedAt   time.Time `json:"created_at"`
}
