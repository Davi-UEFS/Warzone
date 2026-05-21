package main

import (
	"math/rand"

	"github.com/Davi-UEFS/Warzone/shared"
)

// generateFireAlert dispara com 5% de chance, simulando a geração de um alerta de incêndio.
//
// Returns:
//   - bool: true se o alerta de incêndio for gerado, false caso contrário.
func generateFireAlert() bool {
	chance := rand.Float64()
	return chance <= 0.05

}

// generateOilAlert dispara com 15% de chance, simulando a geração de um alerta de vazamento de óleo.
//
// Returns:
//   - bool: true se o alerta de óleo for gerado, false caso contrário.
func generateOilAlert() bool {
	chance := rand.Float64()
	return chance <= 0.15

}

// generateWreckageAlert dispara com 20% de chance, simulando a geração de um alerta de destroços ao mar.
//
// Returns:
//   - bool: true se o alerta de destroços for gerado, false caso contrário.
func generateWreckageAlert() bool {
	chance := rand.Float64()
	return chance <= 0.20

}

// generateInspectionAlert dispara com 25% de chance, simulando a geração de um alerta de inspeção de segurança.
//
// Returns:
//   - bool: true se o alerta de inspeção for gerado, false caso contrário.
func generateInspectionAlert() bool {
	chance := rand.Float64()
	return chance <= 0.25

}

// generateUnknownObjectAlert dispara com 5% de chance, simulando a geração de um alerta de objeto desconhecido identificado.
//
// Returns:
//   - bool: true se o alerta de objeto desconhecido for gerado, false caso contrário.
func generateUnknownObjectAlert() bool {
	chance := rand.Float64()
	return chance <= 0.05
}

// generateBottleneckAlert dispara com 20% de chance, simulando a geração de um alerta de engarrafamento.
//
// Returns:
//   - bool: true se o alerta de engarrafamento for gerado, false caso contrário.
func generateBottleneckAlert() bool {
	chance := rand.Float64()
	return chance <= 0.20
}

// generateRandomCoordinate gera uma coordenada aleatória dentro de um limite pré-definido (0 a 500 para latitude e longitude).
func generateRandomCoordinate() shared.Coordinate {
	return shared.Coordinate{
		Latitude:  rand.Intn(500),
		Longitude: rand.Intn(500),
	}
}
