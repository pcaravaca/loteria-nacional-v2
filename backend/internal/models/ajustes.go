package models

import "time"

// Ajuste es una configuración clave-valor de la plataforma. Hoy guarda la marca
// (nombre, logo e icono) para que el mismo producto pueda venderse con el nombre
// del negocio del cliente.
type Ajuste struct {
	Clave     string    `gorm:"primaryKey;size:64" json:"clave"`
	Valor     string    `gorm:"type:text" json:"valor"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Ajuste) TableName() string { return "ajustes" }

// Claves de marca conocidas. El endpoint publico solo expone estas.
const (
	AjusteBrandNombre = "brand_nombre"
	AjusteBrandLogo   = "brand_logo"
	AjusteBrandIcono  = "brand_icono"
)

var ClavesMarca = []string{AjusteBrandNombre, AjusteBrandLogo, AjusteBrandIcono}
