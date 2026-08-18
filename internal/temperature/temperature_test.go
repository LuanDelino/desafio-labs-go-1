package temperature

import "testing"

func TestFromCelsius(t *testing.T) {
	casos := []struct {
		nome    string
		celsius float64
		f       float64
		k       float64
	}{
		// O enunciado exemplifica 301.65 (C+273.15) mas manda C+273.
		// Seguimos a fórmula; este caso registra a divergência.
		{"exemplo do enunciado", 28.5, 83.3, 301.5},
		{"zero", 0, 32, 273},
		{"negativo", -10, 14, 263},
		{"fracionado", 21.7, 71.06, 294.7},
		{"zero absoluto pela formula", -273, -459.4, 0},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := FromCelsius(c.celsius)
			if !quase(got.Celsius, c.celsius) {
				t.Errorf("Celsius = %v, quero %v", got.Celsius, c.celsius)
			}
			if !quase(got.Fahrenheit, c.f) {
				t.Errorf("Fahrenheit = %v, quero %v", got.Fahrenheit, c.f)
			}
			if !quase(got.Kelvin, c.k) {
				t.Errorf("Kelvin = %v, quero %v", got.Kelvin, c.k)
			}
		})
	}
}

func quase(a, b float64) bool {
	const tolerancia = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < tolerancia
}
