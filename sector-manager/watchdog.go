package main

import (
	"fmt"
	"os"
	"time"
)

// RunWatchdog varre a memória em busca de drones sem heartbeat e os remove.
func RunWatchdog() {
	now := time.Now().Unix()

	// Drones marcados para remoção.
	var deadDrones []string

	GlobalState.Mu.Lock()
	for id, drone := range GlobalState.DroneMap {
		// Verificamos se o drone está neste setor e passou 20s sem sinal.
		// Somente o setor pode remover o drone, pois os outros não recebem o heartbeat.
		if now-drone.LastSeen > 20 && drone.CurrentSector == os.Getenv("SECTOR_ID") {
			deadDrones = append(deadDrones, id)
		}
	}
	GlobalState.Mu.Unlock()

	for _, id := range deadDrones {
		fmt.Printf("\033[1;31m[WATCHDOG]\033[0m Drone %s considerado offline. Último sinal expirou.\n", id)

		// Enterramos o drone na memória local. Isso impede que ele seja reusado pelo Poller.
		GlobalState.BuryDrone(id, time.Now().Unix())

		// Dispara a transação para a blockchain.
		go enviarReportDeadDroneParaBlockchain(id)
	}
}
