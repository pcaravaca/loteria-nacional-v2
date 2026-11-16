package database

import (
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Seed pobla la base con datos de demostración la primera vez (sin usuarios).
func Seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	log.Println("seed: creando datos de demostración")

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	agenteHash, _ := bcrypt.GenerateFromPassword([]byte("agente123"), bcrypt.DefaultCost)
	clienteHash, _ := bcrypt.GenerateFromPassword([]byte("cliente123"), bcrypt.DefaultCost)

	admin := models.User{Nombre: "Administrador", Email: "admin@loteria.test", PasswordHash: string(hash), Rol: models.RolAdmin}
	agente := models.User{Nombre: "Agente Aserrí", Email: "agente@loteria.test", PasswordHash: string(agenteHash), Rol: models.RolAgente, Saldo: 0}
	cliente := models.User{Nombre: "Cliente Demo", Email: "cliente@loteria.test", PasswordHash: string(clienteHash), Rol: models.RolCliente, Saldo: 25000, Bono: 1500}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}
	if err := db.Create(&agente).Error; err != nil {
		return err
	}
	if err := db.Create(&cliente).Error; err != nil {
		return err
	}

	loterias := []models.Loteria{
		{Codigo: "LN", Nombre: "Lotería Nacional", Tipo: models.TipoLoteria, Descripcion: "El sorteo clásico de Costa Rica.", CostoNumero: 1000, LongitudNum: 2, MaxNumeros: 3, Multiplicador: 50, Activa: true},
		{Codigo: "LP", Nombre: "Lotería Popular", Tipo: models.TipoLoteria, Descripcion: "Sorteos diarios de Lotería Popular.", CostoNumero: 500, LongitudNum: 2, MaxNumeros: 3, Multiplicador: 40, Activa: true},
		{Codigo: "QN", Nombre: "Quiniela Nacional", Tipo: models.TipoQuiniela, Descripcion: "Acierta el número ganador de la quiniela.", CostoNumero: 1000, LongitudNum: 2, MaxNumeros: 2, Multiplicador: 35, Activa: true},
		{Codigo: "QD", Nombre: "Quiniela Deportiva", Tipo: models.TipoQuiniela, Descripcion: "Pronostica los resultados de la jornada.", CostoNumero: 2000, LongitudNum: 3, MaxNumeros: 3, Multiplicador: 60, Activa: true},
		{Codigo: "CA", Nombre: "Chances", Tipo: models.TipoLoteria, Descripcion: "El popular sorteo de Chances.", CostoNumero: 800, LongitudNum: 2, MaxNumeros: 5, Multiplicador: 45, Activa: true},
		// Nuevos Tiempos (JPS): número del 00 al 99, tres sorteos diarios. El
		// multiplicador reproduce el pago clasico de 75 veces lo apostado.
		{Codigo: "NT", Nombre: "Nuevos Tiempos", Tipo: models.TipoLoteria, Descripcion: "Elegí un número del 00 al 99. Tres sorteos diarios.", CostoNumero: 500, LongitudNum: 2, MaxNumeros: 1, Multiplicador: 75, Activa: true},
	}
	for i := range loterias {
		if err := db.Create(&loterias[i]).Error; err != nil {
			return err
		}
	}

	abierto := models.Sorteo{
		LoteriaID: loterias[0].ID,
		Titulo:    "Lotería Nacional · Sorteo de la suerte",
		Fecha:     time.Now().Add(48 * time.Hour),
		Estado:    models.SorteoAbierto,
	}
	if err := db.Create(&abierto).Error; err != nil {
		return err
	}

	cerrado := models.Sorteo{
		LoteriaID:        loterias[0].ID,
		Titulo:           "Lotería Nacional · Sorteo anterior",
		Fecha:            time.Now().Add(-72 * time.Hour),
		Estado:           models.SorteoPagado,
		NumerosGanadores: []int{7, 21, 42},
	}
	if err := db.Create(&cerrado).Error; err != nil {
		return err
	}

	apuesta := models.Apuesta{
		UserID:  cliente.ID,
		SorteoID: cerrado.ID,
		Numeros:  []int{21, 42, 88},
		Monto:    1000,
		Estado:   models.ApuestaGanadora,
		Aciertos: 2,
		Premio:   5000,
	}
	if err := db.Create(&apuesta).Error; err != nil {
		return err
	}

	movs := []models.Movimiento{
		{UserID: cliente.ID, Tipo: models.MovRecarga, Monto: 30000, Descripcion: "Recarga inicial"},
		{UserID: cliente.ID, Tipo: models.MovPremio, Monto: 5000, Descripcion: "Premio Lotería Nacional"},
	}
	if err := db.Create(&movs).Error; err != nil {
		return err
	}

	log.Println("seed: admin@loteria.test/admin123 · agente@loteria.test/agente123 · cliente@loteria.test/cliente123")

	return nil
}

