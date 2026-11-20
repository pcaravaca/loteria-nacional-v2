package resultados

import (
	"context"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// Importador cruza los sorteos abiertos de la base con los resultados que
// publican las fuentes y, cuando hay coincidencia, graba el resultado y paga.
type Importador struct {
	db         *gorm.DB
	proveedores []Proveedor
	logger     *log.Logger
	zona       *time.Location
	// Ventana hacia atrás: solo se intenta importar sorteos de los últimos N
	// días, para no estar consultando fechas viejas todo el tiempo.
	ventanaDias int
}

func NuevoImportador(db *gorm.DB, proveedores []Proveedor, logger *log.Logger) *Importador {
	if logger == nil {
		logger = log.Default()
	}
	zona, err := time.LoadLocation(zonaCR)
	if err != nil {
		zona = time.FixedZone("CR", -6*3600)
	}
	return &Importador{db: db, proveedores: proveedores, logger: logger, zona: zona, ventanaDias: 7}
}

type Resumen struct {
	Loteria    string   `json:"loteria"`
	Fuente     string   `json:"fuente"`
	Encontrados int     `json:"encontrados"`
	Aplicados  int      `json:"aplicados"`
	Omitidos   int      `json:"omitidos"`
	Errores    []string `json:"errores,omitempty"`
}

func (i *Importador) proveedorPara(juego string) Proveedor {
	for _, p := range i.proveedores {
		if p.Soporta(juego) {
			return p
		}
	}
	return nil
}

// ImportarLoteria importa los resultados de una lotería para todos sus sorteos
// abiertos dentro de la ventana.
func (i *Importador) ImportarLoteria(ctx context.Context, loteria models.Loteria) Resumen {
	res := Resumen{Loteria: loteria.Codigo}
	p := i.proveedorPara(loteria.Codigo)
	if p == nil {
		return res
	}
	res.Fuente = p.Nombre()

	var sorteos []models.Sorteo
	corte := time.Now().In(i.zona).AddDate(0, 0, -i.ventanaDias)
	if err := i.db.Preload("Loteria").
		Where("loteria_id = ? AND estado = ? AND fecha >= ?", loteria.ID, models.SorteoAbierto, corte).
		Order("fecha asc").Find(&sorteos).Error; err != nil {
		res.Errores = append(res.Errores, err.Error())
		return res
	}

	for _, s := range sorteos {
		rs, err := p.Obtener(ctx, loteria.Codigo, s.Fecha.In(i.zona))
		if err != nil {
			res.Errores = append(res.Errores, err.Error())
			continue
		}
		res.Encontrados += len(rs)
		for _, r := range rs {
			if !mismoTurno(s.Titulo, r.Turno) {
				res.Omitidos++
				continue
			}
			if err := GrabarSorteo(i.db, &s, r.Numeros); err != nil {
				res.Errores = append(res.Errores, err.Error())
				continue
			}
			res.Aplicados++
			i.logger.Printf("resultado importado: %s sorteo #%s (%d numeros) fuente=%s",
				loteria.Codigo, r.SorteoNum, len(r.Numeros), r.Fuente)
		}
	}
	return res
}

// ImportarPendientes recorre las loterías activas que alguna fuente cubre.
func (i *Importador) ImportarPendientes(ctx context.Context) []Resumen {
	var loterias []models.Loteria
	if err := i.db.Where("activa = ?", true).Find(&loterias).Error; err != nil {
		return nil
	}
	out := make([]Resumen, 0, len(loterias))
	for _, l := range loterias {
		if i.proveedorPara(l.Codigo) == nil {
			continue
		}
		out = append(out, i.ImportarLoteria(ctx, l))
	}
	return out
}

// mismoTurno compara el turno del sorteo con el del resultado. Si el resultado
// no tiene turno (LN, Chances, Lotto), vale para cualquier sorteo del día.
func mismoTurno(titulo, turno string) bool {
	if turno == "" {
		return true
	}
	t := NormalizaTurno(titulo)
	switch turno {
	case TurnoMedioDia:
		return strings.Contains(t, "medio")
	case TurnoTarde:
		return strings.Contains(t, "tarde")
	case TurnoNoche:
		return strings.Contains(t, "noche")
	}
	return false
}
