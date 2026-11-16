package middleware

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/config"
)

// GeoBlock rechaza el juego desde paises no permitidos por el operador. El pais
// se toma de los encabezados que entrega la CDN (Cloudflare: CF-IPCountry) o del
// encabezado X-Country si el operador resuelve la ubicacion en su propio proxy.
//
// Desactivado por defecto (GEO_BLOCK_ENABLED=false): sin esa variable no cambia
// nada, asi un entorno local sigue funcionando. Con el bloqueo activo y sin
// encabezado de pais, el acceso se rechaza: mas vale cerrar que admitir por
// defecto cuando el operador exige restringir.
func GeoBlock(cfg *config.Config) gin.HandlerFunc {
	permitidos := make([]string, 0, len(cfg.GeoAllowed))
	for p := range cfg.GeoAllowed {
		permitidos = append(permitidos, p)
	}
	sort.Strings(permitidos)

	return func(c *gin.Context) {
		if !cfg.GeoBlockEnabled {
			c.Next()
			return
		}
		pais := strings.ToUpper(strings.TrimSpace(c.GetHeader("CF-IPCountry")))
		if pais == "" {
			pais = strings.ToUpper(strings.TrimSpace(c.GetHeader("X-Country")))
		}
		if !cfg.GeoPermite(pais) {
			c.AbortWithStatusJSON(http.StatusUnavailableForLegalReasons, gin.H{
				"error":     "el juego no esta disponible desde tu ubicacion",
				"pais":      pais,
				"permitidos": permitidos,
			})
			return
		}
		c.Next()
	}
}
