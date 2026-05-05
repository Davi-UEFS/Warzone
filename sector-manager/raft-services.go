package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	raft "github.com/hashicorp/raft"
)

const (
	OP_ADDI      = "ADD_INCIDENT"
	OP_RMVI      = "REMOVE_INCIDENT"
	OP_ASSIGN    = "ASSIGN_DRONE"
	OP_DEASSIGN  = "DEASSIGN_DRONE"
	OP_UPDATEDRB = "UPDATE_DRONE_BROKER"
)

type RaftCommand struct {
	Operation string
	Payload   json.RawMessage
}

type RaftFSM struct {
	Mu           sync.Mutex
	DroneMap     map[string]shared.Drone
	IncidentList []shared.Incident //TODO: IMPLEMENTAR PRIORITY QUEUE DEPOIS
}

type RaftSnapshot struct {
	DroneMap     map[string]shared.Drone
	IncidentList []shared.Incident
}

func (fsm *RaftFSM) Apply(log *raft.Log) interface{} {

	var cmd RaftCommand

	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		fmt.Printf("Erro ao desserializar comando: %v.\n", err)
		return err
	}

	switch cmd.Operation {
	case OP_ADDI:
		return fsm.handleADDIncident(cmd.Payload)

	case OP_RMVI:
		return fsm.handleRMVIncident(cmd.Payload)

	case OP_ASSIGN:
		return fsm.handleASSIGNDrone(cmd.Payload)

	case OP_DEASSIGN:
		return fsm.handleDEASSIGNDrone(cmd.Payload)

	case OP_UPDATEDRB:
		return fsm.handleUPDATEDRBroker(cmd.Payload)
	default:
		fmt.Printf("Operação desconhecida: %s\n", cmd.Operation)
		return nil //TODO: TRATAR MELHOR ESSA SITUAÇÃO DE OPERAÇÃO DESCONHECIDA
	}

}

func (fsm *RaftFSM) handleADDIncident(payload json.RawMessage) error {
	var incident shared.Incident

	if err := json.Unmarshal(payload, &incident); err != nil {
		fmt.Printf("Erro ao desserializar pacote: %v.\n", err)
		return err
	}
	fsm.Mu.Lock()

	for _, v := range fsm.IncidentList {
		if v.ID == incident.ID {
			fmt.Printf("Incidente %s já existe.\n", v.ID)
			fsm.Mu.Unlock()
			return nil
		}
	}
	fsm.IncidentList = append(fsm.IncidentList, incident) //TODO: IMPLEMENTAR PRIORITY QUEUE DEPOIS
	fsm.Mu.Unlock()

	return nil
}

