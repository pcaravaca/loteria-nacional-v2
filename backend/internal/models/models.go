package models

import (
	"time"

	"gorm.io/gorm"
)

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
	SorteoAbierto = "abierto"
	SorteoCerrado = "cerrado"
	SorteoPagado  = "pagado"
	// SorteoVencido: paso la fecha con CIERRE_AUTOMATICO activo; se reembolsan las apuestas.
	SorteoVencido = "vencido"
)

// Estado de una apuesta.
const (
	ApuestaPendiente = "pendiente"
	ApuestaGanadora  = "ganadora"
	ApuestaPerdedora = "perdedora"
	ApuestaPagada    = "pagada"
	ApuestaCancelada = "cancelada"
)

// Movimiento de saldo.
const (
	MovRecarga         = "recarga"
	MovApuesta         = "apuesta"
	MovPremio          = "premio"
	MovBono            = "bono"
	MovRetiro          = "retiro"
	MovAjuste          = "ajuste"
	MovCasinoApuesta   = "casino_apuesta"
	MovCasinoPremio    = "casino_premio"
	MovRifaCompra      = "rifa_compra"
	MovRifaPremio      = "rifa_premio"
	MovRifaReembolso   = "rifa_reembolso"
	MovSorteoReembolso = "sorteo_reembolso"
)

// Tipos de documento aceptados para la identificacion (KYC basico).
const (
	DocCedulaFisica    = "CEDULA_FISICA"
	DocCedulaJuridica  = "CEDULA_JURIDICA"
	DocDIMEX           = "DIMEX"
	DocPasaporte       = "PASAPORTE"
	DocConsumidorFinal = "CONSUMIDOR_FINAL"
)

// AutoexclusionMinimaMeses es el plazo minimo e irreversible de autoexclusion.
const AutoexclusionMinimaMeses = 6

// Nombres de tabla explícitos (en español, plural) para no depender de la
// pluralización automática de GORM.
func (User) TableName() string       { return "usuarios" }
func (Loteria) TableName() string    { return "loterias" }
func (Sorteo) TableName() string     { return "sorteos" }
func (Apuesta) TableName() string    { return "apuestas" }
func (Movimiento) TableName() string { return "movimientos" }

// Ambientes visuales del casino. El orden es el del selector en la web.
const (
	TemaClasico    = "clasico"
	TemaNeon       = "neon"
	TemaPremium    = "premium"
	TemaTropical   = "tropical"
	TemaMedianoche = "medianoche"
	TemaRubi       = "rubi"
)

var temasCasino = []string{
	TemaClasico, TemaNeon, TemaPremium, TemaTropical, TemaMedianoche, TemaRubi,
}

// Temas de la aplicacion completa (no solo el casino).
const (
	TemaAppEsmeralda = "esmeralda"
	TemaAppZafiro    = "zafiro"
	TemaAppRubi      = "rubi"
	TemaAppAmatista  = "amatista"
	TemaAppGrafito   = "grafito"
)

var temasApp = []string{
	TemaAppEsmeralda, TemaAppZafiro, TemaAppRubi, TemaAppAmatista, TemaAppGrafito,
}

func TemaAppValido(tema string) bool {
	for _, t := range temasApp {
		if t == tema {
			return true
		}
	}
	return false
}

func TemasApp() []string { return append([]string(nil), temasApp...) }

func TemaCasinoValido(tema string) bool {
	for _, t := range temasCasino {
		if t == tema {
			return true
		}
	}
	return false
}

// TemasCasino devuelve los ambientes disponibles (para el catalogo de la web).
func TemasCasino() []string { return append([]string(nil), temasCasino...) }

// User es un jugador, agente o administrador.
type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Nombre       string `gorm:"not null" json:"nombre"`
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"column:password_hash" json:"-"`
	Rol          string `gorm:"not null;default:cliente" json:"rol"`
	Telefono     string `json:"telefono"`
	Saldo        int64  `gorm:"not null;default:0" json:"saldo"`
	Bono         int64  `gorm:"not null;default:0" json:"bono"`
	// Semilla del casino (provably fair): el hash se publica antes de jugar y
	// la semilla se revela al verificar la ronda.
	// Ambiente visual del casino elegido por el usuario (clasico, neon, premium,
	// tropical, medianoche, rubi). Default: premium.
	TemaCasino string `gorm:"size:32;not null;default:'premium'" json:"tema_casino"`
	// Tema visual de toda la aplicacion (esmeralda, zafiro, rubi, amatista, grafito).
	TemaApp     string `gorm:"size:32;not null;default:'esmeralda'" json:"tema_app"`
	CasinoSeed  string `gorm:"size:64" json:"-"`
	CasinoNonce int64  `json:"-"`
	// Juego responsable: autoexclusion voluntaria. Es NULL si no aplica y, si
	// tiene fecha, las jugadas quedan bloqueadas hasta esa fecha (sin reversa).
	AutoexcluidoHasta *time.Time `json:"autoexcluido_hasta"`
	// Identificacion (KYC basico): tipo y numero de documento, y verificacion.
	TipoDocumento       string     `gorm:"size:32" json:"tipo_documento"`
	NumeroDocumento     string     `gorm:"size:64" json:"numero_documento"`
	DocumentoVerificado bool       `gorm:"not null;default:false" json:"documento_verificado"`
	FechaNacimiento     *time.Time `json:"fecha_nacimiento"`
	// FotoDocumento es la ruta del archivo subido; solo se expone al admin.
	FotoDocumento string    `gorm:"size:255" json:"-"`
	TieneFoto     bool      `gorm:"-" json:"tiene_foto"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AfterFind completa las banderas derivadas que no viven en la base.
