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

var (
	ErrFormatoInvalido = errors.New("invalid zipcode")
	ErrNaoEncontrado   = errors.New("can not find zipcode")
	ErrIndisponivel    = errors.New("cep lookup unavailable")
)

var oitoDigitos = regexp.MustCompile(`^[0-9]{8}$`)

// Validar diz se o CEP tem o formato aceito, sem tocar na rede.
func Validar(cep string) error {
	if !oitoDigitos.MatchString(cep) {
		return fmt.Errorf("cep %q: %w", cep, ErrFormatoInvalido)
	}
	return nil
}

// Endereco é o recorte do ViaCEP que este sistema usa.
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

// Option ajusta o Client na construção.
type Option func(*Client)

// WithBaseURL troca o endereço do ViaCEP.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimSuffix(u, "/") }
}

// WithHTTPClient troca o cliente HTTP usado nas chamadas.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// NewClient monta um Client com timeout próprio.
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

type respostaViaCEP struct {
	CEP        string `json:"cep"`
	Localidade string `json:"localidade"`
	UF         string `json:"uf"`
	// RawMessage porque o campo já veio como booleano e como a string "true".
	Erro json.RawMessage `json:"erro"`
}

func (r respostaViaCEP) naoEncontrado() bool {
	if strings.Trim(string(r.Erro), `"`) == "true" {
		return true
	}
	return r.Localidade == ""
}

// Buscar valida o formato e resolve o CEP em cidade e UF.
//
// O ViaCEP responde 200 OK para CEP inexistente e sinaliza a ausência no
// corpo, então o status sozinho não decide nada aqui.
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

	// 400 é como o ViaCEP recusa um CEP malformado: para quem chama, o mesmo
	// que inexistente.
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