func (fsm *RaftFSM) handleRMVIncident(payload json.RawMessage) error {
	var incident shared.Incident

	if err := json.Unmarshal(payload, &incident); err != nil {
		fmt.Printf("Erro ao desserializar pacote: %v.\n", err)
		return err
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	for i, v := range fsm.IncidentList { //TODO: IMPLEMENTAR PRIORITY QUEUE DEPOIS
		if v.ID == incident.ID {
			fsm.IncidentList = append(fsm.IncidentList[:i], fsm.IncidentList[i+1:]...)
			break
		}
	}

	return nil
}

func (fsm *RaftFSM) handleASSIGNDrone(payload json.RawMessage) error {
	var req shared.Requisition
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("erro unmarshal: %w", err)
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	drone, ok := fsm.DroneMap[req.DroneID]
	if !ok {
		fmt.Printf("Abortando: Drone %s não mapeado na FSM.\n", req.DroneID)
		return fmt.Errorf("drone não encontrado")
	}

	foundIndex := -1
	for i, incident := range fsm.IncidentList {
		if incident.ID == req.OccurrenceID {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		fmt.Printf("Abortando: Incidente %s já foi removido ou não existe.\n", req.OccurrenceID)
		return nil
	}

	drone.SetBusy()
	fsm.DroneMap[req.DroneID] = drone
	fsm.IncidentList = slices.Delete(fsm.IncidentList, foundIndex, foundIndex+1)

	fmt.Printf("Sucesso: Drone %s alocado para Incidente %s.\n", req.DroneID, req.OccurrenceID)
	return nil
}

func (fsm *RaftFSM) handleDEASSIGNDrone(payload json.RawMessage) error {

	var droneID string

	if err := json.Unmarshal(payload, &droneID); err != nil {
		fmt.Printf("Erro ao desserializar pacote: %v.\n", err)
		return err
	}

	fsm.Mu.Lock()
	if drone, ok := fsm.DroneMap[droneID]; ok {
		drone.SetIdle()
		fsm.DroneMap[droneID] = drone
	} else {
		fmt.Printf("Drone %s não encontrado.\n", droneID)
	}
	fsm.Mu.Unlock()

	return nil
}

func (fsm *RaftFSM) handleUPDATEDRBroker(payload json.RawMessage) error {

	var values struct {
		DroneID  string `json:"drone_id"`
		BrokerID string `json:"broker_id"`
	}

	if err := json.Unmarshal(payload, &values); err != nil {
		fmt.Printf("Erro ao desserializar pacote: %v.\n", err)
		return err
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	if drone, ok := fsm.DroneMap[values.DroneID]; ok {
		drone.UpdateBroker(values.BrokerID)
		fsm.DroneMap[values.DroneID] = drone
	} else {
		fmt.Printf("Drone %s não encontrado.\n", values.DroneID)
	}

	return nil
}

func (fsm *RaftFSM) Snapshot() (raft.FSMSnapshot, error) {
	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	clonedDrones := make(map[string]shared.Drone)
	for k, v := range fsm.DroneMap {
		clonedDrones[k] = v
	}

	clonedIncidents := slices.Clone(fsm.IncidentList)

	snapshot := &RaftSnapshot{
		DroneMap:     clonedDrones,
		IncidentList: clonedIncidents,
	}
	return snapshot, nil
}

func (snapshot *RaftSnapshot) Persist(sink raft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(snapshot)
	if err != nil {
		sink.Cancel()
		return fmt.Errorf("erro ao codificar snapshot: %v", err)
	}

	if err := sink.Close(); err != nil {
		return fmt.Errorf("erro ao fechar snapshot sink: %v", err)
	}

	return nil
}

func (snaphot *RaftSnapshot) Release() {
}

func (fsm *RaftFSM) Restore(rc io.ReadCloser) error {

	var restored RaftSnapshot
	if err := json.NewDecoder(rc).Decode(&restored); err != nil {
		return fmt.Errorf("erro ao decodificar snapshot: %v", err)
	}

	fsm.Mu.Lock()
	fsm.DroneMap = restored.DroneMap
	fsm.IncidentList = restored.IncidentList
	fsm.Mu.Unlock()

	fmt.Printf("FSM Restaurada: %d drones e %d incidentes carregados.\n",
		len(fsm.DroneMap), len(fsm.IncidentList))

	return nil
}

type LeaderInfo struct {
	RaftAddr   string `json:"raft_addr"`
	BrokerAddr string `json:"broker_addr"`
}

func searchForLeaderInfo(peers []string, sigPort int) LeaderInfo {
	for _, peer := range peers {
		addr := normalizePeerAddr(peer, sigPort)
		if addr == "" {
			continue
		}
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			continue
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		var leaderInfo LeaderInfo

		if scanner.Scan() {
			if err := json.Unmarshal(scanner.Bytes(), &leaderInfo); err == nil && leaderInfo.RaftAddr != "" {
				return leaderInfo
			}
		}
	}
	return LeaderInfo{}
}

func startSignalingServer(raftNode *raft.Raft, sigAddr, brokerAddr string) {
	ln, _ := net.Listen("tcp", sigAddr)
	for {
		conn, _ := ln.Accept()
		go func(c net.Conn) {
			defer c.Close()
			leaderInfo := LeaderInfo{
				RaftAddr:   string(raftNode.Leader()),
				BrokerAddr: brokerAddr,
			}
			json.NewEncoder(c).Encode(leaderInfo)
		}(conn)
	}
}
