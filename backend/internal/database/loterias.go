package database

import (
	"errors"

	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// catalogoLoterias es el catálogo de la casa. Se usa para el seed de una base
// nueva y también para asegurar que una instalación existente tenga todas las
// loterías (las nuevas se agregan apagadas: nadie vende un juego cuyo cuadro de
// pagos todavía no se decidió).
var catalogoLoterias = []models.Loteria{
	{Codigo: "LN", Nombre: "Lotería Nacional", Tipo: models.TipoLoteria, Descripcion: "El sorteo clásico de Costa Rica.", CostoNumero: 1000, LongitudNum: 2, MaxNumeros: 3, Multiplicador: 50, Activa: true},
	{Codigo: "LP", Nombre: "Lotería Popular", Tipo: models.TipoLoteria, Descripcion: "Sorteos diarios de Lotería Popular.", CostoNumero: 500, LongitudNum: 2, MaxNumeros: 3, Multiplicador: 40, Activa: true},
	{Codigo: "QN", Nombre: "Quiniela Nacional", Tipo: models.TipoQuiniela, Descripcion: "Acierta el número ganador de la quiniela.", CostoNumero: 1000, LongitudNum: 2, MaxNumeros: 2, Multiplicador: 35, Activa: true},
	{Codigo: "QD", Nombre: "Quiniela Deportiva", Tipo: models.TipoQuiniela, Descripcion: "Pronostica los resultados de la jornada.", CostoNumero: 2000, LongitudNum: 3, MaxNumeros: 3, Multiplicador: 60, Activa: true},
	{Codigo: "CA", Nombre: "Chances", Tipo: models.TipoLoteria, Descripcion: "El popular sorteo de Chances.", CostoNumero: 800, LongitudNum: 2, MaxNumeros: 5, Multiplicador: 45, Activa: true},
	// Nuevos Tiempos (JPS): número del 00 al 99, tres sorteos diarios. El
	// multiplicador reproduce el pago clasico de 75 veces lo apostado.
	{Codigo: "NT", Nombre: "Nuevos Tiempos", Tipo: models.TipoLoteria, Descripcion: "Elegí un número del 00 al 99. Tres sorteos diarios.", CostoNumero: 500, LongitudNum: 2, MaxNumeros: 1, Multiplicador: 75, Activa: true},
	// Juegos con fuente de resultados ya parseable pero todavía sin cuadro de
	// pagos aprobado: entran apagados y se encienden cuando se decida el pago.
	{Codigo: "LOTTO", Nombre: "Lotto", Tipo: models.TipoLoteria, Descripcion: "Cinco números del 00 al 40.", CostoNumero: 1000, LongitudNum: 2, MaxNumeros: 5, Multiplicador: 1000, Activa: false},
	{Codigo: "REVANCHA", Nombre: "Lotto Revancha", Tipo: models.TipoLoteria, Descripcion: "La revancha del Lotto, cinco números del 00 al 40.", CostoNumero: 1000, LongitudNum: 2, MaxNumeros: 5, Multiplicador: 1000, Activa: false},
	{Codigo: "MONAZOS", Nombre: "3 Monazos", Tipo: models.TipoLoteria, Descripcion: "Tres cifras por turno, tres sorteos diarios.", CostoNumero: 500, LongitudNum: 3, MaxNumeros: 1, Multiplicador: 500, Activa: false},
}

// AsegurarLoterias crea las loterías que falten sin tocar las existentes.
func AsegurarLoterias(db *gorm.DB) error {
	for i := range catalogoLoterias {
		l := catalogoLoterias[i]
		var existente models.Loteria
		err := db.Where("codigo = ?", l.Codigo).First(&existente).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Select("*") para que los booleanos en false también se escriban:
			// el default de `activa` es true y GORM omitiría el cero.
			if err := db.Select("*").Create(&l).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}
