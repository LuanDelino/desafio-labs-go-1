// Package api é a borda HTTP: extrai o CEP, costura endereço com clima e
// traduz erro de domínio em status.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"desafio-labs-go-1/internal/cep"
	"desafio-labs-go-1/internal/temperature"
	"desafio-labs-go-1/internal/weather"
)

// As duas interfaces são declaradas aqui, no consumidor, e não nos pacotes que
// as satisfazem: é o que permite testar o handler com dublês.

// BuscadorDeCEP resolve um CEP em endereço.
type BuscadorDeCEP interface {
	Buscar(ctx context.Context, codigo string) (cep.Endereco, error)
}

// ConsultorDeClima devolve a temperatura atual, em Celsius, de uma localidade.
type ConsultorDeClima interface {
	Atual(ctx context.Context, local weather.Local) (float64, error)
}

// Nomes literais do enunciado, inclusive o maiúsculo depois do underscore.
type resposta struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

// erroResposta mantém sucesso e falha no mesmo formato: quem consome decodifica
// JSON sempre, sem ramificar por status.
type erroResposta struct {
	Message string `json:"message"`
}

// Handler serve o endpoint de clima por CEP.
type Handler struct {
	ceps  BuscadorDeCEP
	clima ConsultorDeClima
	mux   *http.ServeMux
}

// NewHandler monta o roteamento sobre as duas dependências.
func NewHandler(ceps BuscadorDeCEP, clima ConsultorDeClima) *Handler {
	h := &Handler{ceps: ceps, clima: clima}

	h.mux = http.NewServeMux()
	h.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// O enunciado não fixa o caminho; servimos as duas convenções em uso.
	h.mux.HandleFunc("GET /{cep}", h.climaPorCEP)
	h.mux.HandleFunc("GET /{$}", h.climaPorCEP)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) climaPorCEP(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("cep")
	if codigo == "" {
		codigo = r.URL.Query().Get("cep")
	}

	endereco, err := h.ceps.Buscar(r.Context(), codigo)
	if err != nil {
		responderErro(w, err)
		return
	}

	celsius, err := h.clima.Atual(r.Context(), weather.Local{
		Cidade: endereco.Localidade,
		UF:     endereco.UF,
	})
	if err != nil {
		responderErro(w, err)
		return
	}

	lido := temperature.FromCelsius(celsius)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resposta{
		TempC: lido.Celsius,
		TempF: lido.Fahrenheit,
		TempK: lido.Kelvin,
	})
}

// As mensagens são as literais do enunciado; só o invólucro é JSON.
func responderErro(w http.ResponseWriter, err error) {
	status, msg := classificar(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(erroResposta{Message: msg})
}

func classificar(err error) (int, string) {
	switch {
	case errors.Is(err, cep.ErrFormatoInvalido):
		return http.StatusUnprocessableEntity, "invalid zipcode"
	case errors.Is(err, cep.ErrNaoEncontrado):
		return http.StatusNotFound, "can not find zipcode"

	// Falha nossa nunca vira 404: seria mentir para quem chama e esconder o incidente.
	case errors.Is(err, weather.ErrSemChave):
		return http.StatusInternalServerError, "internal error"
	case errors.Is(err, cep.ErrIndisponivel),
		errors.Is(err, weather.ErrIndisponivel),
		errors.Is(err, weather.ErrLocalNaoEncontrado):
		return http.StatusBadGateway, "upstream unavailable"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}
