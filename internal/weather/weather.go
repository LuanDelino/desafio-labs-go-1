// Package weather consulta a temperatura atual de uma localidade na WeatherAPI.
package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrLocalNaoEncontrado: a WeatherAPI nao reconheceu a localidade.
	ErrLocalNaoEncontrado = errors.New("weather location not found")
	// ErrIndisponivel: a consulta falhou (rede, chave recusada, 5xx, corpo ilegivel).
	ErrIndisponivel = errors.New("weather lookup unavailable")
	// ErrSemChave: o servico subiu sem WEATHER_API configurada.
	ErrSemChave = errors.New("weather api key not configured")
)

// Local e' o endereco no vocabulario desta consulta. E' um tipo proprio, e nao
// o Endereco do pacote cep, para que weather nao dependa de cep: os dois sao
// folhas independentes e quem costura e' a borda HTTP.
type Local struct {
	Cidade string
	UF     string
	Pais   string
}

// consulta monta o termo de busca. Cidade sozinha e' ambigua — ha' mais de uma
// Santa Maria no Brasil — entao UF e pais entram sempre que existirem.
func (l Local) consulta() string {
	partes := []string{l.Cidade}
	if l.UF != "" {
		partes = append(partes, l.UF)
	}
	pais := l.Pais
	if pais == "" {
		pais = "Brazil"
	}
	return strings.Join(append(partes, pais), ",")
}

const baseURLPadrao = "https://api.weatherapi.com/v1"

// codigoLocalDesconhecido e' o codigo com que a WeatherAPI sinaliza localidade
// inexistente, dentro de uma resposta 400.
const codigoLocalDesconhecido = 1006

// Client consulta a WeatherAPI.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// Option ajusta o Client na construcao.
type Option func(*Client)

// WithBaseURL aponta o cliente para outro endereco — usado pelo httptest.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimSuffix(u, "/") }
}

// WithHTTPClient troca o cliente HTTP usado nas chamadas.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// NewClient monta um Client para a chave informada.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURLPadrao,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

type respostaWeather struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Atual devolve a temperatura em Celsius da localidade informada.
func (c *Client) Atual(ctx context.Context, l Local) (float64, error) {
	if c.apiKey == "" {
		return 0, ErrSemChave
	}

	q := url.Values{}
	q.Set("key", c.apiKey)
	q.Set("q", l.consulta()) // Encode() cuida do acento e da virgula.
	q.Set("aqi", "no")
	endereco := fmt.Sprintf("%s/current.json?%s", c.baseURL, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return 0, fmt.Errorf("montar requisicao: %w: %v", ErrIndisponivel, err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("consultar weatherapi: %w: %v", ErrIndisponivel, err)
	}
	defer resp.Body.Close()

	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, fmt.Errorf("ler resposta da weatherapi: %w: %v", ErrIndisponivel, err)
	}

	var r respostaWeather
	erroDeDecode := json.Unmarshal(corpo, &r)

	if resp.StatusCode != http.StatusOK {
		// So' o 1006 significa "essa cidade nao existe". Chave recusada ou
		// cota estourada tambem chegam como 4xx e nao podem virar 404 para o
		// usuario — sao falha nossa, nao CEP invalido.
		if erroDeDecode == nil && r.Error.Code == codigoLocalDesconhecido {
			return 0, fmt.Errorf("localidade %q: %w", l.consulta(), ErrLocalNaoEncontrado)
		}
		return 0, fmt.Errorf("weatherapi respondeu %d (%s): %w",
			resp.StatusCode, r.Error.Message, ErrIndisponivel)
	}
	if erroDeDecode != nil {
		return 0, fmt.Errorf("decodificar resposta da weatherapi: %w: %v", ErrIndisponivel, erroDeDecode)
	}

	return r.Current.TempC, nil
}