func (u *User) AfterFind(tx *gorm.DB) error {
	u.TieneFoto = u.FotoDocumento != ""
	return nil
}

// Loteria es un juego (Lotería Nacional, Quiniela, etc.).
type Loteria struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Codigo        string `gorm:"uniqueIndex;not null" json:"codigo"`
	Nombre        string `gorm:"not null" json:"nombre"`
	Tipo          string `gorm:"not null;default:loteria" json:"tipo"`
	Descripcion   string `json:"descripcion"`
	CostoNumero   int64  `gorm:"not null;default:1000" json:"costo_numero"`
	LongitudNum   int    `gorm:"not null;default:2" json:"longitud_num"`
	MaxNumeros    int    `gorm:"not null;default:1" json:"max_numeros"`
	Multiplicador int64  `gorm:"not null;default:50" json:"multiplicador"`
	// TablaPremios es JSON {"1":3,"2":10,...}: multiplicador por aciertos parciales.
	// Sin entrada para un acierto cae al 2x historico.
	TablaPremios string    `gorm:"column:tabla_premios" json:"tabla_premios"`
	Activa       bool      `gorm:"not null;default:true" json:"activa"`
	CreatedAt    time.Time `json:"created_at"`
}

// Sorteo es un concurso con fecha y, al cerrarse, números ganadores.
type Sorteo struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	LoteriaID        uint      `gorm:"not null;index" json:"loteria_id"`
	Titulo           string    `gorm:"not null" json:"titulo"`
	Fecha            time.Time `gorm:"not null;index" json:"fecha"`
	Estado           string    `gorm:"not null;default:abierto" json:"estado"`
	NumerosGanadores []int     `gorm:"-" json:"numeros_ganadores"`
	NumerosRaw       string    `gorm:"column:numeros_raw" json:"-"`
	CreatedAt        time.Time `json:"created_at"`

	Loteria Loteria `gorm:"foreignKey:LoteriaID" json:"loteria,omitempty"`
}

// Apuesta es un ticket de juego de un usuario para un sorteo.
type Apuesta struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	SorteoID   uint      `gorm:"not null;index" json:"sorteo_id"`
	Numeros    []int     `gorm:"-" json:"numeros"`
	NumerosRaw string    `gorm:"column:numeros_raw" json:"-"`
	Monto      int64     `gorm:"not null" json:"monto"`
	Estado     string    `gorm:"not null;default:pendiente" json:"estado"`
	Aciertos   int       `gorm:"not null;default:0" json:"aciertos"`
	Premio     int64     `gorm:"not null;default:0" json:"premio"`
	CreatedAt  time.Time `json:"created_at"`

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

// AuditLog deja constancia inmutable de las acciones relevantes (admin, KYC,
// limites, configuracion de juegos) para la fiscalizacion.
type AuditLog struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ActorID   uint   `gorm:"index" json:"actor_id"`
	ActorName string `gorm:"size:120" json:"actor_nombre"`
	Accion    string `gorm:"not null;size:64;index" json:"accion"`
	Entidad   string `gorm:"size:64;index" json:"entidad"`
	EntidadID string `gorm:"size:64" json:"entidad_id"`
	Detalle   string `gorm:"type:text" json:"detalle"`
	IP        string `gorm:"size:64" json:"ip"`
	// RequestID correlaciona la accion con la peticion HTTP exacta (lo pone el
	// middleware.RequestID y el cliente lo reenvia si reintenta).
	RequestID string    `gorm:"size:40;index" json:"request_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuditLog) TableName() string { return "auditoria" }

// LimitesJugador son los limites voluntarios de deposito y gasto que el
// jugador se autoimpone (juego responsable). 0 = sin limite.
type LimitesJugador struct {
	UserID          uint      `gorm:"primaryKey" json:"user_id"`
	DepositoDiario  int64     `gorm:"not null;default:0" json:"deposito_diario"`
	DepositoSemanal int64     `gorm:"not null;default:0" json:"deposito_semanal"`
	DepositoMensual int64     `gorm:"not null;default:0" json:"deposito_mensual"`
	GastoDiario     int64     `gorm:"not null;default:0" json:"gasto_diario"`
	GastoSemanal    int64     `gorm:"not null;default:0" json:"gasto_semanal"`
	GastoMensual    int64     `gorm:"not null;default:0" json:"gasto_mensual"`
	ActualizadoEn   time.Time `json:"actualizado_en"`
}

func (LimitesJugador) TableName() string { return "limites_jugador" }
