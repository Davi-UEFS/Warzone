package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	raft "github.com/hashicorp/raft"
)

// startSignaling inicia o servidor TCP para receber mensagens de outros setores.
//
// Params:
//   - raftNode: O nó Raft local, necessário para processar as mensagens recebidas e aplicar as mudanças no consenso.
//   - port: A porta TCP onde o servidor irá escutar por conexões.
func startSignaling(raftNode *raft.Raft, port string) {
	ln, _ := net.Listen("tcp", port)
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, raftNode)
	}
}

// handleConnection processa uma conexão TCP recebida, decodificando o comando e executando a ação correspondente.
//
// O comando é esperado no formato JSON e deve conter um campo "Operation" que indica o tipo de operação a ser realizada.
func handleConnection(conn net.Conn, raftNode *raft.Raft) {
	defer conn.Close()

	var cmd shared.HeaderCommand

	if err := json.NewDecoder(conn).Decode(&cmd); err != nil {
		return
	}

	switch cmd.Operation {
	case QUERY:
		// Qualquer nó pode responder a essa consulta, pois é apenas para obter o líder atual.
		leader := string(raftNode.Leader())
		leaderInfo := LeaderInfo{
			RaftAddr: leader,
			SigAddr:  getSigAddr(leader),
		}

		json.NewEncoder(conn).Encode(leaderInfo)

	case JOIN:
		if raftNode.State() != raft.Leader {
			json.NewEncoder(conn).Encode(ERR_NOT_LEADER)
			return
		}

		err := handleJoinRequest(raftNode, cmd.Payload)

		if err == nil {
			json.NewEncoder(conn).Encode(SUCCESS)
		}

	case FORWARD_ALR:
		if raftNode.State() != raft.Leader {
			json.NewEncoder(conn).Encode(ERR_NOT_LEADER)
			return
		}

		err := handleForwardingAlert(raftNode, cmd.Payload)

		if err == nil {
			json.NewEncoder(conn).Encode(SUCCESS)
		}

	case FORWARD_DONE:
		if raftNode.State() != raft.Leader {
			json.NewEncoder(conn).Encode(ERR_NOT_LEADER)
			return
		}

		err := handleForwardingDone(raftNode, cmd.Payload)

		if err == nil {
			json.NewEncoder(conn).Encode(SUCCESS)
		}

	case FORWARD_REG:
		if raftNode.State() != raft.Leader {
			json.NewEncoder(conn).Encode(ERR_NOT_LEADER)
			return
		}

		err := handleForwardingRegisterDrone(raftNode, cmd.Payload)

		if err == nil {
			json.NewEncoder(conn).Encode(SUCCESS)
		}

	case FORWARD_HB:
		if raftNode.State() != raft.Leader {
			json.NewEncoder(conn).Encode(ERR_NOT_LEADER)
			return
		}
		handleForwardingHeartbeat(raftNode, cmd.Payload)
		json.NewEncoder(conn).Encode(SUCCESS)

	default:
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Operação desconhecida recebida via sinalização: %s\n", cmd.Operation)
		json.NewEncoder(conn).Encode("Operação desconhecida")
	}

}

// sendJoinRequest envia uma solicitação de join para o líder do cluster, contendo o ID e endereço do novo nó.
// Params:
//   - sigAddr: endereço TCP de escuta do líder para onde a solicitação deve ser enviada.
//   - cmd: comando de QUERY a ser enviado.
//
// Returns:
//   - error: erro ocorrido durante o processo, ou nil se a solicitação foi bem-sucedida.
func sendJoinRequest(sigAddr string, cmd shared.HeaderCommand) error {
	conn, err := net.DialTimeout("tcp", sigAddr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	json.NewEncoder(conn).Encode(cmd)

	var response string
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return err
	}

	if response != SUCCESS {
		return fmt.Errorf("Resposta inesperada: %s", response)
	}

	return nil

}

// forwardCommand é uma função genérica para encaminhar comandos de operações que devem
// ser processados pelo líder do cluster.
//
// Ela é utilizada para operações como FORWARD_ALR, FORWARD_DONE, FORWARD_REG e FORWARD_HB.
//
// Params:
//   - sigAddr: endereço TCP de escuta do líder para onde a solicitação deve ser enviada.
//   - cmd: comando contendo a operação e payload a ser encaminhado.
//
// Returns:
//   - error: erro ocorrido durante o processo, ou nil se a solicitação foi bem-sucedida.
func forwardCommand(sigAddr string, cmd shared.HeaderCommand) error {
	conn, err := net.DialTimeout("tcp", sigAddr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	json.NewEncoder(conn).Encode(cmd)

	var response string
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return err
	}

	if response != SUCCESS {
		return fmt.Errorf("Resposta inesperada: %s", response)
	}

	return nil

}

