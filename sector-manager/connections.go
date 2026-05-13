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

func handleConnection(conn net.Conn, raftNode *raft.Raft) {
	defer conn.Close()

	var cmd shared.HeaderCommand

	if err := json.NewDecoder(conn).Decode(&cmd); err != nil {
		return
	}

	switch cmd.Operation {
	case QUERY:
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

	default:
		fmt.Printf("Operação desconhecida recebida via sinalização: %s\n", cmd.Operation)
		json.NewEncoder(conn).Encode("Operação desconhecida")
	}

}

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
func handleJoinRequest(raftNode *raft.Raft, payload json.RawMessage) error {

	var req joinReq

	if err := json.Unmarshal(payload, &req); err != nil {
		fmt.Printf("Erro ao desserializar join request: %v\n", err)
		return err
	}

	future := raftNode.AddVoter(raft.ServerID(req.ID), raft.ServerAddress(req.Addr), 0, 0)
	if err := future.Error(); err != nil {
		fmt.Printf("Falha ao adicionar nó %s ao consenso: %v\n", req.ID, err)
		return err
	}

	fmt.Printf("Nó %s integrado com sucesso!\n", req.ID)

	return nil

}

func forwardAlert(sigAddr string, cmd shared.HeaderCommand) error {
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

func forwardDone(sigAddr string, cmd shared.HeaderCommand) error {
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

func forwardRegisterDrone(sigAddr string, cmd shared.HeaderCommand) error {
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

func handleForwardingAlert(raftNode *raft.Raft, payload json.RawMessage) error {

	var fwd forwardedAlert
	if err := json.Unmarshal(payload, &fwd); err != nil {
		fmt.Printf("Erro ao desserializar alerta: %v\n", err)
		return err
	}

	alert := fwd.Alert
	originSector := fwd.OriginSector
	if originSector == "" {
		originSector = sectorFSM.GetSector()
	}

	LClock.CompareAndUpdate(alert.LamportTime)
	LClock.Tick()

	reqID := createIncidentID(alert.SensorID)

	requisition := shared.Requisition{
		ID:           reqID,
		Priority:     1, //TODO: DEFINIR PRIORITY MELHOR DEPOIS
		Coord:        alert.Coordinate,
		OriginSector: originSector,
		LamportTime:  LClock.GetTime(),
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
		fmt.Println("Falha ao adicionar requisição ao consenso: ", err)
		return err
	}

	fmt.Printf("Requisição criada para o sensor %s: %s\n", alert.SensorID, reqID)

	return nil

}

func handleForwardingDone(raftNode *raft.Raft, payload json.RawMessage) error {
	var result shared.DoneInfo

	if err := json.Unmarshal(payload, &result); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return err
	}

	LClock.CompareAndUpdate(result.LCTime)
	LClock.Tick()

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
		return fmt.Errorf("Erro ao aplicar comando no Raft: %v\n", err)
	}

	fmt.Printf("Drone %s liberado da missão %s\n", result.DroneID, result.RequisitionID)
	return nil

}

func handleForwardingRegisterDrone(raftNode *raft.Raft, payload json.RawMessage) error {
	var drone shared.Drone

	if err := json.Unmarshal(payload, &drone); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return err
	}

	LClock.Tick()

	newPayload, _ := json.Marshal(drone)

	cmd := shared.HeaderCommand{
		Operation:   OP_REGDRONE,
		Payload:     newPayload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		return fmt.Errorf("Erro ao aplicar comando no Raft: %v\n", err)
	}

	fmt.Printf("Drone %s registrado com sucesso\n", drone.ID)
	return nil
}

func getSigAddr(raftLeader string) string {

	host, portStr, err := net.SplitHostPort(raftLeader)
	if err != nil {
		return raftLeader
	}

	port, _ := strconv.Atoi(portStr)

	return net.JoinHostPort(host, strconv.Itoa(port+1000))

}

type LeaderInfo struct {
	RaftAddr string `json:"raft_addr"`
	SigAddr  string `json:"sig_addr"`
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
