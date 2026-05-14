package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"
	raft "github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type RaftFSM struct {
	Mu               sync.Mutex
	DroneMap         map[string]shared.Drone
	PendingReqsQueue ReqHeap // priority queue for requisitions
	InProgressReqs   map[string]shared.Requisition
	Sector           string
	Client           mqtt.Client
}

type RaftSnapshot struct {
	DroneMap     map[string]shared.Drone
	IncidentList []shared.Requisition
	InProgress   map[string]shared.Requisition
}

func (fsm *RaftFSM) Apply(log *raft.Log) interface{} {

	var cmd shared.HeaderCommand

	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		fmt.Printf("Erro ao desserializar comando: %v.\n", err)
		fmt.Printf("Log data: %s\n", string(log.Data))
		return err
	}

	LClock.CompareAndUpdate(cmd.LamportTime)

	switch cmd.Operation {
	case OP_ADDREQ:
		return fsm.handleADDRequisition(cmd.Payload)

	case OP_RMVREQ:
		return fsm.handleRMVRequisition(cmd.Payload)

	case OP_ASSIGN:
		return fsm.handleASSIGNDrone(cmd.Payload)

	case OP_DEADDRONE:
		return fsm.handleDEADDrone(cmd.Payload)

	case OP_REGDRONE:
		return fsm.handleREGDrone(cmd.Payload)

	case OP_HEARTBEAT:
		return fsm.handleHEARTBEAT(cmd.Payload)
	case OP_AGING:
		return fsm.handleAGING()
	default:
		fmt.Printf("Operação desconhecida: %s\n", cmd.Operation)
		return nil //TODO: TRATAR MELHOR ESSA SITUAÇÃO DE OPERAÇÃO DESCONHECIDA
	}

}

func (fsm *RaftFSM) handleADDRequisition(payload json.RawMessage) error {
	var requisition shared.Requisition

	if err := json.Unmarshal(payload, &requisition); err != nil {
		fmt.Printf("Erro ao desserializar pacote: %v.\n", err)
		return err
	}
	fsm.Mu.Lock()

	// Evita duplicatas: checar tanto em Pending quanto em InProgress
	for _, v := range fsm.PendingReqsQueue {
		if v.ID == requisition.ID {
			fmt.Printf("Requisição %s já existe na fila pendente.\n", v.ID)
			fsm.Mu.Unlock()
			return nil
		}
	}

	if _, inProgress := fsm.InProgressReqs[requisition.ID]; inProgress {
		fmt.Printf("Requisição %s já está em progresso.\n", requisition.ID)
		fsm.Mu.Unlock()
		return nil
	}

	// push into priority queue
	heap.Push(&fsm.PendingReqsQueue, requisition)
	fsm.Mu.Unlock()

	return nil
}

