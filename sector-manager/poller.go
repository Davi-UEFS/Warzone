package main

import (
	"fmt"
	"time"
)

// SyncStateWithBlockchain puxa a verdade global e atualiza a RAM com segurança
func SyncStateWithBlockchain() {
	// 1. Atualiza as Missões (Requisições)
	reqs, err := fetchRequisitionsFromBlockchain()
	if err != nil {
		fmt.Printf("\033[1;31m[POLLER]\033[0m Falha ao buscar missões REST: %v\n", err)
	} else {
		GlobalState.Mu.Lock()
		GlobalState.PendingReqsQueue.FromSlice(reqs)
		GlobalState.Mu.Unlock()
	}

	// 2. Atualiza os Drones
	drones, err := fetchDronesFromBlockchain()
	if err != nil {
		fmt.Printf("\033[1;31m[POLLER]\033[0m Falha ao buscar drones REST: %v\n", err)
		return
	}

	now := time.Now().Unix()

	for _, blockDrone := range drones {
		// A MÁGICA DA NOVA ARQUITETURA: O Portão do Cemitério
		// Usamos o método limpo do state.go. Se ele for um fantasma recente, a iteração salta-o!
		if GlobalState.IsGhost(blockDrone.ID, now) {
			continue
		}

		GlobalState.Mu.Lock()
		_, exists := GlobalState.DroneMap[blockDrone.ID]

		// Só criamos na RAM se não existir.
		// A bateria e o status reais serão geridos pelo MQTT e pelo Dispatcher depois.
		if !exists {
			fmt.Printf("\033[1;36m[POLLER]\033[0m Sincronizando drone %s da Blockchain para a RAM local...\n", blockDrone.ID)

			novoDrone := blockDrone
			novoDrone.LastSeen = now
			novoDrone.Verified = false // Nasce como fantasma até provar que está vivo pelo MQTT

			GlobalState.DroneMap[blockDrone.ID] = &novoDrone
		}
		GlobalState.Mu.Unlock()
	}
}
