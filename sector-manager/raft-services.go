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
	ActionLogs       []string // NOVO: Guarda os logs para o dashboard
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

// logAction adiciona uma mensagem formatada ao terminal do dashboard
// Assuma que o Mutex (fsm.Mu) ja esta bloqueado pelas funcoes que a chamam.
func (fsm *RaftFSM) logAction(msg string) {
	formattedMsg := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	fsm.ActionLogs = append([]string{formattedMsg}, fsm.ActionLogs...)
	if len(fsm.ActionLogs) > 10 {
		fsm.ActionLogs = fsm.ActionLogs[:10]
	}
}

func (fsm *RaftFSM) Apply(log *raft.Log) interface{} {
	var cmd shared.HeaderCommand

	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		fmt.Printf("\033[1;33m[FSM]:\033[0m Erro ao desserializar comando: %v.\n", err)
		fmt.Printf("\033[1;33m[FSM]:\033[0m Log data: %s\n", string(log.Data))
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
		fmt.Printf("\033[1;33m[FSM]:\033[0m Operação desconhecida: %s\n", cmd.Operation)
		return nil
	}
}

func (fsm *RaftFSM) handleADDRequisition(payload json.RawMessage) error {
	var requisition shared.Requisition

	if err := json.Unmarshal(payload, &requisition); err != nil {
		fmt.Printf("\033[1;33m[FSM]:\033[0m Erro ao desserializar pacote: %v.\n", err)
		return err
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	// Evita duplicatas: checar tanto em Pending quanto em InProgress
	for _, v := range fsm.PendingReqsQueue {
		if v.ID == requisition.ID {
			fmt.Printf("\033[1;33m[FSM]:\033[0m Requisição \033[33m%s\033[1;33m já existe na fila pendente.\033[0m\n", v.ID)
			return nil
		}
	}

	if _, inProgress := fsm.InProgressReqs[requisition.ID]; inProgress {
		fmt.Printf("\033[1;33m[FSM]:\033[0m Requisição \033[33m%s\033[1;33m já está em progresso.\033[0m\n", requisition.ID)
		return nil
	}

	// push into priority queue
	heap.Push(&fsm.PendingReqsQueue, requisition)
	fmt.Printf("\033[1;33m[FSM]:\033[0m Requisição \033[33m%s\033[1;33m adicionada à fila pendente.\033[0m\n", requisition.ID)

	return nil
}

func (fsm *RaftFSM) handleRMVRequisition(payload json.RawMessage) error {
	var doneInfo shared.DoneInfo

	if err := json.Unmarshal(payload, &doneInfo); err != nil {
		fmt.Printf("\033[1;33m[FSM]:\033[0m Erro ao desserializar pacote: %v.\n", err)
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
		// Sem missão atribuída
		fmt.Printf("\033[1;33m[FSM]:\033[0m Drone \033[33m%s\033[1;33m não possui missão atual.\033[0m\n", doneInfo.DroneID)
		return nil
	}

	if _, exist := fsm.InProgressReqs[reqID]; exist {
		LClock.Tick()

		// TODO: DEBUG_MODE_LAMPORT_TICK
		if DebugMode {
			fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Removendo Requisição e Liberando Drone na FSM\n", LClock.GetTime())
		}

		drone.SetIdle()
		fsm.DroneMap[doneInfo.DroneID] = drone
		delete(fsm.InProgressReqs, doneInfo.RequisitionID)

		fmt.Printf("\033[1;33m[FSM]:\033[0m Requisição \033[33m%s\033[1;33m concluída pelo drone \033[33m%s\033[1;33m.\033[0m\n", doneInfo.RequisitionID, doneInfo.DroneID)

		fsm.logAction(fmt.Sprintf("INFO: Drone %s concluiu a missao %s com sucesso", doneInfo.DroneID, doneInfo.RequisitionID))
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

	fmt.Printf("\033[1;33m[FSM]:\033[0m Novo drone registrado: \033[33m%s\033[0m\n", newDrone.ID)
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
			fmt.Printf("\033[1;33m[FSM]:\033[0m Evento gerado para restaurar missão \033[33m%s\033[1;33m do drone \033[33m%s\033[1;33m.\033[0m\n", missionToRestore.RequisitionID, newDrone.ID)
		default:
			fmt.Printf("\033[1;33m[FSM]:\033[0m Aviso: Canal de eventos cheio. Falha ao gerar evento de restauração para \033[33m%s\033[1;33m\033[0m\n", newDrone.ID)
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
		fmt.Printf("\033[1;33m[FSM]:\033[0m Abortando: Requisição \033[33m%s\033[1;33m não encontrada na fila.\033[0m\n", mission.RequisitionID)
		return fmt.Errorf("requisição não encontrada")
	}

	// Já está em progresso?
	if _, exists := fsm.InProgressReqs[mission.RequisitionID]; exists {
		fmt.Printf("\033[1;33m[FSM]:\033[0m Requisição \033[33m%s\033[1;33m já está em progresso.\033[0m\n", mission.RequisitionID)
		return nil
	}

	// Verifica se o drone existe antes de alterar o estado da fila/inq.
	drone, ok := fsm.DroneMap[mission.AssignedDrone]
	if !ok {
		fmt.Printf("\033[1;33m[FSM]:\033[0m Abortando: Drone \033[33m%s\033[1;33m não mapeado na FSM.\033[0m\n", mission.AssignedDrone)
		return fmt.Errorf("drone não encontrado")
	}

	// Agora que tudo foi validado, atualiza o estado
	fsm.InProgressReqs[mission.RequisitionID] = targetReq
	fsm.PendingReqsQueue.RemoveAt(targetReqIndex)

	drone.SetBusy(mission.RequisitionID)
	fsm.DroneMap[mission.AssignedDrone] = drone

	// NOVO: Log de alocacao (chamado enquanto o Mutex ainda esta trancado)
	fsm.logAction(fmt.Sprintf("START: Drone %s foi alocado para o incidente %s", mission.AssignedDrone, mission.RequisitionID))

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
			fmt.Printf("\033[1;33m[FSM]:\033[0m Sucesso: Evento de alocação (Incidente \033[33m%s\033[1;33m -> Drone \033[33m%s\033[1;33m) enviado ao canal.\033[0m\n", mission.RequisitionID, mission.AssignedDrone)
		default:
			fmt.Printf("\033[1;33m[FSM]:\033[0m Aviso: Canal de eventos cheio. Falha ao gerar evento MQTT para drone \033[33m%s\033[1;33m\033[0m\n", mission.AssignedDrone)
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

	// Se o drone morreu com uma missão na mão, devolvo a missão pra fila
	reqID := drone.CurrentMission
	if reqID != shared.NONE {
		if req, exists := fsm.InProgressReqs[reqID]; exists {
			// Reinsere na fila com prioridade maxima para ser despachada primeiro
			req.Priority += 1000
			heap.Push(&fsm.PendingReqsQueue, req)
			delete(fsm.InProgressReqs, reqID)
			fmt.Printf("\033[1;33m[FSM]:\033[0m MISSÃO RESGATADA: Incidente \033[33m%s\033[1;33m voltou para a fila (Drone \033[33m%s\033[1;33m caiu)\033[0m\n", reqID, droneID)

			// NOVO: Log de resgate de missao
			fsm.logAction(fmt.Sprintf("ALERTA: Drone %s inoperante. Missao %s resgatada para a fila!", droneID, reqID))
		}
	}

	// A LIMPEZA: Remove o registro do drone
	delete(fsm.DroneMap, droneID)
	fmt.Printf("\033[1;33m[FSM]:\033[0m DRONE REMOVIDO: \033[33m%s\033[1;33m foi declarado morto por falta de pulso.\033[0m\n", droneID)

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
	fmt.Println("\033[1;33m[FSM]:\033[0m GUARDANDO ESTADO")
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
	fmt.Println("\033[1;33m[FSM]:\033[0m RESTAURANDO...")
	var restored RaftSnapshot
	if err := json.NewDecoder(rc).Decode(&restored); err != nil {
		return fmt.Errorf("Erro ao decodificar snapshot: %v", err)
	}

	fsm.Mu.Lock()
	fsm.DroneMap = restored.DroneMap
	fsm.PendingReqsQueue.FromSlice(restored.IncidentList)
	fsm.InProgressReqs = restored.InProgress
	fsm.Mu.Unlock()

	fmt.Printf("\033[1;33m[FSM]:\033[0m Restaurada: %d drones e %d incidentes carregados.\n",
		len(fsm.DroneMap), len(fsm.PendingReqsQueue))

	return nil
}

func setupRaft(dir, id, raftAddr string, fsm *RaftFSM, bootstrap bool) (*raft.Raft, bool, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, false, err
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)
	raftWriter := &shared.FilteredWriter{
		Output: &RaftStatesWriter{},
		Filters: []string{
			"raft: initial configuration",
			"raft: updating configuration:",
			"raft: election won:",
			"raft: heartbeat timeout reached, starting election",
			"raft: failed to make requestVote RPC",
			"raft: pre-vote campaign failed, waiting for election timeout",
			"no known peers",
			"raft: appendEntries rejected",
			"added peer",
			"raft: Election timeout reached, restarting election",
			"raft: rejecting pre-vote request",
			"raft: failed to contact:",
			"raft: pre-vote successful",
			"raft: failed to get previous log",
			"dial tcp",
			"failed to appendEntries to",
			"failed to heartbeat to",
			"Rollback failed: tx closed",
			"raft: pipelining replication:",
			"raft: aborting pipeline replication:",
		},
	}

	log.Printf("\033[1;94m[LOCAL]:\033[0m Endereço Raft: %s\n", raftAddr)

	config.Logger = hclog.New(&hclog.LoggerOptions{
		Name:   "raft",
		Level:  hclog.Info,
		Output: raftWriter,
	})

	// bind do socket
	tcpAddr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, false, err
	}

	transport, err := raft.NewTCPTransport(raftAddr, tcpAddr, 3, 10*time.Second, raftWriter)
	if err != nil {
		return nil, false, err
	}

	// Diagnostic: show which files/paths Raft will use for persistence
	logPath := filepath.Join(dir, "log.db")
	stablePath := filepath.Join(dir, "stable.db")
	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Raft data dir: %s\n", dir)
	if fi, err := os.Stat(logPath); err == nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Encontrado log.db - Tamanho = %d bytes\n", fi.Size())
	} else {
		fmt.Println("\033[1;94m[LOCAL]:\033[0m log.db não encontrado (será criado)")
	}
	if fi, err := os.Stat(stablePath); err == nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Encontrado stable.db - Tamanho = %d bytes\n", fi.Size())
	} else {
		fmt.Println("\033[1;94m[LOCAL]:\033[0m stable.db não encontrado (será criado)")
	}

	logStore, err := raftboltdb.NewBoltStore(logPath)
	if err != nil {
		return nil, false, fmt.Errorf("Erro abrindo log store (%s): %w", logPath, err)
	}

	stableStore, err := raftboltdb.NewBoltStore(stablePath)
	if err != nil {
		return nil, false, fmt.Errorf("Erro abrindo stable store (%s): %w", stablePath, err)
	}

	snapshots, err := raft.NewFileSnapshotStore(dir, 3, raftWriter)
	if err != nil {
		return nil, false, err
	}

	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		return nil, false, err
	}

	alreadyInDB, err := raft.HasExistingState(logStore, stableStore, snapshots)
	if err != nil {
		return nil, false, fmt.Errorf("erro checando estado: %v", err)
	}

	// Só faz bootstrap se for solicitado e se o nó ja não estiver no cluster (não está na database)
	if bootstrap && !alreadyInDB {
		cfg := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: raft.ServerAddress(raftAddr),
			}},
		}
		if err := raftNode.BootstrapCluster(cfg).Error(); err != nil && err != raft.ErrCantBootstrap {
			return nil, false, err
		}
	}

	return raftNode, alreadyInDB, nil
}
