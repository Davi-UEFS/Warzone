package main

import (
	"fmt"
	"time"
)

// RunWatchdog varre a memória em busca de drones inativos e remove-os
func RunWatchdog() {
	now := time.Now().Unix()

	// 1. Fase de Identificação (Entramos na RAM, anotamos os mortos e saímos rápido)
	var deadDrones []string

	GlobalState.Mu.Lock()
	for id, drone := range GlobalState.DroneMap {
		if now-drone.LastSeen > 20 {
			deadDrones = append(deadDrones, id)
		}
	}
	GlobalState.Mu.Unlock() // Destrancamos a RAM imediatamente!

	// 2. Fase de Execução (O sistema continua a rodar livremente enquanto matamos estes)
	for _, id := range deadDrones {
		fmt.Printf("\033[1;31m[WATCHDOG]\033[0m Drone %s considerado offline. Último sinal expirou.\n", id)

		// Usamos o nosso novo método seguro do state.go para jogar o drone no cemitério
		GlobalState.BuryDrone(id, time.Now().Unix())

		// Dispara a transação para a blockchain.
		// Lembra-se? Já temos o txMutex seguro lá no blockchainclient.go para enfileirar isto perfeitamente!
		go enviarReportDeadDroneParaBlockchain(id)
	}
}
