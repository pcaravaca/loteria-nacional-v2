package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/config"
)

func motorGeoBlock(enabled bool, permitidos map[string]bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(GeoBlock(&config.Config{GeoBlockEnabled: enabled, GeoAllowed: permitidos}))
	e.POST("/jugar", func(c *gin.Context) { c.Status(http.StatusOK) })
	return e
}

func peticion(e *gin.Engine, pais string) int {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/jugar", nil)
	if pais != "" {
		r.Header.Set("CF-IPCountry", pais)
	}
	e.ServeHTTP(w, r)
	return w.Code
}

func TestGeoBlockDesactivado(t *testing.T) {
	e := motorGeoBlock(false, map[string]bool{"CR": true})
	if code := peticion(e, "US"); code != http.StatusOK {
		t.Fatalf("con el bloqueo desactivado deberia dejar jugar, fue %d", code)
	}
	if code := peticion(e, ""); code != http.StatusOK {
		t.Fatalf("sin encabezado y bloqueo desactivado deberia dejar jugar, fue %d", code)
	}
}

func TestGeoBlockActivo(t *testing.T) {
	e := motorGeoBlock(true, map[string]bool{"CR": true, "SV": true})
	if code := peticion(e, "CR"); code != http.StatusOK {
		t.Fatalf("CR deberia estar permitido, fue %d", code)
	}
	if code := peticion(e, "sv"); code != http.StatusOK {
		t.Fatalf("SV en minusculas deberia estar permitido, fue %d", code)
	}
	if code := peticion(e, "US"); code != http.StatusUnavailableForLegalReasons {
		t.Fatalf("US deberia bloquearse con 451, fue %d", code)
	}
	if code := peticion(e, ""); code != http.StatusUnavailableForLegalReasons {
		t.Fatalf("sin encabezado el bloqueo activo debe cerrar, fue %d", code)
	}
}
