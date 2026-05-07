package main

import "math/rand"

// Retorna incidente com chance de 10%
func generateIncident() bool {
	chance := rand.Float64()
	if chance <= 0.1 {
		return true
	}

	return false

}