func (fsm *RaftFSM) handleRMVRequisition(payload json.RawMessage) error {

	var doneInfo shared.DoneInfo

	if err := json.Unmarshal(payload, &doneInfo); err != nil {
		fmt.Printf("Erro ao desserializar pacote: %v.\n", err)
		return err
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	drone, ok := fsm.DroneMap[doneInfo.DroneID]

	if !ok {
		return fmt.Errorf("Drone %s não encontrado\n", doneInfo.DroneID)
	}

	reqID := doneInfo.RequisitionID

	if reqID == "" {
		// Sem missão atribuída; nada a remover
		fmt.Printf("Drone %s não possui missão atual.\n", doneInfo.DroneID)
		return nil
	}

	_, exist := fsm.InProgressReqs[reqID]

	if exist {

		LClock.Tick()

		// Nota: lógica de notificar sensor via MQTT removida — sensores não aguardam confirmação.

		// Primeiro liberta o drone localmente, depois limpa o registro da requisição
		drone.SetIdle()
		fsm.DroneMap[doneInfo.DroneID] = drone
		delete(fsm.InProgressReqs, doneInfo.RequisitionID)

		fmt.Printf("Requisição %s concluída pelo drone %s.\n", doneInfo.RequisitionID, doneInfo.DroneID)

	}

	return nil
}

func (fsm *RaftFSM) handleREGDrone(payload json.RawMessage) error {

	var newDrone shared.Drone

	if err := json.Unmarshal(payload, &newDrone); err != nil {
		return fmt.Errorf("erro unmarshal: %v", err)
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	newDrone.LastSeen = time.Now().Unix()

	if existingDrone, ok := fsm.DroneMap[newDrone.ID]; ok {
		newDrone.Status = existingDrone.Status
		newDrone.CurrentMission = existingDrone.CurrentMission

	} else {
		fmt.Printf("FSM: Novo drone registrado: %s (Broker: %s)\n", newDrone.ID, newDrone.CurrentBroker)
	}

	fsm.DroneMap[newDrone.ID] = newDrone

	return nil

}

func (fsm *RaftFSM) handleASSIGNDrone(payload json.RawMessage) error {
	var mission shared.DroneMission
	if err := json.Unmarshal(payload, &mission); err != nil {
		return fmt.Errorf("erro unmarshal: %v", err)
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	var targetReq shared.Requisition
	targetReqIndex := -1

	// Find target requisition index in heap underlying slice
	for i, req := range fsm.PendingReqsQueue {
		if req.ID == mission.RequisitionID {
			targetReq = req
			targetReqIndex = i
			break
		}
	}

	if targetReqIndex == -1 {
		fmt.Printf("Abortando: Requisição %s não encontrada na fila.\n", mission.RequisitionID)
		return fmt.Errorf("requisição não encontrada")
	}

	// Já está em progresso?
	if _, exists := fsm.InProgressReqs[mission.RequisitionID]; exists {
		fmt.Printf("Requisição %s já está em progresso.\n", mission.RequisitionID)
		return nil
	}

	// Verifica se o drone existe antes de alterar o estado da fila/inq.
	drone, ok := fsm.DroneMap[mission.AssignedDrone]
	if !ok {
		fmt.Printf("Abortando: Drone %s não mapeado na FSM.\n", mission.AssignedDrone)
		return fmt.Errorf("drone não encontrado")
	}

	// Agora que tudo foi validado, atualiza o estado
	fsm.InProgressReqs[mission.RequisitionID] = targetReq
	// remove from heap at index
	fsm.PendingReqsQueue.RemoveAt(targetReqIndex)

	drone.SetBusy(mission.RequisitionID)
	fsm.DroneMap[mission.AssignedDrone] = drone

	//TODO: DEIXAR DRONE CUIDAR DO ASSIGNED MISSION?

	if drone.CurrentSector == fsm.Sector {
		topic := fmt.Sprintf("drones/%s/mission", mission.AssignedDrone)
		fmt.Printf("Publicando missão para tópico: %s | payload: %s\n", topic, string(payload))
		token := fsm.Client.Publish(topic, 1, false, []byte(payload))
		token.Wait()
		if token.Error() != nil {
			fmt.Printf("Erro ao publicar missão: %v\n", token.Error())
		}
	}

	fmt.Printf("Sucesso: Drone %s alocado para Incidente %s.\n", mission.AssignedDrone, mission.RequisitionID)
	return nil
}

func (fsm *RaftFSM) handleDEADDrone(payload json.RawMessage) error {
	var droneID string
	if err := json.Unmarshal(payload, &droneID); err != nil {
		return err
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	drone, ok := fsm.DroneMap[droneID]
	if !ok {
		return nil // O drone já foi removido ou nunca existiu
	}

	// O RESGATE: Se o drone morreu com uma missão na mão, devolvemos a missão à fila
	reqID := drone.CurrentMission
	if reqID != "" {
		if req, exists := fsm.InProgressReqs[reqID]; exists {
			// Reinsere na fila com prioridade elevada para ser despachada primeiro
			req.Priority += 1000
			heap.Push(&fsm.PendingReqsQueue, req)
			delete(fsm.InProgressReqs, reqID)
			fmt.Printf("FSM: MISSÃO RESGATADA: Incidente %s voltou para a fila (Drone %s caiu)\n", reqID, droneID)
		}
	}

	// A LIMPEZA: Remove o registro do drone
	delete(fsm.DroneMap, droneID)
	fmt.Printf("FSM: DRONE REMOVIDO: %s foi declarado morto por falta de pulso.\n", droneID)

	return nil
}

func (fsm *RaftFSM) handleHEARTBEAT(payload json.RawMessage) error {
	var heartbeat shared.DroneHeartbeat
	if err := json.Unmarshal(payload, &heartbeat); err != nil {
		return err
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	if drone, ok := fsm.DroneMap[heartbeat.ID]; ok {
		drone.BatteryLevel = heartbeat.BatteryLevel
		drone.LastSeen = time.Now().Unix()
		fsm.DroneMap[heartbeat.ID] = drone
		fmt.Printf("FSM: Heartbeat recebido do Drone %s | Bateria: %d%%\n", heartbeat.ID, heartbeat.BatteryLevel)
	} else {
		fmt.Printf("FSM: Heartbeat recebido de drone desconhecido: %s\n", heartbeat.ID)
	}

	return nil
}

func (fsm *RaftFSM) handleAGING() error {
	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	// Apply aging: requisições esperando > 10s ganham +1 prioridade
	fsm.PendingReqsQueue.ApplyAging(time.Now().Unix(), 10, 1)

	return nil
}

/*
	DEPRECATED

func (fsm *RaftFSM) handleDEASSIGNDrone(payload json.RawMessage) error {

		var droneID string

		if err := json.Unmarshal(payload, &droneID); err != nil {
			fmt.Printf("FSM: Erro ao desserializar pacote: %v.\n", err)
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
			drone.UpdateSector(values.BrokerID)
			fsm.DroneMap[values.DroneID] = drone
		} else {
			fmt.Printf("Drone %s não encontrado.\n", values.DroneID)
		}

		return nil
	}
*/
func (fsm *RaftFSM) GetSector() string {
	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()
	return fsm.Sector
}

func (fsm *RaftFSM) Snapshot() (raft.FSMSnapshot, error) {
	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	clonedDrones := make(map[string]shared.Drone)
	maps.Copy(clonedDrones, fsm.DroneMap)

	clonedIncidents := fsm.PendingReqsQueue.ToSlice()

	clonedInProgress := make(map[string]shared.Requisition)
	maps.Copy(clonedInProgress, fsm.InProgressReqs)

	snapshot := &RaftSnapshot{
		DroneMap:     clonedDrones,
		IncidentList: clonedIncidents,
		InProgress:   clonedInProgress,
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
	fsm.PendingReqsQueue.FromSlice(restored.IncidentList)
	fsm.InProgressReqs = restored.InProgress
	fsm.Mu.Unlock()

	fmt.Printf("FSM Restaurada: %d drones e %d incidentes carregados.\n",
		len(fsm.DroneMap), len(fsm.PendingReqsQueue))

	return nil
}

func setupRaft(dir, id, raftAddr string, fsm *RaftFSM, bootstrap bool) (*raft.Raft, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)

	filtered := &shared.FilteredWriter{
		Output: os.Stderr,
		Filters: []string{
			"dial tcp",
			"failed to appendEntries to",
			"failed to heartbeat to",
		},
	}

	log.Printf("Raft address: %s\n", raftAddr)

	config.Logger = hclog.New(&hclog.LoggerOptions{
		Name:   "raft",
		Level:  hclog.Info,
		Output: filtered,
	})

	// bind do socket
	tcpAddr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, err
	}

	transport, err := raft.NewTCPTransport(raftAddr, tcpAddr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Diagnostic: show which files/paths Raft will use for persistence
	logPath := filepath.Join(dir, "log.db")
	stablePath := filepath.Join(dir, "stable.db")
	fmt.Printf("Raft data dir: %s\n", dir)
	if fi, err := os.Stat(logPath); err == nil {
		fmt.Printf("Found log.db - size=%d bytes\n", fi.Size())
	} else {
		fmt.Printf("log.db not found (will be created): %v\n", err)
	}
	if fi, err := os.Stat(stablePath); err == nil {
		fmt.Printf("Found stable.db - size=%d bytes\n", fi.Size())
	} else {
		fmt.Printf("stable.db not found (will be created): %v\n", err)
	}

	logStore, err := raftboltdb.NewBoltStore(logPath)
	if err != nil {
		return nil, fmt.Errorf("erro abrindo log store (%s): %w", logPath, err)
	}

	stableStore, err := raftboltdb.NewBoltStore(stablePath)
	if err != nil {
		return nil, fmt.Errorf("erro abrindo stable store (%s): %w", stablePath, err)
	}

	snapshots, err := raft.NewFileSnapshotStore(dir, 3, os.Stderr)
	if err != nil {
		return nil, err
	}

	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		return nil, err
	}

	if bootstrap {
		cfg := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: raft.ServerAddress(raftAddr),
			}},
		}
		if err := raftNode.BootstrapCluster(cfg).Error(); err != nil && err != raft.ErrCantBootstrap {
			return nil, err
		}
	}

	return raftNode, nil
}
