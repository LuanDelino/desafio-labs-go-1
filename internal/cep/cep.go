// Package cep resolve um CEP brasileiro em cidade e UF consultando o ViaCEP.
package cep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Os erros que o resto do sistema precisa distinguir. O handler os traduz em
// status HTTP; nada fora daqui conhece o formato da resposta do ViaCEP.
var (
	// ErrFormatoInvalido: nao sao oito digitos. Vira 422 na borda.
	ErrFormatoInvalido = errors.New("invalid zipcode")
	// ErrNaoEncontrado: formato bom, CEP inexistente. Vira 404 na borda.
	ErrNaoEncontrado = errors.New("can not find zipcode")
	// ErrIndisponivel: a consulta em si falhou (rede, 5xx, corpo ilegivel).
	ErrIndisponivel = errors.New("cep lookup unavailable")
)

// oitoDigitos e' o formato literal exigido pelo desafio: oito digitos, nada
// mais. Hifen e espaco contam como caractere invalido, nao como ruido a limpar.
var oitoDigitos = regexp.MustCompile(`^[0-9]{8}$`)

// Validar diz se o CEP tem o formato aceito, sem tocar na rede.
func Validar(cep string) error {
	if !oitoDigitos.MatchString(cep) {
		return fmt.Errorf("cep %q: %w", cep, ErrFormatoInvalido)
	}
	return nil
}

// Endereco e' o recorte do ViaCEP que este sistema usa.
type Endereco struct {
	CEP        string
	Localidade string
	UF         string
}

const baseURLPadrao = "https://viacep.com.br/ws"

// Client consulta o ViaCEP.
type Client struct {
	baseURL string
	http    *http.Client
}

// Option ajusta o Client na construcao.
type Option func(*Client)

// WithBaseURL troca o endereco do ViaCEP — e' o que permite apontar para um
// httptest.Server no teste sem mexer em variavel global.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimSuffix(u, "/") }
}

// WithHTTPClient troca o cliente HTTP usado nas chamadas.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// NewClient monta um Client com timeout proprio, para uma consulta lenta nao
// segurar a requisicao do usuario indefinidamente.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL: baseURLPadrao,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// respostaViaCEP espelha o corpo do ViaCEP. O campo erro chega ora como
// booleano, ora como a string "true", dependendo da versao — por isso ele e'
// json.RawMessage e nao bool: decodificar em bool quebra na string.
type respostaViaCEP struct {
	CEP        string          `json:"cep"`
	Localidade string          `json:"localidade"`
	UF         string          `json:"uf"`
	Erro       json.RawMessage `json:"erro"`
}

// naoEncontrado interpreta o campo erro nas duas formas que a API ja usou.
func (r respostaViaCEP) naoEncontrado() bool {
	switch strings.Trim(string(r.Erro), `"`) {
	case "true":
		return true
	}
	// Sem o campo erro, cidade vazia significa a mesma coisa: nao da' para
	// seguir para o clima.
	return r.Localidade == ""
}

// Buscar valida o formato e resolve o CEP em cidade e UF.
//
// O ViaCEP responde 200 OK mesmo para CEP inexistente, sinalizando a ausencia
// no corpo. Por isso o status sozinho nao decide nada aqui.
func (c *Client) Buscar(ctx context.Context, cep string) (Endereco, error) {
	if err := Validar(cep); err != nil {
		return Endereco{}, err
	}

	url := fmt.Sprintf("%s/%s/json/", c.baseURL, cep)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Endereco{}, fmt.Errorf("montar requisicao: %w: %v", ErrIndisponivel, err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Endereco{}, fmt.Errorf("consultar viacep: %w: %v", ErrIndisponivel, err)
	}
	defer resp.Body.Close()

	// 400 e' como o ViaCEP recusa um CEP que ele considera malformado. Para
	// quem chama, isso e' indistinguivel de inexistente.
	if resp.StatusCode == http.StatusBadRequest {
		return Endereco{}, fmt.Errorf("cep %q: %w", cep, ErrNaoEncontrado)
	}
	if resp.StatusCode != http.StatusOK {
		return Endereco{}, fmt.Errorf("viacep respondeu %d: %w", resp.StatusCode, ErrIndisponivel)
	}

	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Endereco{}, fmt.Errorf("ler resposta do viacep: %w: %v", ErrIndisponivel, err)
	}

	var r respostaViaCEP
	if err := json.Unmarshal(corpo, &r); err != nil {
		return Endereco{}, fmt.Errorf("decodificar resposta do viacep: %w: %v", ErrIndisponivel, err)
	}
	if r.naoEncontrado() {
		return Endereco{}, fmt.Errorf("cep %q: %w", cep, ErrNaoEncontrado)
	}

	return Endereco{CEP: r.CEP, Localidade: r.Localidade, UF: r.UF}, nil
}
