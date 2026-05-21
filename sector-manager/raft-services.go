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

// RaftFSM é a máquina de estado finito que representa o estado do setor gerenciado por este nó.
// Todas as mudanças de estado (adicionar requisição, alocar drone, etc) devem passar por esta FSM
//
//	para garantir a consistência entre os nós do cluster Raft.
type RaftFSM struct {
	Mu               sync.Mutex                    // Protege o acesso ao estado interno da FSM
	DroneMap         map[string]shared.Drone       // Mapa de drones conhecidos
	PendingReqsQueue ReqHeap                       // Fila de requisições pendentes (priority queue)
	InProgressReqs   map[string]shared.Requisition // Mapa de requisições sendo atendidas
	Sector           string                        // O nome/ID do setor
	EventsChan       chan MissionPublishEvent      // Canal de eventos para comunicar missões que precisam ser publicadas no MQTT.
	ActionLogs       []string                      // Logs de ações realizados pela FSM. Usado para o dashboard HTML.
}

// RaftSnapshot é a estrutura que representa o estado completo da FSM para fins de snapshotting no Raft.
// O snapshot contém as informações dos drones, requisições pendentes e requisições em progresso.
type RaftSnapshot struct {
	DroneMap     map[string]shared.Drone
	IncidentList []shared.Requisition
	InProgress   map[string]shared.Requisition
}

// MissionPublishEvent representa um evento que a FSM gera quando uma missão é atribuida.
// O setor-manager deve escutar este canal e publicar a missão no MQTT para o drone correspondente.
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

// Apply é o método central da FSM onde todas as mudanças de estado são aplicadas. Ele recebe um log do Raft,
// desserializa o comando, atualiza o relógio de Lamport e chama a função auxiliar específica para cada tipo de operação.
//
// Todas as operações que modificam o estado da FSM (adicionar requisição, remover requisição, alocar drone, etc)
// devem ser implementadas como comandos do Raft e processadas aqui para garantir a consistência entre os nós do cluster.
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

// handleADDRequisition processa o comando de adicionar uma nova requisição à fila.
// Não adiciona requisições que já existem (mesmo ID) nem na fila pendente nem em progresso.
//
// Params:
//   - payload: os dados da requisição serializados em JSON (deve conter um shared.Requisition)
//
// Returns:
//   - error: um erro se a desserialização falhar, ou nil se a requisição for adicionada com sucesso ou já existir.
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

	// Coloca na priority queue
	heap.Push(&fsm.PendingReqsQueue, requisition)
	fmt.Printf("\033[1;33m[FSM]:\033[0m Requisição \033[33m%s\033[1;33m adicionada à fila pendente.\033[0m\n", requisition.ID)

	return nil
}

// handleRMVRequisition processa o comando de remoção de uma requisição que foi concluída.
// Ele libera o drone associado e remove a requisição do mapa de requisições em progresso.
//
// Incrementa o relógio de Lamport se a requisição for de fato removida.
//
// Params:
//   - payload: os dados da requisição concluída serializados em JSON (deve conter um shared.DoneInfo)
//
// Returns:
//   - error: um erro se a desserialização falhar ou o drone não existir, ou nil se a
//     requisição for removida ou não existir.
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

// handleREGDrone processa o comando de registro de um novo drone ou atualização de um drone existente.
// Ele atualiza o mapa de drones da FSM e, se o drone já existia e tinha uma missão em andamento,
// gera um evento para re-publicar a missão no MQTT.
//
// Params:
//   - payload: os dados do drone serializados em JSON (deve conter um shared.Drone)
//
// Returns:
//   - error: um erro se a desserialização falhar, ou nil se o drone for registrado/atualizado com sucesso.
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

// handleASSIGNDrone processa o comando de alocação de um drone para uma requisição.
// Ele move a requisição da fila pendente para o mapa de requisições em progresso,
// atualiza o estado do drone para ocupado e gera um evento para publicar a missão no MQTT.
//
// Params:
//   - payload: os dados da missão de alocação serializados em JSON (deve conter um shared.DroneMission)
//
// Returns:
//   - error: um erro se a desserialização falhar, se a requisição não existir, se o drone não existir;
//     ou nil se a alocação for processada com sucesso ou já estiver em progresso.
func (fsm *RaftFSM) handleASSIGNDrone(payload json.RawMessage) error {
	var mission shared.DroneMission
	if err := json.Unmarshal(payload, &mission); err != nil {
		return fmt.Errorf("erro unmarshal: %v", err)
	}

	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	var targetReq shared.Requisition
	targetReqIndex := -1

	// Procura a requisição na fila com base no ID.
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

// handleDEADDrone processa o comando de um drone que foi declarado morto por falta de pulso.
// Ele remove o drone do mapa de drones e, se o drone tinha uma missão em andamento, resgata a missão de volta para a fila pendente.
//
// Params:
//   - payload: os dados do ID do drone morto serializados em JSON (deve conter uma string com o ID do drone)
//
// Returns:
//   - error: um erro se a desserialização falhar, ou nil se o drone for removido e a missão resgatada com sucesso ou não existir.
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

	// Se o drone morreu com uma missão na mão, devolve a missão pra fila
	reqID := drone.CurrentMission
	if reqID != shared.NONE {
		if req, exists := fsm.InProgressReqs[reqID]; exists {
			// Reinsere na fila com prioridade maxima para ser despachada logo
			req.Priority += 1000
			heap.Push(&fsm.PendingReqsQueue, req)
			delete(fsm.InProgressReqs, reqID)
			fmt.Printf("\033[1;33m[FSM]:\033[0m MISSÃO RESGATADA: Incidente \033[33m%s\033[1;33m voltou para a fila (Drone \033[33m%s\033[1;33m caiu)\033[0m\n", reqID, droneID)

			// NOVO: Log de resgate de missao
			fsm.logAction(fmt.Sprintf("ALERTA: Drone %s inoperante. Missao %s resgatada para a fila!", droneID, reqID))
		}
	}

	// Remove o registro do drone
	delete(fsm.DroneMap, droneID)
	fmt.Printf("\033[1;33m[FSM]:\033[0m DRONE REMOVIDO: \033[33m%s\033[1;33m foi declarado morto por falta de pulso.\033[0m\n", droneID)

	return nil
}

