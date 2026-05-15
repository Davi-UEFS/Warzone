package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	raft "github.com/hashicorp/raft"
)

func startDispatcher() {
	ticker := time.NewTicker(2 * time.Second)
	watchdogTicker := time.NewTicker(10 * time.Second) // O Cão de Guarda roda a cada 10s
	agingTicker := time.NewTicker(20 * time.Second)    // Aging a cada 20s

	for {
		select {
		case <-ticker.C:
			// Apenas o Líder despacha novas missões
			if raftNode.State() == raft.Leader {
				processRequisitions()
			}
		case <-watchdogTicker.C:
			// Apenas o Líder caça os drones caídos
			if raftNode.State() == raft.Leader {
				checkDeadDrones()
			}
		case <-agingTicker.C:
			// Apenas o Líder aplica aging (será replicado via Raft)
			if raftNode.State() == raft.Leader {
				applyAging()
			}
		}
	}
}

func processRequisitions() {
	sectorFSM.Mu.Lock()

	if len(sectorFSM.PendingReqsQueue) == 0 {
		sectorFSM.Mu.Unlock()
		return
	}

	//TODO: Talvez seja melhor pedir para o Raft me dar um drone inves de iterar aqui.
	var freeDroneID string

	for id, drone := range sectorFSM.DroneMap {
		if drone.Status == shared.DRONE_IDLE {
			freeDroneID = id
			break
		}
	}

	if freeDroneID != "" {
		req := sectorFSM.PendingReqsQueue.Peek()
		sectorFSM.Mu.Unlock()

		dispatch(raftNode, freeDroneID, req)
	} else {
		sectorFSM.Mu.Unlock()
	}
}

func dispatch(raftNode *raft.Raft, droneID string, req shared.Requisition) {
	LClock.Tick()

	mission := shared.DroneMission{
		RequisitionID: req.ID,
		AssignedDrone: droneID,
		Type:          req.Type,
		Coordinate:    req.Coord,
		LamportTime:   LClock.GetTime(),
	}

	payload, _ := json.Marshal(mission)

	cmd := shared.HeaderCommand{
		Operation:   OP_ASSIGN,
		Payload:     payload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		fmt.Printf("Erro ao aplicar comando ASSIGN no Raft: %v\n", err)
	}
}

// applyAging envia comando OP_AGING via Raft para envelhecer requisições
func applyAging() {
	LClock.Tick()

	sectorFSM.Mu.Lock()
	if len(sectorFSM.PendingReqsQueue) == 0 {
		sectorFSM.Mu.Unlock()
		return
	}
	sectorFSM.Mu.Unlock()

	payload := []byte(`{}`) // Aging não precisa de payload, mas ainda preciso mandar um RawMEssage válido

	cmd := shared.HeaderCommand{
		Operation:   OP_AGING,
		Payload:     payload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, err := json.Marshal(cmd)

	if err != nil {
		fmt.Printf("Erro ao serializar comando AGING: %v\n", err)
		return
	}

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		fmt.Printf("Erro ao aplicar comando AGING no Raft: %v\n", err)
	}
}

// --- LÓGICA DO WATCHDOG (CÃO DE GUARDA) ---
func checkDeadDrones() {
	now := time.Now().Unix()

	sectorFSM.Mu.Lock()
	var deadDrones []string

	// Procura drones que não mandam o heartbeat há mais de 15 segundos
	for id, drone := range sectorFSM.DroneMap {
		if now-drone.LastSeen > 15 {
			deadDrones = append(deadDrones, id)
		}
	}
	sectorFSM.Mu.Unlock()

	// Para cada drone inativo, avisa o Raft para matá-lo e resgatar a missão
	for _, id := range deadDrones {
		LClock.Tick()
		payload, _ := json.Marshal(id)

		cmd := shared.HeaderCommand{
			Operation:   OP_DEADDRONE,
			Payload:     payload,
			LamportTime: LClock.GetTime(),
		}

		cmdBytes, _ := json.Marshal(cmd)

		// Envia a ordem de morte para a FSM (o handleDEADDrone fará o resgate da requisição)
		raftNode.Apply(cmdBytes, 5*time.Second)
	}
}