// handleJoinRequest processa uma solicitação de join recebida pelo líder, adicionando o novo nó ao cluster Raft.
//
// Params:
//   - raftNode: instância do Raft para adicionar o novo nó.
//   - payload: dados da solicitação de join, contendo o ID e endereço do novo nó.
//
// Returns:
//   - error: erro ocorrido durante o processo, ou nil se o nó foi adicionado com sucesso.
func handleJoinRequest(raftNode *raft.Raft, payload json.RawMessage) error {

	var req joinReq

	if err := json.Unmarshal(payload, &req); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao desserializar join request: %v\n", err)
		return err
	}

	future := raftNode.AddVoter(raft.ServerID(req.ID), raft.ServerAddress(req.Addr), 0, 0)
	if err := future.Error(); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Falha ao adicionar nó %s ao consenso: %v\n", req.ID, err)
		return err
	}

	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Nó %s integrado com sucesso!\n", req.ID)

	return nil

}

// handleForwardingAlert processa um alerta encaminhado por outro setor.
//
// Cria uma nova requisição no sistema.
//
// Atualiza o relógio de Lamport.
//
// Params:
//   - raftNode: instância do Raft para aplicar o comando de criação da requisição no consenso.
//   - payload: dados do alerta encaminhado, contendo informações como sensor, tipo de alerta, coordenadas e tempo de Lamport. O payload vem com o ID do setor de origem.
//
// Returns:
//   - error: erro ocorrido durante o processo, ou nil se a requisição foi criada com sucesso.
func handleForwardingAlert(raftNode *raft.Raft, payload json.RawMessage) error {

	var fwd forwardedAlert
	if err := json.Unmarshal(payload, &fwd); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao desserializar alerta: %v\n", err)
		return err
	}

	alert := fwd.Alert
	originSector := fwd.OriginSector
	if originSector == "" {
		originSector = sectorFSM.GetSector()
	}

	LClock.CompareAndUpdate(alert.LamportTime)
	LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Encaminhamento via TCP\n", LClock.GetTime())
	}

	reqID := createIncidentID(alert.SensorID)

	requisition := shared.Requisition{
		ID:           reqID,
		Priority:     PRIOTIRIES[alert.Type], //Prioridade baseada no tipo de alerta
		Type:         alert.Type,
		Coord:        alert.Coordinate,
		OriginSector: originSector,
		LamportTime:  LClock.GetTime(),
		CreatedAt:    time.Now().Unix(),
	}

	newPayload, _ := json.Marshal(requisition)

	cmd := shared.HeaderCommand{
		Operation:   OP_ADDREQ,
		Payload:     newPayload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		fmt.Println("\033[1;94m[LOCAL]:\033[0m Falha ao adicionar requisição ao consenso: ", err)
		return err
	}

	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Requisição criada para o sensor %s: %s\n", alert.SensorID, reqID)

	return nil

}

// handleForwardingDone processa uma notificação de missão concluída encaminhada por outro setor.
//
// Remove a requisição do sistema e libera o drone associado via Raft utilizando o comando RMVREQ.
//
// Atualiza o relógio de Lamport.
//
// Params:
//   - raftNode: instância do Raft para aplicar o comando de remoção da requisição no consenso.
//   - payload: dados da notificação de missão concluída, contendo informações como ID do drone, ID da requisição, tempo de Lamport, etc.
//
// Returns:
//   - error: erro ocorrido durante o processo, ou nil se a requisição foi removida e o drone liberado com sucesso.
func handleForwardingDone(raftNode *raft.Raft, payload json.RawMessage) error {
	var result shared.DoneInfo

	if err := json.Unmarshal(payload, &result); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao unmarshal payload: %v\n", err)
		return err
	}

	LClock.CompareAndUpdate(result.LCTime)
	LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Done de drone recebido por TCP\n", LClock.GetTime())
	}

	// Preciso fazer isso pq RawMessage nao e considerado []byte.
	newPayload, _ := json.Marshal(result)

	cmd := shared.HeaderCommand{
		Operation:   OP_RMVREQ,
		Payload:     newPayload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		return fmt.Errorf("\033[1;94m[LOCAL]:\033[0m Erro ao aplicar comando no Raft: %v\n", err)
	}

	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Drone %s liberado da missão %s\n", result.DroneID, result.RequisitionID)
	return nil

}

