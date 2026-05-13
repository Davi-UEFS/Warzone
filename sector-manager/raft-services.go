package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net"
	"os"
	"path/filepath"
	"slices"
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
	PendingReqsQueue []shared.Requisition //TODO: IMPLEMENTAR PRIORITY QUEUE DEPOIS
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

	case OP_DEASSIGN:
		return fsm.handleDEASSIGNDrone(cmd.Payload)

		/* DEPRECATED
		case OP_UPDATEDRB:
			return fsm.handleUPDATEDRBroker(cmd.Payload)
		*/
	case OP_REGDRONE:
		return fsm.handleREGDrone(cmd.Payload)
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

	fsm.PendingReqsQueue = append(fsm.PendingReqsQueue, requisition) //TODO: IMPLEMENTAR PRIORITY QUEUE DEPOIS
	fsm.Mu.Unlock()

	return nil
}

func (fsm *RaftFSM) handleRMVRequisition(payload json.RawMessage) error {

	var droneID string

	if err := json.Unmarshal(payload, &droneID); err != nil {
		fmt.Printf("Erro ao desserializar ID do drone: %v.\n", err)
		return err
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	drone, ok := fsm.DroneMap[droneID]

	if !ok {
		return fmt.Errorf("Drone %s não encontrado\n", droneID)
	}

	reqID := drone.CurrentMission

	if reqID == "" {
		// Sem missão atribuída; nada a remover
		fmt.Printf("Drone %s não possui missão atual.\n", droneID)
		return nil
	}

	requisition, exist := fsm.InProgressReqs[reqID]

	if exist {

		LClock.Tick()

		if requisition.OriginSector == fsm.Sector {
			topic := fmt.Sprintf("sensors/%s/solved", shared.ExtractSensorID(requisition.ID))
			response := shared.SolvedInfo{
				RequisitionID: requisition.ID,
				LCTime:        LClock.GetTime(),
			}

			payload, _ := json.Marshal(response)

			token := fsm.Client.Publish(topic, 1, false, payload)
			token.Wait()

			if token.Error() != nil {
				fmt.Printf("Erro ao publicar resolução de incidente: %v\n", token.Error())
			}
		}

		// Primeiro liberta o drone localmente, depois limpa o registro da requisição
		drone.SetIdle()
		fsm.DroneMap[droneID] = drone
		delete(fsm.InProgressReqs, reqID)

		fmt.Printf("Requisição %s concluída pelo drone %s.\n", reqID, droneID)

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

	fsm.DroneMap[newDrone.ID] = newDrone

	fmt.Printf("FSM: Novo drone registrado: %s (Broker: %s)\n", newDrone.ID, newDrone.CurrentBroker)

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
	fsm.PendingReqsQueue = append(fsm.PendingReqsQueue[:targetReqIndex], fsm.PendingReqsQueue[targetReqIndex+1:]...)

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

/*
	DEPRECATED

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

	clonedIncidents := slices.Clone(fsm.PendingReqsQueue)

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
	fsm.PendingReqsQueue = restored.IncidentList
	fsm.InProgressReqs = restored.InProgress
	fsm.Mu.Unlock()

	fmt.Printf("FSM Restaurada: %d drones e %d incidentes carregados.\n",
		len(fsm.DroneMap), len(fsm.PendingReqsQueue))

	return nil
}

func setupRaft(dir, id, bindAddr, advertiseAddr string, fsm *RaftFSM, bootstrap bool) (*raft.Raft, error) {
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

	log.Printf("Raft bind=%s advertise=%s\n", bindAddr, advertiseAddr)

	config.Logger = hclog.New(&hclog.LoggerOptions{
		Name:   "raft",
		Level:  hclog.Info,
		Output: filtered,
	})

	// Endereço anunciado aos demais nós precisa ser roteável (não 0.0.0.0).
	advertiseTCPAddr, err := net.ResolveTCPAddr("tcp", advertiseAddr)
	if err != nil {
		return nil, err
	}

	transport, err := raft.NewTCPTransport(bindAddr, advertiseTCPAddr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dir, "log.db"))
	if err != nil {
		return nil, err
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dir, "stable.db"))
	if err != nil {
		return nil, err
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
				Address: raft.ServerAddress(advertiseAddr),
			}},
		}
		if err := raftNode.BootstrapCluster(cfg).Error(); err != nil && err != raft.ErrCantBootstrap {
			return nil, err
		}
	}

	return raftNode, nil
}
