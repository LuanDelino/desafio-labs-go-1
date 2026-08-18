//go:build roundtrip

// Verificacao de ponta a ponta contra o ViaCEP e a WeatherAPI de verdade.
// Atras da tag roundtrip para que `go test ./...` siga offline:
//
//	WEATHER_API=... go test -tags roundtrip ./internal/api/
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"desafio-labs-go-1/internal/cep"
	"desafio-labs-go-1/internal/weather"
)

func handlerReal(t *testing.T) http.Handler {
	t.Helper()
	chave := os.Getenv("WEATHER_API")
	if chave == "" {
		t.Skip("WEATHER_API nao definida")
	}
	return NewHandler(cep.NewClient(), weather.NewClient(chave))
}

func TestRoundTrip(t *testing.T) {
	h := handlerReal(t)

	casos := []struct {
		nome   string
		alvo   string
		status int
		msg    string
	}{
		{"cep real", "/01310100", http.StatusOK, ""},
		{"cep real via query", "/?cep=97010000", http.StatusOK, ""},
		{"formato invalido", "/1234567", http.StatusUnprocessableEntity, "invalid zipcode"},
		{"formato com hifen", "/01310-100", http.StatusUnprocessableEntity, "invalid zipcode"},
		{"cep inexistente", "/99999999", http.StatusNotFound, "can not find zipcode"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.alvo, nil))

			if rec.Code != c.status {
				t.Fatalf("status = %d, quero %d. corpo: %s", rec.Code, c.status, rec.Body)
			}
			if c.msg != "" {
				if got := strings.TrimSpace(rec.Body.String()); got != c.msg {
					t.Fatalf("corpo = %q, quero %q", got, c.msg)
				}
				return
			}

			var r map[string]float64
			if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
				t.Fatalf("corpo nao e' json: %v (%s)", err, rec.Body)
			}
			for _, k := range []string{"temp_C", "temp_F", "temp_K"} {
				if _, ok := r[k]; !ok {
					t.Errorf("falta o campo %s em %v", k, r)
				}
			}
			// As tres escalas tem que ser coerentes entre si.
			if d := r["temp_F"] - (r["temp_C"]*1.8 + 32); d > 0.01 || d < -0.01 {
				t.Errorf("temp_F incoerente com temp_C: %v", r)
			}
			if d := r["temp_K"] - (r["temp_C"] + 273); d > 0.01 || d < -0.01 {
				t.Errorf("temp_K incoerente com temp_C: %v", r)
			}
		})
	}
}
