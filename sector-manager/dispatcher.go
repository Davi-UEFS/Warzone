package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// startDispatcher inicia as rotinas de polling da blockchain e aging local.
func startDispatcher() {
	pollingTicker := time.NewTicker(2 * time.Second) // Vai à blockchain a cada 2s
	agingTicker := time.NewTicker(20 * time.Second)  // Aplica o Aging a cada 20s

	for {
		select {
		case <-pollingTicker.C:
			// 1. Busca missões na blockchain (A verdade global)
			reqs, err := fetchRequisitionsFromBlockchain()
			if err != nil {
				fmt.Printf("\033[1;31m[DISPATCHER]\033[0m Falha no polling de requisições: %v\n", err)
				continue
			}

			// 2. Updates local ReqHeap
			sectorState.Mu.Lock()
			sectorState.PendingReqsQueue.FromSlice(reqs)
			sectorState.Mu.Unlock()

			// 3. Tenta processar e despachar
			processRequisitions()

		case <-agingTicker.C:
			sectorState.Mu.Lock()
			sectorState.PendingReqsQueue.ApplyAging(time.Now().Unix(), 20, 1)
			sectorState.Mu.Unlock()

			processRequisitions()
		}
	}
}

// processRequisitions verifica se há requisições pendentes e drones livres.
func processRequisitions() {
	sectorState.Mu.Lock()
	if len(sectorState.PendingReqsQueue) == 0 {
		sectorState.Mu.Unlock()
		return
	}
	sectorState.Mu.Unlock()

	// Busca o censo global de drones na blockchain
	drones, err := fetchDronesFromBlockchain()
	if err != nil {
		fmt.Printf("\033[1;31m[DISPATCHER]\033[0m Falha ao buscar drones na blockchain: %v\n", err)
		return
	}

	var freeDroneID string = ""

	for _, drone := range drones {
		// 1. A Blockchain diz que ele está livre?
		if drone.Status == shared.DRONE_IDLE {

			// 2. Cross-check com o Heartbeat local (TTL)
			sectorState.Mu.Lock()
			localDrone, exists := sectorState.DroneMap[drone.ID]
			sectorState.Mu.Unlock()

			// Se o drone existe na memória e mandou sinal nos últimos 20 segundos
			if exists && (time.Now().Unix()-localDrone.LastSeen <= 20) {
				freeDroneID = drone.ID
				break
			}
		}
	}

	if freeDroneID != "" {
		sectorState.Mu.Lock()
		req := sectorState.PendingReqsQueue.Peek()
		sectorState.Mu.Unlock()

		// Despacha fisicamente via MQTT e atualiza a blockchain
		dispatch(freeDroneID, req)
	}
}

// dispatch cria o payload e publica no broker MQTT para o drone atuar.
func dispatch(droneID string, req shared.Requisition) {
	mission := shared.DroneMission{
		RequisitionID: req.ID,
		AssignedDrone: droneID,
		Type:          req.Type,
		Coordinate:    req.Coord,
		LamportTime:   0,
	}

	payload, err := json.Marshal(mission)
	if err != nil {
		fmt.Printf("\033[1;31m[DISPATCHER]\033[0m Erro ao serializar missão: %v\n", err)
		return
	}

	topic := fmt.Sprintf("drones/%s/mission", droneID)

	// globalClient configurado no vars.go/init
	token := globalClient.Publish(topic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("\033[1;31m[DISPATCHER]\033[0m Erro ao enviar missão via MQTT: %v\n", token.Error())
		return
	}

	fmt.Printf("\033[1;32m[DISPATCHER]\033[0m Missão %s despachada com sucesso para o drone %s!\n", req.ID, droneID)

	// Alteração preventiva rápida na RAM local para evitar corrida
	sectorState.Mu.Lock()
	if localDrone, exists := sectorState.DroneMap[droneID]; exists {
		localDrone.Status = shared.DRONE_BUSY
	}
	sectorState.Mu.Unlock()

	// Dispara a transação assíncrona para a blockchain
	go enviarAssignDroneParaBlockchain(droneID, req.ID)
}
