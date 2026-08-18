package weather

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAtual(t *testing.T) {
	casos := []struct {
		nome    string
		status  int
		corpo   string
		querErr error
		querC   float64
	}{
		{
			nome:   "sucesso",
			status: http.StatusOK,
			corpo:  `{"location":{"name":"Sao Paulo"},"current":{"temp_c":28.5}}`,
			querC:  28.5,
		},
		{
			nome:   "temperatura negativa",
			status: http.StatusOK,
			corpo:  `{"current":{"temp_c":-3.2}}`,
			querC:  -3.2,
		},
		{
			// 1006 e' como a WeatherAPI diz que nao achou a localidade.
			nome:    "localidade desconhecida",
			status:  http.StatusBadRequest,
			corpo:   `{"error":{"code":1006,"message":"No matching location found."}}`,
			querErr: ErrLocalNaoEncontrado,
		},
		{
			nome:    "chave invalida",
			status:  http.StatusUnauthorized,
			corpo:   `{"error":{"code":2006,"message":"API key is invalid."}}`,
			querErr: ErrIndisponivel,
		},
		{
			nome:    "servico fora",
			status:  http.StatusInternalServerError,
			corpo:   `oops`,
			querErr: ErrIndisponivel,
		},
		{
			nome:    "json quebrado",
			status:  http.StatusOK,
			corpo:   `{"current":`,
			querErr: ErrIndisponivel,
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				if got := q.Get("key"); got != "chave-de-teste" {
					t.Errorf("key = %q, quero chave-de-teste", got)
				}
				// A cidade sozinha e' ambigua no Brasil; a UF e o pais entram
				// na consulta de proposito.
				if got := q.Get("q"); got != "São Paulo,SP,Brazil" {
					t.Errorf("q = %q, quero São Paulo,SP,Brazil", got)
				}
				w.WriteHeader(c.status)
				w.Write([]byte(c.corpo))
			}))
			defer srv.Close()

			cli := NewClient("chave-de-teste", WithBaseURL(srv.URL))
			got, err := cli.Atual(context.Background(), Local{Cidade: "São Paulo", UF: "SP"})

			if c.querErr != nil {
				if !errors.Is(err, c.querErr) {
					t.Fatalf("err = %v, quero %v", err, c.querErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, quero nil", err)
			}
			if got != c.querC {
				t.Errorf("celsius = %v, quero %v", got, c.querC)
			}
		})
	}
}

func TestAtualExigeChave(t *testing.T) {
	cli := NewClient("")
	if _, err := cli.Atual(context.Background(), Local{Cidade: "São Paulo"}); !errors.Is(err, ErrSemChave) {
		t.Fatalf("err = %v, quero ErrSemChave", err)
	}
}
