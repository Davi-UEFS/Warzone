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
	"github.com/hashicorp/go-hclog"
	raft "github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type RaftFSM struct {
	Mu               sync.Mutex
	DroneMap         map[string]shared.Drone
	PendingReqsQueue ReqHeap
	InProgressReqs   map[string]shared.Requisition
	Sector           string
	EventsChan       chan MissionPublishEvent
}

type RaftSnapshot struct {
	DroneMap     map[string]shared.Drone
	IncidentList []shared.Requisition
	InProgress   map[string]shared.Requisition
}

type MissionPublishEvent struct {
	Topic   string
	Qos     byte
	Payload []byte
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
		return nil
	}
}

func (fsm *RaftFSM) handleADDRequisition(payload json.RawMessage) error {
	var requisition shared.Requisition

	if err := json.Unmarshal(payload, &requisition); err != nil {
		fmt.Printf("Erro ao desserializar pacote: %v.\n", err)
		return err
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	// Evita duplicatas: checar tanto em Pending quanto em InProgress
	for _, v := range fsm.PendingReqsQueue {
		if v.ID == requisition.ID {
			fmt.Printf("Requisição %s já existe na fila pendente.\n", v.ID)
			return nil
		}
	}

	if _, inProgress := fsm.InProgressReqs[requisition.ID]; inProgress {
		fmt.Printf("Requisição %s já está em progresso.\n", requisition.ID)
		return nil
	}

	// push into priority queue
	heap.Push(&fsm.PendingReqsQueue, requisition)
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

	if reqID == shared.NONE {
		// Sem missão atribuída; nada a remover
		fmt.Printf("Drone %s não possui missão atual.\n", doneInfo.DroneID)
		return nil
	}

	if _, exist := fsm.InProgressReqs[reqID]; exist {
		LClock.Tick()
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
	prevDrone, exists := fsm.DroneMap[newDrone.ID]
	newDrone.LastSeen = time.Now().Unix()

	var missionToRestore shared.DroneMission
	var shouldRestoreMission bool

	// Detecta se precisa restaurar a missão (Fast-Reboot ou Reconexão)
	if exists && prevDrone.CurrentMission != shared.NONE {
		newDrone.Status = prevDrone.Status
		newDrone.CurrentMission = prevDrone.CurrentMission

		if req, ok := fsm.InProgressReqs[prevDrone.CurrentMission]; ok {
			missionToRestore = shared.DroneMission{
				RequisitionID: req.ID,
				AssignedDrone: newDrone.ID,
				Type:          req.Type,
				Coordinate:    req.Coord,
				LamportTime:   req.LamportTime,
			}
			shouldRestoreMission = true
		}
	}

	fmt.Printf("FSM: Novo drone registrado: %s (Broker: %s)\n", newDrone.ID, newDrone.CurrentBroker)
	fsm.DroneMap[newDrone.ID] = newDrone
	fsm.Mu.Unlock()

	// Envia o evento de re-publicação da missão apenas se o nó atual for o dono do setor do drone
	if shouldRestoreMission && newDrone.CurrentSector == fsm.Sector {
		missionPayload, err := json.Marshal(missionToRestore)
		if err != nil {
			return fmt.Errorf("erro ao serializar missão de recuperação: %v", err)
		}

		event := MissionPublishEvent{
			Topic:   fmt.Sprintf("drones/%s/mission", newDrone.ID),
			Qos:     1,
			Payload: missionPayload,
		}

		select {
		case fsm.EventsChan <- event:
			fmt.Printf("FSM: Evento gerado para restaurar missão %s do drone %s.\n", missionToRestore.RequisitionID, newDrone.ID)
		default:
			fmt.Printf("Aviso: Canal de eventos cheio. Falha ao gerar evento de restauração para %s\n", newDrone.ID)
		}
	}

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
	fsm.PendingReqsQueue.RemoveAt(targetReqIndex)

	drone.SetBusy(mission.RequisitionID)
	fsm.DroneMap[mission.AssignedDrone] = drone

	log.Printf("O setor do drone atual é: %s\n", drone.CurrentSector)

	// A FSM NÃO publica mais no MQTT diretamente. Em vez disso, gera um evento.
	// Só despachamos o evento se este nó do Raft representar o setor atual do drone.
	if drone.CurrentSector == fsm.Sector {
		event := MissionPublishEvent{
			Topic:   fmt.Sprintf("drones/%s/mission", mission.AssignedDrone),
			Qos:     1,
			Payload: payload,
		}

		select {
		case fsm.EventsChan <- event:
			fmt.Printf("Sucesso: Evento de alocação (Incidente %s -> Drone %s) enviado ao canal.\n", mission.RequisitionID, mission.AssignedDrone)
		default:
			fmt.Printf("Aviso: Canal de eventos cheio. Falha ao gerar evento MQTT para drone %s\n", mission.AssignedDrone)
		}
	}

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
	if reqID != shared.NONE { // CORREÇÃO: Tem que ser != (diferente de NONE) para indicar que existe missão
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
	}

	return nil
}

func (fsm *RaftFSM) handleAGING() error {
	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	// Apply aging: requisições esperando > 20s ganham +1 prioridade
	fsm.PendingReqsQueue.ApplyAging(time.Now().Unix(), 20, 1)

	return nil
}

func (fsm *RaftFSM) GetSector() string {
	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()
	return fsm.Sector
}

func (fsm *RaftFSM) Snapshot() (raft.FSMSnapshot, error) {
	log.Println("GUARDANDO ESTADO")
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
		return fmt.Errorf("Erro ao codificar snapshot: %v", err)
	}

	if err := sink.Close(); err != nil {
		return fmt.Errorf("Erro ao fechar snapshot sink: %v", err)
	}

	return nil
}

func (snaphot *RaftSnapshot) Release() {
}

func (fsm *RaftFSM) Restore(rc io.ReadCloser) error {
	log.Println("RESTAURANDO...")
	var restored RaftSnapshot
	if err := json.NewDecoder(rc).Decode(&restored); err != nil {
		return fmt.Errorf("Erro ao decodificar snapshot: %v", err)
	}

	fsm.Mu.Lock()
	fsm.DroneMap = restored.DroneMap
	fsm.PendingReqsQueue.FromSlice(restored.IncidentList)
	fsm.InProgressReqs = restored.InProgress
	fsm.Mu.Unlock()

	log.Printf("FSM Restaurada: %d drones e %d incidentes carregados.\n",
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

	log.Printf("Endereço Raft: %s\n", raftAddr)

	config.Logger = hclog.New(&hclog.LoggerOptions{
		Name:   "raft",
		Level:  hclog.Error, // Loga apenas erros para evitar poluição
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
		fmt.Printf("Encontrado log.db - Tamanho = %d bytes\n", fi.Size())
	} else {
		fmt.Println("log.db não encontrado (será criado)")
	}
	if fi, err := os.Stat(stablePath); err == nil {
		fmt.Printf("Encontrado stable.db - Tamanho = %d bytes\n", fi.Size())
	} else {
		fmt.Println("stable.db não encontrado (será criado)")
	}

	logStore, err := raftboltdb.NewBoltStore(logPath)
	if err != nil {
		return nil, fmt.Errorf("Erro abrindo log store (%s): %w", logPath, err)
	}

	stableStore, err := raftboltdb.NewBoltStore(stablePath)
	if err != nil {
		return nil, fmt.Errorf("Erro abrindo stable store (%s): %w", stablePath, err)
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
