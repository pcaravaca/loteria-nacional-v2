package models

import (
	"encoding/json"

	"gorm.io/gorm"
)

// Los números se guardan en SQLite como JSON en texto (NumerosRaw) y se
// exponen como []int en la API.

func (s *Sorteo) BeforeSave(tx *gorm.DB) error {
	if s.NumerosGanadores == nil {
		s.NumerosGanadores = []int{}
	}
	b, err := json.Marshal(s.NumerosGanadores)
	s.NumerosRaw = string(b)
	return err
}

func (s *Sorteo) AfterFind(tx *gorm.DB) error {
	if s.NumerosRaw == "" {
		s.NumerosGanadores = []int{}
		return nil
	}
	return json.Unmarshal([]byte(s.NumerosRaw), &s.NumerosGanadores)
}

func (a *Apuesta) BeforeSave(tx *gorm.DB) error {
	if a.Numeros == nil {
		a.Numeros = []int{}
	}
	b, err := json.Marshal(a.Numeros)
	a.NumerosRaw = string(b)
	return err
}

func (a *Apuesta) AfterFind(tx *gorm.DB) error {
	if a.NumerosRaw == "" {
		a.Numeros = []int{}
		return nil
	}
	return json.Unmarshal([]byte(a.NumerosRaw), &a.Numeros)
}