// handleHEARTBEAT processa o comando de pulso de um drone. Ele atualiza o nível de bateria e
// o timestamp de última vez que o drone foi visto.
//
// Params:
//   - payload: os dados do pulso serializados em JSON (deve conter um shared.DroneHeartbeat)
//
// Returns:
//   - error: um erro se a desserialização falhar, ou nil se o drone for atualizado com sucesso ou não existir.
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

// handleAGING processa o comando de envelhecimento das requisições na fila.
//
// Adiciona +1 de prioridade para requisições que estão esperando há mais de 20 segundos.
//
// Returns:
//   - error: sempre nil (apenas para manter a assinatura consistente com os outros handlers de comando).
func (fsm *RaftFSM) handleAGING() error {
	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()

	// Apply aging: requisições esperando > 20s ganham +1 prioridade
	fsm.PendingReqsQueue.ApplyAging(time.Now().Unix(), 20, 1)
	return nil
}

// GetSector retorna o nome/ID do setor gerenciado por esta FSM.
func (fsm *RaftFSM) GetSector() string {
	fsm.Mu.Lock()
	defer fsm.Mu.Unlock()
	return fsm.Sector
}

// Snapshot deve ser implementado para gerar um snapshot do estado atual da FSM para ser salvo pelo Raft.
// Ele clona o estado interno (drones, requisições pendentes e em progresso).
//
// Returns:
//   - raft.FSMSnapshot: o snapshot do estado atual da FSM, que será salvo pelo Raft.
//   - error: um erro se algo der errado durante a criação do snapshot, ou nil se o snapshot for criado com sucesso.
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

// Persist é o método do snapshot que o Raft chama para salvar o snapshot em um destino (sink).
// Ele serializa o snapshot em JSON e escreve no sink. O Raft cuida de salvar o snapshot no disco
// e gerenciar os arquivos.
//
// Essa implementação é simples e genérica.
//
// Returns:
//   - error: um erro se a serialização falhar ou se houver um problema ao fechar o sink, ou nil se o snapshot for persistido com sucesso.
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

// Release é o método do snapshot que o Raft chama quando o snapshot pode ser liberado da memória.
// Não é necessário fazer nada aqui, pois não abro arquivos externos.
//
// Ainda assim, deve ser implementado para satisfazer a interface raft.FSMSnapshot, mesmo que seja vazio.

func (snapshot *RaftSnapshot) Release() {
}

// Restore é o método implementado da FSM que o Raft chama para restaurar o estado da FSM a partir de um snapshot.
// Ele lê os dados do snapshot, desserializa o JSON e atualiza o estado interno
// da FSM (drones, requisições pendentes e em progresso).
//
// Returns:
//   - error: um erro se a desserialização falhar ou se houver um problema ao ler o snapshot, ou nil se a FSM for restaurada com sucesso.
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

// setupRaft é a função responsável por configurar e iniciar o nó Raft para este setor.
// Ela configura o armazenamento, transporte e outras opções do Raft, e inicia o nó Raft com a FSM fornecida.
//
// Se o nó não tiver estado existente (logs ou snapshot) e o parâmetro bootstrap for true, ela faz o bootstrap do cluster.
// Se o nó já tiver estado existente, ela apenas inicia o Raft normalmente para se juntar ao cluster existente.
//
// A saída padrão é filtrada para evitar poluição do terminal. Apenas logs relevantes, como mudanças de liderança, são exibidos.
// Params:
//   - dir: o diretório onde os arquivos de log e snapshot do Raft serão armazenados.
//   - id: o ID único deste nó/setor no cluster Raft.
//   - raftAddr: o endereço de rede onde este nó irá escutar as comunicações do Raft.
//   - fsm: a instância da FSM que representa o estado deste setor, que será usada pelo Raft para aplicar os logs.
//   - bootstrap: um booleano indicando se este nó deve tentar fazer o bootstrap do cluster (deve ser true para o primeiro nó do cluster).
//
// Returns:
//   - *raft.Raft: a instância do nó Raft iniciado, ou nil se houve um erro.
//   - bool: true se o nó já tinha estado existente (log/snapshot) e false se é um nó novo sem estado.
//   - error: um erro se algo der errado durante a configuração ou inicialização do Raft, ou nil se tudo for bem-sucedido.
func setupRaft(dir, id, raftAddr string, fsm *RaftFSM, bootstrap bool) (*raft.Raft, bool, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, false, err
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)

	// As mensagens abaixo são filtradas e não aparecem no terminal.
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

	// Configura o logger do Raft para usar o filtered writer. É possível ajustar o nível de log para remover mensagens de nível inferior.
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

	// Configura o transporte TCP interno para o Raft. Ele usará o endereço especificado e o filtered writer para logs.
	transport, err := raft.NewTCPTransport(raftAddr, tcpAddr, 3, 10*time.Second, raftWriter)
	if err != nil {
		return nil, false, err
	}

	// Configura os caminhos para os arquivos de log e estado estável do Raft. O Raft usará o BoltDB
	// (também da Hashicorp) para armazenar esses dados.
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

	// Cria o nó com todas as configurações.
	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		return nil, false, err
	}

	// Verifica se já existe estado.
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
