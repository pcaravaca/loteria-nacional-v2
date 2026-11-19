package models

import "encoding/json"

// TablaPremiosPorAciertos devuelve los multiplicadores declarados por la loteria,
// indexados por cantidad de aciertos (p.ej. {"1":3,"2":10}).
func (l Loteria) TablaPremiosPorAciertos() map[int]int64 {
	out := map[int]int64{}
	if l.TablaPremios == "" {
		return out
	}
	_ = json.Unmarshal([]byte(l.TablaPremios), &out)
	return out
}

// Premio calcula el premio de una apuesta segun sus aciertos:
//   - acierto total            -> monto x multiplicador de la loteria;
//   - acierto parcial con tabla -> monto x multiplicador de la tabla;
//   - acierto parcial sin tabla -> monto x 2 (regla historica).
//
// Devuelve 0 si no hubo aciertos.
func (l Loteria) Premio(monto int64, aciertos, numerosJugados int) int64 {
	if aciertos <= 0 || numerosJugados <= 0 {
		return 0
	}
	if aciertos >= numerosJugados {
		mult := l.Multiplicador
		if mult <= 0 {
			mult = 50
		}
		return monto * mult
	}
	if v, ok := l.TablaPremiosPorAciertos()[aciertos]; ok && v > 0 {
		return monto * v
	}
	return monto * 2
}
