package casino

import "testing"

// manoPoker arma una carta a partir de rango y palo (palo 0-3).
func manoPoker(rango, palo int) cartaPoker {
	return cartaPoker{Rango: rango, Palo: palo}
}

// TestOptimizarRetencionParBajo fija el comportamiento medido: en las mesas sin
// comodin, un par bajo (22-TT) se sostiene, porque sostenerlo rinde mas que
// tirar las cinco cartas. Antes el sistema lo descartaba y eso costaba entre 4
// y 5 puntos de RTP.
func TestOptimizarRetencionParBajo(t *testing.T) {
	cfg, ok := BuscarPoker("jacksorbetter")
	if !ok {
		t.Fatal("no se encontro jacksorbetter")
	}
	// 5-5-9-2-J de distinto palo: un par bajo sin nada mas (el J no arma
	// escalera con el par).
	mano := []cartaPoker{
		manoPoker(5, 0), manoPoker(5, 1), manoPoker(9, 2), manoPoker(2, 3), manoPoker(11, 0),
	}
	retener := optimizarRetencion(cfg, mano)
	if !retener[0] || !retener[1] {
		t.Errorf("par bajo: el sistema lo descarta (retener=%v)", retener)
	}
	for i := 2; i < 5; i++ {
		if retener[i] {
			t.Errorf("par bajo: sostiene de mas la carta %d", i)
		}
	}
}

// TestOptimizarRetencionDeuceWildSueltaElPar: en Deuce Wild los dos son
// comodin, y sostener el par bajo rinde 58,3% contra 65,2% soltandolo.
func TestOptimizarRetencionDeuceWildSueltaElPar(t *testing.T) {
	cfg, ok := BuscarPoker("deucewild")
	if !ok {
		t.Fatal("no se encontro deucewild")
	}
	// 5-5-9-J-K: sin comodin en la mano, el par bajo no se sostiene (conviene
	// soltarlo para buscar los dos).
	mano := []cartaPoker{
		manoPoker(5, 0), manoPoker(5, 1), manoPoker(9, 2), manoPoker(11, 3), manoPoker(13, 0),
	}
	retener := optimizarRetencion(cfg, mano)
	if retener[0] || retener[1] {
		t.Errorf("deuce wild: sostiene el par bajo (retener=%v)", retener)
	}
	if !retener[3] || !retener[4] {
		t.Errorf("deuce wild: no sostiene las figuras (retener=%v)", retener)
	}
}

// TestOptimizarRetencionDeuceWildSostieneElComodin: el dos es comodin y siempre
// se queda.
func TestOptimizarRetencionDeuceWildSostieneElComodin(t *testing.T) {
	cfg, ok := BuscarPoker("deucewild")
	if !ok {
		t.Fatal("no se encontro deucewild")
	}
	mano := []cartaPoker{
		manoPoker(5, 0), manoPoker(9, 1), {Rango: 2, Palo: 2, Comodin: true}, manoPoker(11, 3), manoPoker(13, 0),
	}
	retener := optimizarRetencion(cfg, mano)
	if !retener[2] {
		t.Errorf("deuce wild: no sostiene el comodin (retener=%v)", retener)
	}
}

// TestOptimizarRetencionEscaleraSinRepetidos: los rangos repetidos no suman
// para la escalera. Con 5,5,7,9 hay tres rangos, no cuatro: debe sostener el
// par bajo y no las cuatro cartas.
func TestOptimizarRetencionEscaleraSinRepetidos(t *testing.T) {
	cfg, ok := BuscarPoker("jacksorbetter")
	if !ok {
		t.Fatal("no se encontro jacksorbetter")
	}
	mano := []cartaPoker{
		manoPoker(5, 0), manoPoker(5, 1), manoPoker(7, 2), manoPoker(9, 3), manoPoker(13, 0),
	}
	retener := optimizarRetencion(cfg, mano)
	if !retener[0] || !retener[1] {
		t.Errorf("deberia sostener el par bajo (retener=%v)", retener)
	}
	for i := 2; i < 5; i++ {
		if retener[i] {
			t.Errorf("sostiene de mas la carta %d (retener=%v)", i, retener)
		}
	}
}

// TestRTPPokerDeclaradoCoincide con lo medido: el RTP declarado no puede estar
// por encima de lo que la estrategia automatica paga (queda margen solo si el
// jugador sostiene sus propias cartas mejor que el sistema).
func TestRTPPokerDeclaradoCoincide(t *testing.T) {
	declarado := map[string]float64{
		"jacksorbetter": 78.4,
		"allamerican":   79.6,
		"tensorbetter":  80.7,
		"acesandfaces":  78.8,
		"eightsandaces": 79.8,
		"twowaysroyal":  80.1,
		"deucewild":     63.5,
		"jokerpoker":    60.9,
	}
	for codigo, esperado := range declarado {
		if got := rtpPoker(codigo); got != esperado {
			t.Errorf("%s: rtpPoker = %.2f, esperaba %.2f", codigo, got, esperado)
		}
	}
}
