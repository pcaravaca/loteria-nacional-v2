package database

import (
	"testing"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
)

// TestSeedCasinoDobleRTP verifica la declaracion de doble RTP: los juegos con
// decisiones (domino, ajedrez, backgammon, tetris) declaran el RTP habil y el
// RTP casual medido jugando al azar, y el habil queda dentro de la banda del
// proyecto (78-92%). Los juegos de azar puro quedan con rtp_casual = 0
// ("no aplica") y se muestran con un solo numero.
func TestSeedCasinoDobleRTP(t *testing.T) {
	db, err := Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := SeedCasino(db); err != nil {
		t.Fatalf("seed casino: %v", err)
	}

	casuales := map[string]float64{
		models.CasinoDomino:     79.08,
		models.CasinoAjedrez:    86.59,
		models.CasinoBackgammon: 19.17,
		models.CasinoTetris:     4.51,
	}
	for codigo, esperado := range casuales {
		var juego models.CasinoGame
		if err := db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
			t.Fatalf("%s: %v", codigo, err)
		}
		if juego.RTPCasual <= 0 {
			t.Errorf("%s: rtp_casual = %.2f, debe estar declarado", codigo, juego.RTPCasual)
		}
		if juego.RTPCasual != esperado {
			t.Errorf("%s: rtp_casual = %.2f, esperaba %.2f", codigo, juego.RTPCasual, esperado)
		}
		if juego.RTP < 78 || juego.RTP > 92 {
			t.Errorf("%s: RTP habil = %.2f, fuera de la banda 78-92", codigo, juego.RTP)
		}
	}

	// Las tres mesas de pinball son de habilidad: declaran los dos numeros
	// (el hábil es el campo rtp). Sus cifras salen del mismo motor que paga
	// (casino.PinballRTPHabil/Casual), no de una estimacion.
	for _, codigo := range []string{
		models.CasinoPinballNeon, models.CasinoPinballClasico, models.CasinoPlinkoPremium,
	} {
		var juego models.CasinoGame
		if err := db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
			t.Fatalf("%s: %v", codigo, err)
		}
		if juego.RTPCasual <= 0 {
			t.Errorf("%s: rtp_casual = %.2f, debe estar declarado", codigo, juego.RTPCasual)
		}
		if juego.RTP != casino.PinballRTPHabil() || juego.RTPCasual != casino.PinballRTPCasual() {
			t.Errorf("%s: declara %.2f/%.2f, el motor mide %.2f/%.2f",
				codigo, juego.RTP, juego.RTPCasual,
				casino.PinballRTPHabil(), casino.PinballRTPCasual())
		}
	}

	// Azar puro: un solo RTP, rtp_casual = 0.
	for _, codigo := range []string{
		models.CasinoDados, models.CasinoSlots, models.CasinoRuleta,
		models.CasinoRaspa, models.CasinoKeno, models.CasinoBingo,
		models.CasinoMinaDeOro,
	} {
		var juego models.CasinoGame
		if err := db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
			t.Fatalf("%s: %v", codigo, err)
		}
		if juego.RTPCasual != 0 {
			t.Errorf("%s: rtp_casual = %.2f, deberia ser 0 (no aplica)", codigo, juego.RTPCasual)
		}
	}
}