// SeedCasino crea los juegos del mini casino si faltan. Es idempotente y se
// puede llamar en cada arranque (las tablas de casino pueden no existir en
// bases creadas antes de esta version).
//
// MaxPremio es el techo absoluto por jugada (0 = sin techo). El valor por
// defecto es una politica propuesta para una instalacion nueva; como se vende
// la plataforma a clientes que operan con sus propios permisos, cada operador
// lo ajusta desde el panel de administracion. En bases ya existentes el
// seed no toca las filas, asi que el techo queda en 0 hasta que se configure.
func SeedCasino(db *gorm.DB) error {
	juegos := []models.CasinoGame{
		{Codigo: models.CasinoDados, Nombre: "Dados del Sur", Descripcion: "Acerta la suma de dos dados.", MinApuesta: 200, MaxApuesta: 50000, RTP: 88.0, MaxMultipl: 30, MaxPremio: 1000000, Orden: 1, LimiteDiario: 200000, Activo: true},
		{Codigo: models.CasinoSlots, Nombre: "Fortuna CR", Descripcion: "Tragamonedas de 3 rodillos y 6 simbolos.", MinApuesta: 500, MaxApuesta: 50000, RTP: 94.7, MaxMultipl: 50, MaxPremio: 1000000, Orden: 2, LimiteDiario: 200000, Activo: true},
		{Codigo: models.CasinoRuleta, Nombre: "Rueda Fortuna", Descripcion: "Ruleta de 37 casillas: pleno o color.", MinApuesta: 500, MaxApuesta: 50000, RTP: 94.4, MaxMultipl: 35, MaxPremio: 1000000, Orden: 3, LimiteDiario: 200000, Activo: true},
		{Codigo: models.CasinoRaspa, Nombre: "Raspa y Gana", Descripcion: "Tres casillas: si coinciden, ganaste.", MinApuesta: 200, MaxApuesta: 50000, RTP: 92.1, MaxMultipl: 18, MaxPremio: 1000000, Orden: 4, LimiteDiario: 200000, Activo: true},
		{Codigo: models.CasinoKeno, Nombre: "Keno CR", Descripcion: "Elegi 5 numeros y salen 15.", MinApuesta: 500, MaxApuesta: 50000, RTP: 92.5, MaxMultipl: 65, MaxPremio: 1000000, Orden: 5, LimiteDiario: 200000, Activo: true},
		{Codigo: models.CasinoBlackjack, Nombre: "21 Blackjack", Descripcion: "Pedir, plantarse o doblar contra la casa.", MinApuesta: 500, MaxApuesta: 50000, RTP: 99.4, MaxMultipl: 3, MaxPremio: 1000000, Orden: 6, LimiteDiario: 200000, Activo: true},
	{Codigo: models.CasinoCascada, Nombre: "Cascada Fortuna", Descripcion: "La bola rebota por los pinchos hasta una casilla con premio.", MinApuesta: 200, MaxApuesta: 50000, RTP: 96.5, MaxMultipl: 10, MaxPremio: 1000000, Orden: 7, LimiteDiario: 200000, Activo: true},
	{Codigo: models.CasinoHilo, Nombre: "Mayor o Menor", Descripcion: "Adivina si la proxima carta es mas alta o mas baja. Si empata, perdes.", MinApuesta: 200, MaxApuesta: 50000, RTP: 92.3, MaxMultipl: 2, MaxPremio: 1000000, Orden: 8, LimiteDiario: 200000, Activo: true},
	{Codigo: models.CasinoAlto, Nombre: "Bajo o Sobre", Descripcion: "Elegi el tope y acerta si el dado del 1 al 100 cae abajo o arriba.", MinApuesta: 200, MaxApuesta: 50000, RTP: 95.0, MaxMultipl: 48, MaxPremio: 1000000, Orden: 9, LimiteDiario: 200000, Activo: true},
	{Codigo: models.CasinoFortuna, Nombre: "Fortuna Rápida", Descripcion: "La aguja gira hasta detenerse en un multiplicador.", MinApuesta: 200, MaxApuesta: 50000, RTP: 95.0, MaxMultipl: 5, MaxPremio: 1000000, Orden: 10, LimiteDiario: 200000, Activo: true},
	{Codigo: models.CasinoMoneda, Nombre: "Cara o Sello", Descripcion: "Elegí el lado de la moneda: acertarte duplica.", MinApuesta: 200, MaxApuesta: 50000, RTP: 99.0, MaxMultipl: 2, MaxPremio: 1000000, Orden: 11, LimiteDiario: 200000, Activo: true},
	{Codigo: models.CasinoSuertudo, Nombre: "Número de la Suerte", Descripcion: "Elegí un número del 00 al 99: si sale, pagás x75.", MinApuesta: 200, MaxApuesta: 50000, RTP: 75.0, MaxMultipl: 75, MaxPremio: 1000000, Orden: 12, LimiteDiario: 200000, Activo: true},
	{Codigo: models.CasinoParley, Nombre: "Parley de 2", Descripcion: "Marcá 2 números de 20 y ganá si salen en las 4 bolillas.", MinApuesta: 200, MaxApuesta: 50000, RTP: 91.0, MaxMultipl: 18, MaxPremio: 1000000, Orden: 13, LimiteDiario: 200000, Activo: true},
	{Codigo: models.CasinoBaccarat, Nombre: "Baccarat", Descripcion: "Jugador, banca o empate: el empate paga x8 y la banca cobra 5% de comisión.", MinApuesta: 500, MaxApuesta: 50000, RTP: 98.7, MaxMultipl: 8, MaxPremio: 1000000, Orden: 14, LimiteDiario: 200000, Activo: true},
	}
		// Catalogo AST Casino: tragamonedas y video poker. Sin techo de premio
		// (MaxPremio 0) porque la volatilidad de las lineas hace que el techo recorte
		// demasiado la apuesta maxima; los limites diarios siguen aplicando.
		orden := 15
		for _, spec := range casino.SpecsAST() {
			juegos = append(juegos, models.CasinoGame{
				Codigo:      spec.Codigo,
				Nombre:      spec.Nombre,
				Descripcion: spec.Descripcion,
				MinApuesta:  500,
				MaxApuesta:  50000,
				RTP:         spec.RTP,
				MaxMultipl:  spec.MaxMultipl,
				MaxPremio:   0,
				Orden:       orden,
				LimiteDiario: 200000,
				Activo:      true,
			})
			orden++
		}
	for i := range juegos {
		var existente models.CasinoGame
		if err := db.Where("codigo = ?", juegos[i].Codigo).First(&existente).Error; err == nil {
			continue
		}
		// Select("*") para que GORM no omita los campos con valor cero que
		// llevan `default` en el tag (activo:false se ignoraria si no).
		// GORM rellena los campos con `default` en el propio struct durante el
		// Create (mutando Activo a true), asi que se guarda el valor a mano.
		activo := juegos[i].Activo
		if err := db.Select("*").Create(&juegos[i]).Error; err != nil {
			return err
		}
		if err := db.Exec("UPDATE casino_juegos SET activo = ? WHERE codigo = ?", activo, juegos[i].Codigo).Error; err != nil {
			return err
		}
	}
	return nil
}

// SeedRifas crea una rifa de demostracion si no hay ninguna.
func SeedRifas(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Rifa{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	rifa := models.Rifa{
		Titulo:       "Rifa Navideña · Moto Pulsar 200",
		Descripcion:  "Rifa de demostración: 100 números del 00 al 99 y sorteo en vivo.",
		Premio:       "Moto Pulsar 200NS 0km",
		PrecioNumero: 5000,
		DesdeNumero:  0,
		HastaNumero:  99,
		FechaSorteo:  time.Now().Add(168 * time.Hour),
		Estado:       models.RifaAbierta,
		TipoSorteo:   models.RifaSorteoAzar,
	}
	if err := db.Create(&rifa).Error; err != nil {
		return err
	}
	numeros := make([]models.RifaNumero, 0, 100)
	for n := rifa.DesdeNumero; n <= rifa.HastaNumero; n++ {
		numeros = append(numeros, models.RifaNumero{RifaID: rifa.ID, Numero: n})
	}
	return db.CreateInBatches(numeros, 100).Error
}
