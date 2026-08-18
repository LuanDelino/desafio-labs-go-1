// Package temperature converte uma leitura em Celsius para as três escalas
// que o contrato da API expõe.
package temperature

import "math"

// deslocamentoKelvin é o 273 da fórmula do desafio (K = C + 273), e não o
// 273.15 do valor físico. Está isolado aqui de propósito: se a correção
// passar a exigir o valor físico, esta linha é a única que muda.
const deslocamentoKelvin = 273

// Reading é a mesma temperatura nas três escalas.
type Reading struct {
	Celsius    float64
	Fahrenheit float64
	Kelvin     float64
}

// casasDecimais e' a precisao com que as temperaturas sao publicadas. Existe
// porque float64 nao representa 83.3 exatamente: 28.5*1.8+32 rende
// 83.30000000000001, e essa cauda vazaria no JSON, divergindo do contrato.
const casasDecimais = 2

func arredondar(v float64) float64 {
	const fator = 100 // 10^casasDecimais
	return math.Round(v*fator) / fator
}

// FromCelsius aplica as fórmulas do desafio a uma leitura em Celsius.
func FromCelsius(c float64) Reading {
	return Reading{
		Celsius:    arredondar(c),
		Fahrenheit: arredondar(c*1.8 + 32),
		Kelvin:     arredondar(c + deslocamentoKelvin),
	}
}
