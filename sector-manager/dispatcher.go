package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	raft "github.com/hashicorp/raft"
)

// startDispatcher inicia as rotinas de despacho de missões, aging e watchdog.
//
//	Apenas o líder executa as ações.
func startDispatcher() {
	watchdogTicker := time.NewTicker(5 * time.Second) // O Cão de Guarda roda a cada 5s
	agingTicker := time.NewTicker(20 * time.Second)   // Aging a cada 20s

	go func() {
		for {
			// Apenas o Líder despacha novas missões
			// Não utiliza ticker porque quer despachar assim que chegar uma nova requisição
			if raftNode.State() == raft.Leader {
				processRequisitions()
			}
		}
	}()

	for {

		select {

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

// processRequisitions verifica se há requisições pendentes e drones livres para despachar missões.
// Se encontrar um drone IDLE, despacha a missão no topo da fila de prioridades para ele.
// Deve ser chamado em loop para garantir que novas requisições sejam processadas assim que chegarem.
func processRequisitions() {
	sectorFSM.Mu.Lock()

	if len(sectorFSM.PendingReqsQueue) == 0 {
		sectorFSM.Mu.Unlock()
		return
	}

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

// dispatch cria e envia a missão para o drone escolhido via Raft, garantindo que a ação seja replicada e consistente entre os nós.
// O comando OP_ASSIGN é processado pela FSM para atualizar o mapa de drones.
//
// Incrementa o relógio de Lamport.
//
// Params:
//   - raftNode: instância do Raft para enviar o comando
//   - droneID: ID do drone que receberá a missão
//   - req: requisição que será transformada em missão e enviada para o drone
func dispatch(raftNode *raft.Raft, droneID string, req shared.Requisition) {
	LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Despachando Missão para Drone\n", LClock.GetTime())
	}

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
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao aplicar comando ASSIGN no Raft: %v\n", err)
	}
}

// applyAging envia comando OP_AGING via Raft para envelhecer requisições.
//
// Incrementa o relógio de Lamport se houver requisições pendentes para envelhecer.
func applyAging() {

	sectorFSM.Mu.Lock()
	if len(sectorFSM.PendingReqsQueue) == 0 {
		sectorFSM.Mu.Unlock()
		return
	}
	LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Aplicando Aging na Fila de Prioridades\n", LClock.GetTime())
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
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao serializar comando AGING: %v\n", err)
		return
	}

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao aplicar comando AGING no Raft: %v\n", err)
	}
}

// checkDeadDrones verifica se há drones que não enviaram heartbeat há mais de 20 segundos e os declara como mortos,
// enviando um comando OP_DEADDRONE via Raft para que a FSM possa resgatar a missão e liberar o drone.
//
// Incrementa relógio de Lamport se houver drones mortos para declarar.
func checkDeadDrones() {
	now := time.Now().Unix()

	sectorFSM.Mu.Lock()
	var deadDrones []string

	// Procura drones que não mandam o heartbeat há mais de 20 segundos
	for id, drone := range sectorFSM.DroneMap {
		if now-drone.LastSeen > 20 {
			deadDrones = append(deadDrones, id)
		}
	}

	sectorFSM.Mu.Unlock()

	// Para cada drone inativo, avisa o Raft para matá-lo e resgatar a missão
	for _, id := range deadDrones {
		LClock.Tick()

		// TODO: DEBUG_MODE_LAMPORT_TICK
		if DebugMode {
			fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Declarando Drone Morto (Watchdog)\n", LClock.GetTime())
		}

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
