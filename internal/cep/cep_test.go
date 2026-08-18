package cep

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidar(t *testing.T) {
	validos := []string{"01153000", "00000000", "99999999"}
	for _, v := range validos {
		if err := Validar(v); err != nil {
			t.Errorf("Validar(%q) = %v, quero nil", v, err)
		}
	}

	invalidos := []string{
		"",           // vazio
		"1234567",    // 7 digitos
		"123456789",  // 9 digitos
		"01153-000",  // hifen e' caractere invalido
		"0115300a",   // letra
		" 01153000",  // espaco a esquerda
		"01153000 ",  // espaco a direita
		"abcdefgh",   // so letras
	}
	for _, v := range invalidos {
		if !errors.Is(Validar(v), ErrFormatoInvalido) {
			t.Errorf("Validar(%q) = %v, quero ErrFormatoInvalido", v, Validar(v))
		}
	}
}

func TestBuscar(t *testing.T) {
	casos := []struct {
		nome       string
		status     int
		corpo      string
		querErr    error
		querCidade string
		querUF     string
	}{
		{
			nome:   "sucesso",
			status: http.StatusOK,
			corpo: `{"cep":"01153-000","logradouro":"Rua Vitorino Carmilo",
			         "bairro":"Barra Funda","localidade":"São Paulo","uf":"SP"}`,
			querCidade: "São Paulo",
			querUF:     "SP",
		},
		{
			// A pegadinha: ViaCEP devolve 200 OK para CEP inexistente.
			nome:    "nao encontrado com erro string",
			status:  http.StatusOK,
			corpo:   `{"erro": "true"}`,
			querErr: ErrNaoEncontrado,
		},
		{
			// A mesma API ja devolveu booleano em vez de string.
			nome:    "nao encontrado com erro booleano",
			status:  http.StatusOK,
			corpo:   `{"erro": true}`,
			querErr: ErrNaoEncontrado,
		},
		{
			nome:    "corpo sem localidade",
			status:  http.StatusOK,
			corpo:   `{"cep":"01153-000","uf":"SP"}`,
			querErr: ErrNaoEncontrado,
		},
		{
			nome:    "viacep recusa o formato",
			status:  http.StatusBadRequest,
			corpo:   `{"erro": true}`,
			querErr: ErrNaoEncontrado,
		},
		{
			nome:    "indisponivel",
			status:  http.StatusInternalServerError,
			corpo:   `oops`,
			querErr: ErrIndisponivel,
		},
		{
			nome:    "json quebrado",
			status:  http.StatusOK,
			corpo:   `{"localidade":`,
			querErr: ErrIndisponivel,
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/01153000/json/" {
					t.Errorf("path = %q, quero /01153000/json/", r.URL.Path)
				}
				w.WriteHeader(c.status)
				w.Write([]byte(c.corpo))
			}))
			defer srv.Close()

			cli := NewClient(WithBaseURL(srv.URL))
			end, err := cli.Buscar(context.Background(), "01153000")

			if c.querErr != nil {
				if !errors.Is(err, c.querErr) {
					t.Fatalf("err = %v, quero %v", err, c.querErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, quero nil", err)
			}
			if end.Localidade != c.querCidade {
				t.Errorf("Localidade = %q, quero %q", end.Localidade, c.querCidade)
			}
			if end.UF != c.querUF {
				t.Errorf("UF = %q, quero %q", end.UF, c.querUF)
			}
		})
	}
}

func TestBuscarValidaAntesDeChamar(t *testing.T) {
	chamou := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamou = true
	}))
	defer srv.Close()

	cli := NewClient(WithBaseURL(srv.URL))
	if _, err := cli.Buscar(context.Background(), "123"); !errors.Is(err, ErrFormatoInvalido) {
		t.Fatalf("err = %v, quero ErrFormatoInvalido", err)
	}
	if chamou {
		t.Error("chamou o ViaCEP com CEP de formato invalido; devia barrar antes da rede")
	}
}
