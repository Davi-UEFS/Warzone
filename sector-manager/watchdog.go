package main

import (
	"fmt"
	"os"
	"time"
)

// RunWatchdog varre a memória em busca de drones inativos e remove-os
func RunWatchdog() {
	now := time.Now().Unix()

	// 1. Fase de Identificação (Entramos na RAM, anotamos os mortos e saímos rápido)
	var deadDrones []string

	GlobalState.Mu.Lock()
	for id, drone := range GlobalState.DroneMap {
		// Somente o setor pode remover o drone, pois ele é o único que sabe se o drone está ativo ou não.
		if now-drone.LastSeen > 20 && drone.CurrentSector == os.Getenv("SECTOR_ID") {
			deadDrones = append(deadDrones, id)
		}
	}
	GlobalState.Mu.Unlock()

	// 2. Fase de Execução (O sistema continua a rodar livremente enquanto matamos estes)
	for _, id := range deadDrones {
		fmt.Printf("\033[1;31m[WATCHDOG]\033[0m Drone %s considerado offline. Último sinal expirou.\n", id)

		// Usamos o nosso novo método seguro do state.go para jogar o drone no cemitério
		GlobalState.BuryDrone(id, time.Now().Unix())

		// Dispara a transação para a blockchain.
		go enviarReportDeadDroneParaBlockchain(id)
	}
}
