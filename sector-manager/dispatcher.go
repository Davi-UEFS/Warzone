package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// ProcessRequisitions cruza missões pendentes com drones livres na RAM
func ProcessRequisitions() {
	GlobalState.Mu.Lock()

	// 1. Há missões na fila?
	if len(GlobalState.PendingReqsQueue) == 0 {
		GlobalState.Mu.Unlock()
		return // Silêncio, nada a fazer.
	}

	req := GlobalState.PendingReqsQueue.Peek()

	// Proteção contra duplo despacho
	if _, alreadyDispatched := GlobalState.DispatchedSet[req.ID]; alreadyDispatched {
		GlobalState.PendingReqsQueue.Pop()
		GlobalState.Mu.Unlock()
		return
	}

	var freeDroneID string = ""
	now := time.Now().Unix()

	// 2. Busca o drone perfeito diretamente no nosso Cofre de RAM
	for id, localDrone := range GlobalState.DroneMap {
		isIdle := localDrone.Status == shared.DRONE_IDLE
		isAlive := (now - localDrone.LastSeen) <= 20

		if isIdle && localDrone.Verified && isAlive {
			freeDroneID = id
			break
		}
	}

	// 3. Se achou um drone, aplica o bloqueio otimista e remove a missão da fila
	if freeDroneID != "" {

		// Tranca o drone imediatamente para que o próximo ciclo não o use!
		GlobalState.DroneMap[freeDroneID].Status = shared.DRONE_BUSY
		GlobalState.PendingReqsQueue.Pop()
		GlobalState.DispatchedSet[req.ID] = time.Now().Unix()
		GlobalState.Mu.Unlock() // Liberta a RAM o mais rápido possível

		// TODO: DEBUG
		log.Printf("\033[1;35m[DEBUG DESPACHO]\033[0m Missão %s alocada para o drone %s\n", req.ID, freeDroneID)

		// 4. Despacha a missão (Físico + Blockchain)
		dispatch(freeDroneID, req)
		return
	}

	GlobalState.Mu.Unlock()
}

// dispatch cria o payload, envia para o MQTT e avisa a Blockchain
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

	// globalClient é o cliente MQTT configurado no seu vars.go / init
	token := globalClient.Publish(topic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("\033[1;31m[DISPATCHER]\033[0m Erro ao enviar missão via MQTT: %v\n", token.Error())

		// Se o rádio falhou, devolvemos o drone e a missão para tentarem no próximo ciclo
		GlobalState.Mu.Lock()
		if drone, ok := GlobalState.DroneMap[droneID]; ok {
			drone.Status = shared.DRONE_IDLE
		}
		GlobalState.PendingReqsQueue.Push(&req)
		GlobalState.Mu.Unlock()
		return
	}

	fmt.Printf("\033[1;32m[DISPATCHER]\033[0m Missão %s despachada fisicamente para o drone %s!\n", req.ID, droneID)

	// Dispara a transação assíncrona para selar o contrato na Blockchain
	go enviarAssignDroneParaBlockchain(req.ID, droneID)
}
