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
		log.Printf("\033[1;35m[DEBUG DESPACHO]\033[0m Tentando alocar missão %s para o drone %s na blockchain...\n", req.ID, freeDroneID)

		// 4. Despacha a missão (Blockchain PRIMEIRO, Físico depois)
		dispatch(freeDroneID, req)
		return
	}

	GlobalState.Mu.Unlock()
}

// dispatch cria o payload, avisa a Blockchain e, SE aprovado, envia para o MQTT
func dispatch(droneID string, req shared.Requisition) {
	// =========================================================================
	// 1. A BLOCKCHAIN ATUA COMO JUÍZA (Síncrono)
	// =========================================================================
	err := enviarAssignDroneParaBlockchain(req.ID, droneID)

	if err != nil {
		// Se deu erro (ex: drone já foi pego por outro Manager no mesmo milissegundo), aborta!
		fmt.Printf("\033[1;33m[DISPATCHER]\033[0m Perdeu a corrida pelo drone %s. Devolvendo missão para a fila.\n", droneID)

		GlobalState.Mu.Lock()
		if drone, ok := GlobalState.DroneMap[droneID]; ok {
			drone.Status = shared.DRONE_IDLE // Libera o drone localmente
		}
		GlobalState.PendingReqsQueue.Push(&req)   // Devolve a missão para a fila
		delete(GlobalState.DispatchedSet, req.ID) // Remove do set de despachados para não ser ignorada
		GlobalState.Mu.Unlock()
		return
	}

	// =========================================================================
	// 2. SÓ ENVIA O SINAL DE RÁDIO (MQTT) SE A BLOCKCHAIN AUTORIZOU
	// =========================================================================
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
		// Nota: Aqui a blockchain já marcou o drone como ocupado.
		// O poller futuramente pode sincronizar isso, ou o drone dar timeout.
	} else {
		fmt.Printf("\033[1;32m[DISPATCHER]\033[0m Missão %s despachada fisicamente para o drone %s!\n", req.ID, droneID)
	}
}
