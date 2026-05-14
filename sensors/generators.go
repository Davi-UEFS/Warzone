package main

import (
	"math/rand"

	"github.com/Davi-UEFS/Warzone/shared"
)

// Retorna incidente com chance de 10%
func generateFireAlert() bool {
	chance := rand.Float64()
	if chance <= 0.05 {
		return true
	}

	return false

}

func generateOilAlert() bool {
	chance := rand.Float64()
	if chance <= 0.15 {
		return true
	}

	return false

}

func generateWreckageAlert() bool {
	chance := rand.Float64()
	if chance <= 0.20 {
		return true
	}

	return false

}

func generateInspectionAlert() bool {
	chance := rand.Float64()
	if chance <= 0.25 {
		return true
	}

	return false

}

func generateUnknownObjectAlert() bool {
	chance := rand.Float64()
	if chance <= 0.05 {
		return true
	}

	return false

}

func generateBottleneckAlert() bool {
	chance := rand.Float64()
	if chance <= 0.20 {
		return true
	}

	return false

}

func generateRandomCoordinate() shared.Coordinate {
	return shared.Coordinate{
		Latitude:  rand.Intn(500),
		Longitude: rand.Intn(500),
	}
}
