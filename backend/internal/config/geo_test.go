package config

import "testing"

func TestEnvCountriesYGeoPermite(t *testing.T) {
	t.Setenv("GEO_ALLOWED_COUNTRIES", "")
	if got := envCountries(""); !got["CR"] || len(got) != 1 {
		t.Fatalf("sin valor deberia quedar solo CR, fue %v", got)
	}

	t.Setenv("GEO_ALLOWED_COUNTRIES", "cr, SV ,gt")
	cfg := &Config{GeoAllowed: envCountries("cr, SV ,gt")}
	for _, p := range []string{"CR", "cr", "Sv"} {
		if !cfg.GeoPermite(p) {
			t.Errorf("%s deberia estar permitido", p)
		}
	}
	if cfg.GeoPermite("US") {
		t.Error("US no deberia estar permitido")
	}
	if cfg.GeoPermite("") {
		t.Error("un pais vacio nunca se permite")
	}
}

func TestEnvBool(t *testing.T) {
	t.Setenv("BOOL_TEST", "")
	if envBool("BOOL_TEST", false) {
		t.Error("vacio deberia tomar el default false")
	}
	t.Setenv("BOOL_TEST", "true")
	if !envBool("BOOL_TEST", false) {
		t.Error("true deberia activar")
	}
	t.Setenv("BOOL_TEST", "no-se")
	if envBool("BOOL_TEST", true) != true {
		t.Error("valor invalido deberia tomar el default")
	}
}