// handleForwardingRegisterDrone processa uma notificação de registro de drone encaminhada por outro setor.
//
// Registra o drone no sistema via Raft utilizando o comando REGDRONE.
//
// Atualiza o relógio de Lamport.
//
// Params:
//   - raftNode: instância do Raft para aplicar o comando de registro do drone no consenso.
//   - payload: dados da notificação de registro de drone, contendo informações como ID do drone, coordenadas, etc.
//
// Returns:
//   - error: erro ocorrido durante o processo, ou nil se o drone foi registrado com sucesso.
func handleForwardingRegisterDrone(raftNode *raft.Raft, payload json.RawMessage) error {
	var drone shared.Drone

	if err := json.Unmarshal(payload, &drone); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao unmarshal payload: %v\n", err)
		return err
	}

	LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Novo Registro de Drone Recebido por TCP\n", LClock.GetTime())
	}

	// Preciso fazer isso pq RawMessage nao e considerado []byte.
	newPayload, _ := json.Marshal(drone)

	cmd := shared.HeaderCommand{
		Operation:   OP_REGDRONE,
		Payload:     newPayload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		return fmt.Errorf("\033[1;94m[LOCAL]:\033[0m Erro ao aplicar comando no Raft: %v\n", err)
	}

	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Drone %s registrado com sucesso\n", drone.ID)
	return nil
}

// handleForwardingHeartbeat processa um heartbeat de drone encaminhado por outro setor.
//
// Atualiza o status do drone no sistema via Raft utilizando o comando OP_HEARTBEAT.
//
// Params:
//   - raftNode: instância do Raft para aplicar o comando de heartbeat no consenso.
//   - payload: dados do heartbeat, contendo informações como ID do drone, coordenadas, tempo de Lamport, etc.
func handleForwardingHeartbeat(raftNode *raft.Raft, payload json.RawMessage) {

	// Preciso fazer isso pq RawMessage nao e considerado []byte.
	newPayload := []byte(payload)

	cmd := shared.HeaderCommand{
		Operation:   OP_HEARTBEAT,
		Payload:     newPayload,
		LamportTime: LClock.GetTime(),
	}
	cmdBytes, _ := json.Marshal(cmd)
	raftNode.Apply(cmdBytes, 1*time.Second)
}

// getSigAddr calcula o endereço de sinalização a partir do endereço Raft do líder.
//
// O endereço de sinalização é obtido incrementando a porta do endereço Raft em 1000.
//
// Params:
//   - raftLeader: endereço Raft do líder no formato "host:port".
//
// Returns:
//   - string: endereço de sinalização correspondente ao líder, no formato "host:(port+1000)".
func getSigAddr(raftLeader string) string {

	host, portStr, err := net.SplitHostPort(raftLeader)
	if err != nil {
		return raftLeader
	}

	port, _ := strconv.Atoi(portStr)

	return net.JoinHostPort(host, strconv.Itoa(port+1000))

}

// LeaderInfo contém o endereço do líder do cluster e seu endereço de escuta para sinalização.
type LeaderInfo struct {
	RaftAddr string `json:"raft_addr"`
	SigAddr  string `json:"sig_addr"`
}

// searchForLeaderInfo consulta os peers conhecidos para obter as informações do líder atual do cluster.
//
// Envia uma solicitação QUERY para cada peer e espera pela resposta contendo o endereço do líder e seu endereço de sinalização.
//
// Esta função não precisa de sigPort se os peers já vierem com o endereço completo.
//
// Params:
//   - peers: lista de endereços dos peers para consultar.
//   - sigPort: porta base de sinalização, usada para calcular o endereço de sinalização do líder a partir do endereço Raft.
//
// Returns:
//   - LeaderInfo: informações do líder atual do cluster, incluindo endereço Raft e endereço de sinalização. Retorna um LeaderInfo vazio se nenhum líder for encontrado.
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

		cmd := shared.HeaderCommand{
			Operation: QUERY,
		}

		json.NewEncoder(conn).Encode(cmd)

		var leaderInfo LeaderInfo

		decoder := json.NewDecoder(conn)

		if err := decoder.Decode(&leaderInfo); err == nil && leaderInfo.RaftAddr != "" {
			return leaderInfo
		}

	}
	return LeaderInfo{}
}
