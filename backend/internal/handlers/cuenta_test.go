package handlers

import (
	"testing"
	"time"
)

func TestAutoexclusionActiva(t *testing.T) {
	if AutoexclusionActiva(nil) {
		t.Error("sin fecha no deberia estar activa")
	}
	pasado := time.Now().Add(-time.Hour)
	if AutoexclusionActiva(&pasado) {
		t.Error("una fecha pasada ya no esta activa")
	}
	futuro := time.Now().Add(24 * time.Hour)
	if !AutoexclusionActiva(&futuro) {
		t.Error("una fecha futura deberia estar activa")
	}
}
