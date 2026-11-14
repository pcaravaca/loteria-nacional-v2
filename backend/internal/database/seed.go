package database

import (
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

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
		{Codigo: "CH", Nombre: "Chaves", Tipo: models.TipoLoteria, Descripcion: "El juego de los 5 de la suerte.", CostoNumero: 800, LongitudNum: 2, MaxNumeros: 5, Multiplicador: 45, Activa: true},
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
