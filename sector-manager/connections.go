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

	case FORWARD:
		if raftNode.State() != raft.Leader {
			json.NewEncoder(conn).Encode(ERR_NOT_LEADER)
			return
		}

		err := handleForwardingIncident(raftNode, cmd.Payload)

		if err == nil {
			json.NewEncoder(conn).Encode(SUCCESS)
		}
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

func sendForwardingIncident(sigAddr string, cmd shared.HeaderCommand) error {
	conn, err := net.DialTimeout("tcp", sigAddr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	var incident shared.Incident
	if err := json.Unmarshal(cmd.Payload, &incident); err != nil {
		fmt.Printf("Erro ao desserializar incidente: %v\n", err)
		return err
	}

	LClock.CompareAndUpdate(incident.LamportTime)
	incident.LamportTime = LClock.GetTime()

	newPayload, _ := json.Marshal(incident)

	cmd.Payload = newPayload

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

func handleForwardingIncident(raftNode *raft.Raft, payload json.RawMessage) error {

	var incident shared.Incident

	if err := json.Unmarshal(payload, &incident); err != nil {
		fmt.Printf("Erro ao desserializar incidente: %v\n", err)
		return err
	}

	LClock.CompareAndUpdate(incident.LamportTime)
	incident.LamportTime = LClock.GetTime()

	newPayload, _ := json.Marshal(incident)

	cmd := shared.HeaderCommand{
		Operation: OP_ADDI,
		Payload:   newPayload,
	}

	cmdData, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdData, 5*time.Second)

	if err := future.Error(); err != nil {
		fmt.Println("Falha ao adicionar incidente")
		return err
	}

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
