package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"desafio-labs-go-1/internal/cep"
	"desafio-labs-go-1/internal/weather"
)

type cepFalso struct {
	end cep.Endereco
	err error
}

func (c cepFalso) Buscar(_ context.Context, code string) (cep.Endereco, error) {
	if c.err != nil {
		return cep.Endereco{}, c.err
	}
	return c.end, nil
}

type climaFalso struct {
	c   float64
	err error
}

func (w climaFalso) Atual(_ context.Context, _ weather.Local) (float64, error) {
	if w.err != nil {
		return 0, w.err
	}
	return w.c, nil
}

func novoHandler(c cepFalso, w climaFalso) http.Handler {
	return NewHandler(c, w)
}

func TestSucesso(t *testing.T) {
	h := novoHandler(
		cepFalso{end: cep.Endereco{Localidade: "São Paulo", UF: "SP"}},
		climaFalso{c: 28.5},
	)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/01153000", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quero 200. corpo: %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, quero application/json", ct)
	}

	var got map[string]float64
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("corpo nao e' json: %v (%s)", err, rec.Body)
	}
	// As chaves sao exatamente as do contrato do desafio.
	quer := map[string]float64{"temp_C": 28.5, "temp_F": 83.3, "temp_K": 301.5}
	if len(got) != len(quer) {
		t.Errorf("corpo tem %d campos (%v), quero %d", len(got), got, len(quer))
	}
	for k, v := range quer {
		if got[k] != v {
			t.Errorf("%s = %v, quero %v", k, got[k], v)
		}
	}
}

func TestCEPChegaNaQueryString(t *testing.T) {
	h := novoHandler(
		cepFalso{end: cep.Endereco{Localidade: "São Paulo", UF: "SP"}},
		climaFalso{c: 10},
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?cep=01153000", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quero 200. corpo: %s", rec.Code, rec.Body)
	}
}

func TestFalhas(t *testing.T) {
	casos := []struct {
		nome       string
		alvo       string
		c          cepFalso
		w          climaFalso
		querStatus int
		querMsg    string
	}{
		{
			nome:       "formato invalido",
			alvo:       "/123",
			c:          cepFalso{err: cep.ErrFormatoInvalido},
			querStatus: http.StatusUnprocessableEntity,
			querMsg:    "invalid zipcode",
		},
		{
			nome:       "cep sem nenhum digito",
			alvo:       "/",
			c:          cepFalso{err: cep.ErrFormatoInvalido},
			querStatus: http.StatusUnprocessableEntity,
			querMsg:    "invalid zipcode",
		},
		{
			nome:       "cep inexistente",
			alvo:       "/99999999",
			c:          cepFalso{err: cep.ErrNaoEncontrado},
			querStatus: http.StatusNotFound,
			querMsg:    "can not find zipcode",
		},
		{
			nome:       "viacep fora do ar nao vira 404",
			alvo:       "/01153000",
			c:          cepFalso{err: cep.ErrIndisponivel},
			querStatus: http.StatusBadGateway,
		},
		{
			nome:       "weatherapi fora do ar nao vira 404",
			alvo:       "/01153000",
			c:          cepFalso{end: cep.Endereco{Localidade: "São Paulo", UF: "SP"}},
			w:          climaFalso{err: weather.ErrIndisponivel},
			querStatus: http.StatusBadGateway,
		},
		{
			nome:       "chave ausente e' erro do servidor",
			alvo:       "/01153000",
			c:          cepFalso{end: cep.Endereco{Localidade: "São Paulo", UF: "SP"}},
			w:          climaFalso{err: weather.ErrSemChave},
			querStatus: http.StatusInternalServerError,
		},
	}

	for _, cs := range casos {
		t.Run(cs.nome, func(t *testing.T) {
			h := novoHandler(cs.c, cs.w)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, cs.alvo, nil))

			if rec.Code != cs.querStatus {
				t.Fatalf("status = %d, quero %d. corpo: %s", rec.Code, cs.querStatus, rec.Body)
			}
			if cs.querMsg != "" {
				if got := strings.TrimSpace(rec.Body.String()); got != cs.querMsg {
					t.Errorf("corpo = %q, quero %q", got, cs.querMsg)
				}
			}
		})
	}
}

func TestMetodoNaoPermitido(t *testing.T) {
	h := novoHandler(cepFalso{}, climaFalso{})
	for _, m := range []string{http.MethodPost, http.MethodDelete, http.MethodPatch} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(m, "/01153000", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status = %d, quero 405", m, rec.Code)
		}
	}
}

func TestHealth(t *testing.T) {
	h := novoHandler(cepFalso{}, climaFalso{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quero 200", rec.Code)
	}
}

func TestLocalRepassadoAoClima(t *testing.T) {
	var visto weather.Local
	h := NewHandler(
		cepFalso{end: cep.Endereco{Localidade: "Santa Maria", UF: "RS"}},
		funcClima(func(_ context.Context, l weather.Local) (float64, error) {
			visto = l
			return 1, nil
		}),
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/97010000", nil))

	// Sem a UF, "Santa Maria" resolveria para a cidade errada.
	if visto.Cidade != "Santa Maria" || visto.UF != "RS" {
		t.Errorf("local = %+v, quero Santa Maria/RS", visto)
	}
	_ = fmt.Sprint(rec.Code)
}

type funcClima func(context.Context, weather.Local) (float64, error)

func (f funcClima) Atual(ctx context.Context, l weather.Local) (float64, error) { return f(ctx, l) }
