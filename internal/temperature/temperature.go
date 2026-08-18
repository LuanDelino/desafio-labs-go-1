// Package temperature converte uma leitura em Celsius para as escalas do contrato.
package temperature

import "math"

// O enunciado manda C + 273, e não o 273.15 físico. Trocar aqui muda tudo.
const deslocamentoKelvin = 273

// Reading é a mesma temperatura nas três escalas.
type Reading struct {
	Celsius    float64
	Fahrenheit float64
	Kelvin     float64
}

// Sem arredondar, 28.5*1.8+32 vira 83.30000000000001 e a cauda vaza no JSON.
func arredondar(v float64) float64 {
	return math.Round(v*100) / 100
}

// FromCelsius aplica as fórmulas do desafio a uma leitura em Celsius.
func FromCelsius(c float64) Reading {
	return Reading{
		Celsius:    arredondar(c),
		Fahrenheit: arredondar(c*1.8 + 32),
		Kelvin:     arredondar(c + deslocamentoKelvin),
	}
}
