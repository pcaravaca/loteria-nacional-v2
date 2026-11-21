package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/models"
)

// Historial y verificacion unificados del casino.
//
// Los juegos no todos guardan sus jugadas en la misma tabla: los rapidos usan
// casino_rondas, el bingo usa bingo_rondas, gemas usa gemas_rondas y el
// blackjack guarda cada mano en casino_blackjack. El lobby consulta una sola
// lista y verifica con un solo endpoint, asi que aca se juntan las fuentes en
// un formato comun y el endpoint de verificacion despacha segun el origen de
// la ronda.

// Fuentes del historial. El cliente manda `source` al verificar para no
// depender de que los ids de las tablas no se pisen entre si.
const (
	FuenteCasino         = "casino"
	FuenteBingo          = "bingo"
	FuenteGemas          = "gemas"
	FuenteBlackjack      = "blackjack"
	FuenteDomino         = "domino"
	FuenteAjedrez        = "ajedrez"
	FuenteAjedrezPartida = "ajedrez_partida"
	FuenteBackgammon     = "backgammon"
	FuenteTetris         = "tetris"
)

// RondaHistorial es la fila comun del historial. Mantiene los campos que ya
// consumia CasinoView (id, game, monto, multiplicador, premio, created_at) y
// suma `source` y los datos propios de cada juego.
type RondaHistorial struct {
	ID             uint      `json:"id"`
	Source         string    `json:"source"`
	Game           string    `json:"game"`
	Nombre         string    `json:"nombre"`
	Monto          int64     `json:"monto"`
	Multiplicador  float64   `json:"multiplicador"`
	Premio         int64     `json:"premio"`
	Puntos         int64     `json:"puntos,omitempty"`      // gemas
	Lineas         int       `json:"lineas,omitempty"`      // bingo
	Cartones       int       `json:"cartones,omitempty"`    // bingo
	Movimientos    int       `json:"movimientos,omitempty"` // gemas
	Estado         string    `json:"estado"`
	ClientSeed     string    `json:"client_seed"`
	ServerSeedHash string    `json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	Resultado      any       `json:"resultado,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Verificacion es la respuesta comun de la verificacion provably fair, sea la
// ronda de casino_rondas, bingo_rondas o gemas_rondas.
type Verificacion struct {
	RondaID        uint      `json:"ronda_id"`
	Source         string    `json:"source"`
	Juego          string    `json:"juego"`
	Nombre         string    `json:"nombre"`
	Monto          int64     `json:"monto"`
	Multiplicador  float64   `json:"multiplicador"`
	Premio         int64     `json:"premio"`
	Puntos         int64     `json:"puntos,omitempty"`
	Lineas         int       `json:"lineas,omitempty"`
	Cartones       int       `json:"cartones,omitempty"`
	Movimientos    int       `json:"movimientos,omitempty"`
	Estado         string    `json:"estado,omitempty"`
	ClientSeed     string    `json:"client_seed"`
	ServerSeedHash string    `json:"server_seed_hash"`
	ServerSeed     string    `json:"server_seed"`
	Nonce          int64     `json:"nonce"`
	Sorteo         any       `json:"sorteo,omitempty"`
	Resultado      any       `json:"resultado,omitempty"`
	Verificado     bool      `json:"verificado"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
}

// nombresJuegos arma el mapa codigo -> nombre para no mostrar el codigo crudo.
func (h *CasinoHandler) nombresJuegos() map[string]string {
	var juegos []models.CasinoGame
	if err := h.db.Find(&juegos).Error; err != nil {
		return map[string]string{}
	}
	nombres := make(map[string]string, len(juegos))
	for _, j := range juegos {
		nombres[j.Codigo] = j.Nombre
	}
	return nombres
}

// nombreJuego resuelve el nombre de un juego; si no esta en el catalogo
// devuelve el codigo.
func (h *CasinoHandler) nombreJuego(codigo string) string {
	var juego models.CasinoGame
	if err := h.db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
		return codigo
	}
	return juego.Nombre
}

// historialRondas junta las ultimas jugadas del usuario de todas las fuentes y
// las ordena por fecha. `limite` se aplica al resultado final, no por fuente.
func (h *CasinoHandler) historialRondas(userID uint, limite int) []RondaHistorial {
	nombres := h.nombresJuegos()
	filas := make([]RondaHistorial, 0, limite*3)

	var rondas []models.CasinoRound
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&rondas).Error; err == nil {
		for _, r := range rondas {
			filas = append(filas, RondaHistorial{
				ID:             r.ID,
				Source:         FuenteCasino,
				Game:           r.GameCodigo,
				Nombre:         nombres[r.GameCodigo],
				Monto:          r.Monto,
				Multiplicador:  float64(r.Multiplicador),
				Premio:         r.Premio,
				Estado:         "pagada",
				ClientSeed:     r.ClientSeed,
				ServerSeedHash: r.ServerSeedHash,
				Nonce:          r.Nonce,
				Resultado:      r.Resultado,
				CreatedAt:      r.CreatedAt,
			})
		}
	}

	var bingos []models.BingoRonda
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&bingos).Error; err == nil {
		for _, r := range bingos {
			codigo := r.Variante
			if codigo == "" {
				codigo = models.CasinoBingo
			}
			_, lineas, lleno, _, _ := casino.BingoPremioVariante(r.MontoPorCarton, r.Grids, r.Cantadas, codigo)
			nombre := nombres[codigo]
			if nombre == "" {
				nombre = "Bingo"
			}
			filas = append(filas, RondaHistorial{
				ID:             r.ID,
				Source:         FuenteBingo,
				Game:           codigo,
				Nombre:         nombre,
				Monto:          r.Monto,
				Multiplicador:  r.Multiplicador,
				Premio:         r.Premio,
				Lineas:         lineas,
				Cartones:       r.Cartones,
				Estado:         r.Estado,
				ClientSeed:     r.ClientSeed,
				ServerSeedHash: r.ServerSeedHash,
				Nonce:          r.Nonce,
				Resultado: gin.H{
					"lineas":   lineas,
					"lleno":    lleno,
					"cantadas": len(r.Cantadas),
					"cartones": r.Cartones,
				},
				CreatedAt: r.CreatedAt,
			})
		}
	}

	var gemas []models.GemasRonda
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&gemas).Error; err == nil {
		for _, r := range gemas {
			filas = append(filas, RondaHistorial{
				ID:             r.ID,
				Source:         FuenteGemas,
				Game:           models.CasinoGemas,
				Nombre:         nombres[models.CasinoGemas],
				Monto:          r.Monto,
				Multiplicador:  r.Multiplicador,
				Premio:         r.Premio,
				Puntos:         r.Puntos,
				Movimientos:    r.Movimientos,
				Estado:         r.Estado,
				ClientSeed:     r.ClientSeed,
				ServerSeedHash: r.ServerSeedHash,
				Nonce:          r.Nonce,
				Resultado: gin.H{
					"puntos":      r.Puntos,
					"movimientos": r.Movimientos,
				},
				CreatedAt: r.CreatedAt,
			})
		}
	}

	var manos []models.BlackjackMano
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&manos).Error; err == nil {
		for _, r := range manos {
			filas = append(filas, RondaHistorial{
				ID:             r.ID,
				Source:         FuenteBlackjack,
				Game:           models.CasinoBlackjack,
				Nombre:         nombres[models.CasinoBlackjack],
				Monto:          r.Monto,
				Multiplicador:  r.Multiplicador,
				Premio:         r.Premio,
				Estado:         r.Estado,
				ClientSeed:     r.ClientSeed,
				ServerSeedHash: r.ServerSeedHash,
				Nonce:          r.Nonce,
				Resultado: gin.H{
					"mano":    r.Mano,
					"crupier": r.Crupier,
					"seguro":  r.Seguro,
				},
				CreatedAt: r.CreatedAt,
			})
		}
	}

	var dominos []models.DominoRonda
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&dominos).Error; err == nil {
		for _, r := range dominos {
			filas = append(filas, RondaHistorial{
				ID:             r.ID,
				Source:         FuenteDomino,
				Game:           models.CasinoDomino,
				Nombre:         nombres[models.CasinoDomino],
				Monto:          r.Monto,
				Multiplicador:  r.Multiplicador,
				Premio:         r.Premio,
				Estado:         r.Estado,
				ClientSeed:     r.ClientSeed,
				ServerSeedHash: r.ServerSeedHash,
				Nonce:          r.Nonce,
				Resultado: gin.H{
					"mano":      r.Mano,
					"casa":      r.Casa,
					"tablero":   r.Tablero,
					"jugadas":   len(r.Jugadas),
					"fichas":    len(r.Mano),
					"contrario": len(r.Casa),
				},
				CreatedAt: r.CreatedAt,
			})
		}
	}

	var ajedrez []models.AjedrezRonda
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&ajedrez).Error; err == nil {
		for _, r := range ajedrez {
			filas = append(filas, RondaHistorial{
				ID:             r.ID,
				Source:         FuenteAjedrez,
				Game:           models.CasinoAjedrez,
				Nombre:         nombres[models.CasinoAjedrez],
				Monto:          r.Monto,
				Multiplicador:  r.Multiplicador,
				Premio:         r.Premio,
				Estado:         r.Estado,
				ClientSeed:     r.ClientSeed,
				ServerSeedHash: r.ServerSeedHash,
				Nonce:          r.Nonce,
				Resultado: gin.H{
					"fen":      r.FEN,
					"tema":     r.Tema,
					"opciones": r.Opciones,
					"jugadas":  len(r.Jugadas),
				},
				CreatedAt: r.CreatedAt,
			})
		}
	}

	var partidas []models.AjedrezPartida
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&partidas).Error; err == nil {
		for _, p := range partidas {
			filas = append(filas, RondaHistorial{
				ID:             p.ID,
				Source:         FuenteAjedrezPartida,
				Game:           models.CasinoAjedrez,
				Nombre:         nombres[models.CasinoAjedrez],
				Monto:          p.Monto,
				Multiplicador:  p.Multiplicador,
				Premio:         p.Premio,
				Estado:         p.Estado,
				ClientSeed:     p.ClientSeed,
				ServerSeedHash: p.ServerSeedHash,
				Nonce:          p.Nonce,
				Resultado: gin.H{
					"fen":       p.FEN,
					"color":     p.ColorJugador,
					"resultado": p.Resultado,
					"motivo":    p.Motivo,
					"jugadas":   len(p.Jugadas),
				},
				CreatedAt: p.CreatedAt,
			})
		}
	}

	var backgammons []models.BackgammonRonda
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&backgammons).Error; err == nil {
		for _, r := range backgammons {
			filas = append(filas, RondaHistorial{
				ID:             r.ID,
				Source:         FuenteBackgammon,
				Game:           models.CasinoBackgammon,
				Nombre:         nombres[models.CasinoBackgammon],
				Monto:          r.Monto,
				Multiplicador:  r.Multiplicador,
				Premio:         r.Premio,
				Estado:         r.Estado,
				ClientSeed:     r.ClientSeed,
				ServerSeedHash: r.ServerSeedHash,
				Nonce:          r.Nonce,
				Resultado: gin.H{
					"jugador": r.Jugador,
					"casa":    r.Casa,
					"ganador": r.Ganador,
					"jugadas": len(r.Jugadas),
				},
				CreatedAt: r.CreatedAt,
			})
		}
	}

	var tetris []models.TetrisRonda
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at desc").Limit(limite).Find(&tetris).Error; err == nil {
		for _, r := range tetris {
			filas = append(filas, RondaHistorial{
				ID:             r.ID,
				Source:         FuenteTetris,
				Game:           models.CasinoTetris,
				Nombre:         nombres[models.CasinoTetris],
				Monto:          r.Monto,
				Multiplicador:  r.Multiplicador,
				Premio:         r.Premio,
				Lineas:         r.Lineas,
				Estado:         r.Estado,
				ClientSeed:     r.ClientSeed,
				ServerSeedHash: r.ServerSeedHash,
				Nonce:          r.Nonce,
				Resultado: gin.H{
					"lineas":  r.Lineas,
					"pieza":   r.Pieza,
					"piezas":  len(r.Piezas),
					"jugadas": len(r.Jugadas),
				},
				CreatedAt: r.CreatedAt,
			})
		}
	}

	sort.SliceStable(filas, func(i, j int) bool { return filas[i].CreatedAt.After(filas[j].CreatedAt) })
	if len(filas) > limite {
		filas = filas[:limite]
	}
	return filas
}

// Verify es el endpoint unificado de verificacion. Acepta `?source=casino`,
// `?source=bingo` o `?source=gemas`; sin `source` busca la ronda en las tres
// fuentes (casero para el cliente viejo).
func (h *CasinoHandler) Verify(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	switch fuente := c.Query("source"); fuente {
	case "":
		for _, f := range []string{
			FuenteCasino, FuenteBingo, FuenteGemas, FuenteBlackjack,
			FuenteDomino, FuenteAjedrez, FuenteAjedrezPartida, FuenteBackgammon, FuenteTetris,
		} {
			if h.verificarFuente(c, userID, id, f) {
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "ronda no encontrada"})
	case FuenteCasino, FuenteBingo, FuenteGemas, FuenteBlackjack,
		FuenteDomino, FuenteAjedrez, FuenteAjedrezPartida, FuenteBackgammon, FuenteTetris:
		if !h.verificarFuente(c, userID, id, fuente) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ronda no encontrada"})
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "fuente invalida"})
	}
}

// verificarFuente despacha a la verificacion de cada tabla. Devuelve true si
// respondio (encontró la ronda o fallo de usuario); false si la ronda no existe
// en esa fuente.
func (h *CasinoHandler) verificarFuente(c *gin.Context, userID uint, id uint64, fuente string) bool {
	switch fuente {
	case FuenteCasino:
		return h.verificarCasino(c, userID, id)
	case FuenteBingo:
		return h.verificarBingo(c, userID, id)
	case FuenteGemas:
		return h.verificarGemas(c, userID, id)
	case FuenteBlackjack:
		return h.verificarBlackjack(c, userID, id)
	case FuenteDomino:
		return h.verificarDomino(c, userID, id)
	case FuenteAjedrez:
		return h.verificarAjedrez(c, userID, id)
	case FuenteAjedrezPartida:
		return h.verificarAjedrezPartida(c, userID, id)
	case FuenteBackgammon:
		return h.verificarBackgammon(c, userID, id)
	case FuenteTetris:
		return h.verificarTetris(c, userID, id)
	}
	return false
}

func (h *CasinoHandler) verificarCasino(c *gin.Context, userID uint, id uint64) bool {
	var ronda models.CasinoRound
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	draws := casino.Sortear(user.CasinoSeed, ronda.ClientSeed, ronda.Nonce, casino.Draws(ronda.GameCodigo), 1<<31-1)
	c.JSON(http.StatusOK, Verificacion{
		RondaID:        ronda.ID,
		Source:         FuenteCasino,
		Juego:          ronda.GameCodigo,
		Nombre:         h.nombreJuego(ronda.GameCodigo),
		Monto:          ronda.Monto,
		Multiplicador:  float64(ronda.Multiplicador),
		Premio:         ronda.Premio,
		Estado:         "pagada",
		ClientSeed:     ronda.ClientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          ronda.Nonce,
		Sorteo:         draws,
		Resultado:      ronda.Resultado,
		Verificado:     casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash,
		CreatedAt:      ronda.CreatedAt,
	})
	return true
}

func (h *CasinoHandler) verificarBingo(c *gin.Context, userID uint, id uint64) bool {
	var ronda models.BingoRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	orden, grids := casino.BingoSorteo(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Cartones)
	premio, lineas, lleno, tiers := casino.BingoPremio(ronda.MontoPorCarton, grids, ronda.Cantadas)

	verificado := casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		igualarEnteros(orden, ronda.Orden) &&
		igualarGrids(grids, ronda.Grids)
	// Si la ronda ya se cerro, el premio guardado tiene que coincidir con el
	// que sale de las semillas y las bolillas cantadas.
	if ronda.Estado == "pagada" {
		verificado = verificado && ronda.Premio == premio && ronda.Lineas == lineas
	}

	c.JSON(http.StatusOK, Verificacion{
		RondaID:        ronda.ID,
		Source:         FuenteBingo,
		Juego:          models.CasinoBingo,
		Nombre:         h.nombreJuego(models.CasinoBingo),
		Monto:          ronda.Monto,
		Multiplicador:  ronda.Multiplicador,
		Premio:         ronda.Premio,
		Lineas:         lineas,
		Cartones:       ronda.Cartones,
		Estado:         ronda.Estado,
		ClientSeed:     ronda.ClientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          ronda.Nonce,
		Sorteo:         orden,
		Resultado: gin.H{
			"grids":    ronda.Grids,
			"cantadas": ronda.Cantadas,
			"lineas":   lineas,
			"lleno":    lleno,
			"tiers":    tiers,
			"cartones": ronda.Cartones,
			"premio":   premio,
		},
		Verificado: verificado,
		CreatedAt:  ronda.CreatedAt,
	})
	return true
}

func (h *CasinoHandler) verificarGemas(c *gin.Context, userID uint, id uint64) bool {
	var ronda models.GemasRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	tablero, offset := casino.GemasSorteo(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce)
	chorro := casino.NuevoGemasChorro(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce)
	chorro.Saltar(offset)
	repetido, err := casino.GemasRepetir(tablero, chorro, ronda.Jugadas)

	verificado := err == nil &&
		casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		offset == ronda.StreamOffset &&
		repetido.Puntos == ronda.Puntos &&
		igualarTablero(repetido.Tablero, ronda.Tablero)
	if ronda.Estado == "pagada" {
		verificado = verificado && ronda.Premio == casino.GemasPremio(ronda.Monto, ronda.Puntos)
	}

	c.JSON(http.StatusOK, Verificacion{
		RondaID:        ronda.ID,
		Source:         FuenteGemas,
		Juego:          models.CasinoGemas,
		Nombre:         h.nombreJuego(models.CasinoGemas),
		Monto:          ronda.Monto,
		Multiplicador:  ronda.Multiplicador,
		Premio:         ronda.Premio,
		Puntos:         ronda.Puntos,
		Movimientos:    ronda.Movimientos,
		Estado:         ronda.Estado,
		ClientSeed:     ronda.ClientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          ronda.Nonce,
		Sorteo:         tablero,
		Resultado: gin.H{
			"tablero":     ronda.Tablero,
			"jugadas":     ronda.Jugadas,
			"puntos":      ronda.Puntos,
			"movimientos": ronda.Movimientos,
		},
		Verificado: verificado,
		CreatedAt:  ronda.CreatedAt,
	})
	return true
}

// verificarBlackjack reproduce la mano inicial desde las semillas: el mazo sale
// de Baraja(client, server, nonce) y las posiciones 0/2 son del jugador y 1/3
// del crupier. Verificar las cuatro cartas iniciales es suficiente para probar
// que la mano no se altero despues del reparto.
func (h *CasinoHandler) verificarBlackjack(c *gin.Context, userID uint, id uint64) bool {
	var ronda models.BlackjackMano
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	mazo := casino.Baraja(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, 64)
	verificado := casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		len(ronda.Mano) >= 2 && len(ronda.Crupier) >= 2 &&
		ronda.Mano[0] == mazo[0] && ronda.Mano[1] == mazo[2] &&
		ronda.Crupier[0] == mazo[1] && ronda.Crupier[1] == mazo[3]

	c.JSON(http.StatusOK, Verificacion{
		RondaID:        ronda.ID,
		Source:         FuenteBlackjack,
		Juego:          models.CasinoBlackjack,
		Nombre:         h.nombreJuego(models.CasinoBlackjack),
		Monto:          ronda.Monto,
		Multiplicador:  ronda.Multiplicador,
		Premio:         ronda.Premio,
		Estado:         ronda.Estado,
		ClientSeed:     ronda.ClientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          ronda.Nonce,
		Sorteo:         mazo,
		Resultado: gin.H{
			"mano":    ronda.Mano,
			"crupier": ronda.Crupier,
			"seguro":  ronda.Seguro,
		},
		Verificado: verificado,
		CreatedAt:  ronda.CreatedAt,
	})
	return true
}

// verificarDomino reproduce la mano desde las semillas: el reparto sale de
// DominoSorteo y la casa responde siempre con la ficha valida de mayor
// puntaje, de modo que las jugadas guardadas reproducen mano, casa y tablero.
// Si la ronda ya se cerro, ademas exige que el premio y el multiplicador
// coincidan con el ganador reproducido.
func (h *CasinoHandler) verificarDomino(c *gin.Context, userID uint, id uint64) bool {
	var ronda models.DominoRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	rep, err := casino.DominoRepetir(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Jugadas)
	verificado := err == nil &&
		casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		dominoIgualar(dominoManoA(rep.Mano), ronda.Mano) &&
		dominoIgualar(dominoManoA(rep.Casa), ronda.Casa) &&
		dominoIgualar(dominoManoA(rep.Tablero), ronda.Tablero)
	if ronda.Estado == "pagada" {
		verificado = verificado && rep.Fin &&
			ronda.Premio == casino.DominoPremio(ronda.Monto, rep.Ganador) &&
			ronda.Multiplicador == casino.DominoMultiplicador(rep.Ganador)
	}

	c.JSON(http.StatusOK, Verificacion{
		RondaID:        ronda.ID,
		Source:         FuenteDomino,
		Juego:          models.CasinoDomino,
		Nombre:         h.nombreJuego(models.CasinoDomino),
		Monto:          ronda.Monto,
		Multiplicador:  ronda.Multiplicador,
		Premio:         ronda.Premio,
		Estado:         ronda.Estado,
		ClientSeed:     ronda.ClientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          ronda.Nonce,
		Sorteo: gin.H{
			"mano":    ronda.Mano,
			"casa":    ronda.Casa,
			"tablero": ronda.Tablero,
		},
		Resultado: gin.H{
			"jugadas": ronda.Jugadas,
			"ganador": rep.Ganador,
			"fin":     rep.Fin,
		},
		Verificado: verificado,
		CreatedAt:  ronda.CreatedAt,
	})
	return true
}

// verificarAjedrez reproduce la posicion y el orden de las opciones desde las
// semillas. Cubre el hash de la semilla, el FEN, las cuatro opciones y el
// premio/multiplicador segun el acierto reproducido.
func (h *CasinoHandler) verificarAjedrez(c *gin.Context, userID uint, id uint64) bool {
	var ronda models.AjedrezRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	p, opciones, acierto := casino.AjedrezRepetir(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Jugadas)
	verificado := casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		p.FEN == ronda.FEN &&
		p.Correcta == ronda.Correcta &&
		igualarCadenas(opciones, ronda.Opciones) &&
		ronda.Premio == casino.AjedrezPremio(ronda.Monto, acierto) &&
		ronda.Multiplicador == casino.AjedrezMultiplicador(acierto)

	c.JSON(http.StatusOK, Verificacion{
		RondaID:        ronda.ID,
		Source:         FuenteAjedrez,
		Juego:          models.CasinoAjedrez,
		Nombre:         h.nombreJuego(models.CasinoAjedrez),
		Monto:          ronda.Monto,
		Multiplicador:  ronda.Multiplicador,
		Premio:         ronda.Premio,
		Estado:         ronda.Estado,
		ClientSeed:     ronda.ClientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          ronda.Nonce,
		Sorteo:         opciones,
		Resultado: gin.H{
			"fen":      ronda.FEN,
			"tema":     ronda.Tema,
			"correcta": ronda.Correcta,
			"jugadas":  ronda.Jugadas,
			"acierto":  acierto,
		},
		Verificado: verificado,
		CreatedAt:  ronda.CreatedAt,
	})
	return true
}

// verificarAjedrezPartida reproduce la partida completa contra el adversario:
// reconstruye la posicion desde las jugadas guardadas y vuelve a pedir cada
// jugada de la casa a partir de las semillas. Cubre hash de semilla, FEN final,
// resultado, premio y multiplicador.
func (h *CasinoHandler) verificarAjedrezPartida(c *gin.Context, userID uint, id uint64) bool {
	var partida models.AjedrezPartida
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&partida).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	j, reconstruido := casino.AjedrezReconstruir(partida.Jugadas)
	coincide := reconstruido && j.Posicion.AFEN() == partida.FEN && j.Resultado == partida.Resultado

	pos := casino.NuevoAjedrezJuego()
	casaOK := true
	for ply, uci := range partida.Jugadas {
		esCasa := (pos.Posicion.Turno == 'w') != (partida.ColorJugador == "blancas")
		if esCasa {
			m, ok := casino.AjedrezJugadaCasa(partida.ClientSeed, user.CasinoSeed, partida.Nonce, &pos.Posicion, ply)
			if !ok || m.UCI() != uci {
				casaOK = false
				break
			}
		}
		if !pos.AplicarUCI(uci) {
			casaOK = false
			break
		}
	}
	premioOK := partida.Premio == casino.AjedrezPartidaPremio(partida.Monto, partida.Resultado, partida.ColorJugador)
	multOK := partida.Multiplicador == casino.AjedrezPartidaMultiplicador(partida.Resultado, partida.ColorJugador)
	hashOK := casino.HashSemilla(user.CasinoSeed) == partida.ServerSeedHash
	verificado := coincide && casaOK && premioOK && multOK && hashOK

	c.JSON(http.StatusOK, Verificacion{
		RondaID:        partida.ID,
		Source:         FuenteAjedrezPartida,
		Juego:          models.CasinoAjedrez,
		Nombre:         h.nombreJuego(models.CasinoAjedrez),
		Monto:          partida.Monto,
		Multiplicador:  partida.Multiplicador,
		Premio:         partida.Premio,
		Estado:         partida.Estado,
		ClientSeed:     partida.ClientSeed,
		ServerSeedHash: partida.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          partida.Nonce,
		Resultado: gin.H{
			"fen":       partida.FEN,
			"color":     partida.ColorJugador,
			"resultado": partida.Resultado,
			"motivo":    partida.Motivo,
			"jugadas":   partida.Jugadas,
		},
		Verificado: verificado,
		CreatedAt:  partida.CreatedAt,
	})
	return true
}

// verificarBackgammon reproduce la carrera completa desde las semillas cuando
// la ronda ya esta cerrada (la casa mueve siempre la ficha mas atrasada). La
// ronda abierta no se puede reproducir entera porque el motor sigue pidiendo
// jugadas hasta el fin; para ese estado se verifican el hash de la semilla y
// que los dados guardados coincidan con el lance que les toca.
func (h *CasinoHandler) verificarBackgammon(c *gin.Context, userID uint, id uint64) bool {
	var ronda models.BackgammonRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	dado1, dado2 := casino.BackgammonDados(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Lance)
	verificado := casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		dado1 == ronda.Dado1 && dado2 == ronda.Dado2
	rep, err := casino.BackgammonRepetir(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Jugadas)
	if ronda.Estado == "pagada" {
		verificado = verificado && err == nil && rep.Fin &&
			rep.Ganador == ronda.Ganador &&
			igualarEnteros(rep.Jugador[:], ronda.Jugador[:]) &&
			igualarEnteros(rep.Casa[:], ronda.Casa[:]) &&
			ronda.Premio == casino.BackgammonPremio(ronda.Monto, rep.Ganador) &&
			ronda.Multiplicador == casino.BackgammonMultiplicador(rep.Ganador)
	}

	c.JSON(http.StatusOK, Verificacion{
		RondaID:        ronda.ID,
		Source:         FuenteBackgammon,
		Juego:          models.CasinoBackgammon,
		Nombre:         h.nombreJuego(models.CasinoBackgammon),
		Monto:          ronda.Monto,
		Multiplicador:  ronda.Multiplicador,
		Premio:         ronda.Premio,
		Estado:         ronda.Estado,
		ClientSeed:     ronda.ClientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          ronda.Nonce,
		Sorteo:         []int{dado1, dado2},
		Resultado: gin.H{
			"jugador": ronda.Jugador,
			"casa":    ronda.Casa,
			"ganador": ronda.Ganador,
			"jugadas": ronda.Jugadas,
		},
		Verificado: verificado,
		CreatedAt:  ronda.CreatedAt,
	})
	return true
}

// verificarTetris reproduce el tablero y las lineas desde las semillas: la
// secuencia de piezas sale de TetrisSecuencia y cada jugada guardada coloca la
// pieza en la columna y rotacion elegidas.
func (h *CasinoHandler) verificarTetris(c *gin.Context, userID uint, id uint64) bool {
	var ronda models.TetrisRonda
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&ronda).Error; err != nil {
		return false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return true
	}

	tablero, lineas, err := casino.TetrisRepetir(ronda.ClientSeed, user.CasinoSeed, ronda.Nonce, ronda.Jugadas)
	verificado := err == nil &&
		casino.HashSemilla(user.CasinoSeed) == ronda.ServerSeedHash &&
		lineas == ronda.Lineas &&
		tetrisIgualar(tablero, ronda.Tablero) &&
		ronda.Premio == casino.TetrisPremio(ronda.Monto, lineas) &&
		ronda.Multiplicador == casino.TetrisMult(lineas)

	c.JSON(http.StatusOK, Verificacion{
		RondaID:        ronda.ID,
		Source:         FuenteTetris,
		Juego:          models.CasinoTetris,
		Nombre:         h.nombreJuego(models.CasinoTetris),
		Monto:          ronda.Monto,
		Multiplicador:  ronda.Multiplicador,
		Premio:         ronda.Premio,
		Lineas:         lineas,
		Estado:         ronda.Estado,
		ClientSeed:     ronda.ClientSeed,
		ServerSeedHash: ronda.ServerSeedHash,
		ServerSeed:     user.CasinoSeed,
		Nonce:          ronda.Nonce,
		Sorteo:         ronda.Piezas,
		Resultado: gin.H{
			"tablero": ronda.Tablero,
			"piezas":  ronda.Piezas,
			"lineas":  lineas,
			"jugadas": ronda.Jugadas,
		},
		Verificado: verificado,
		CreatedAt:  ronda.CreatedAt,
	})
	return true
}

func igualarEnteros(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func igualarCadenas(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func igualarGrids(a, b [][][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for r := range a[i] {
			if len(a[i][r]) != len(b[i][r]) {
				return false
			}
			for c := range a[i][r] {
				if a[i][r][c] != b[i][r][c] {
					return false
				}
			}
		}
	}
	return true
}
