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
// casino_rondas, el bingo usa bingo_rondas y gemas usa gemas_rondas. El lobby
// consulta una sola lista y verifica con un solo endpoint, asi que aca se juntan
// las fuentes en un formato comun y el endpoint de verificacion despacha segun
// el origen de la ronda.

// Fuentes del historial. El cliente manda `source` al verificar para no
// depender de que los ids de las tablas no se pisen entre si.
const (
	FuenteCasino = "casino"
	FuenteBingo  = "bingo"
	FuenteGemas  = "gemas"
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
		for _, f := range []string{FuenteCasino, FuenteBingo, FuenteGemas} {
			if h.verificarFuente(c, userID, id, f) {
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "ronda no encontrada"})
	case FuenteCasino, FuenteBingo, FuenteGemas:
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
