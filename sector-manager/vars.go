package main

import (
	"sync"

	"github.com/Davi-UEFS/Warzone/shared"
	raft "github.com/hashicorp/raft"
)

const (
	OP_ADDREQ    = "ADD_REQUISITION"
	OP_RMVREQ    = "REMOVE_REQUISITION"
	OP_ASSIGN    = "ASSIGN_DRONE"
	OP_DEASSIGN  = "DEASSIGN_DRONE"
	OP_UPDATEDRB = "UPDATE_DRONE_BROKER"
	OP_REGDRONE  = "REGISTER_DRONE"
	OP_DEADDRONE = "DRONE_DEAD"
	OP_HEARTBEAT = "DRONE_HEARTBEAT"
	OP_AGING     = "APPLY_AGING"
)

const (
	QUERY          = "QUERY_LEADER"
	JOIN           = "JOIN_CLUSTER"
	FORWARD_ALR    = "FORWARD_ALERT"
	FORWARD_DONE   = "FORWARD_DONE"
	FORWARD_ASSIGN = "FORWARD_ASSIGN"
	FORWARD_REG    = "FORWARD_REGISTER"
	FORWARD_HB     = "FORWARD_HEARTBEAT"
	SUCCESS        = "SUCESSO: OPERAÇÃO CONCLUÍDA"
	ERR_NOT_LEADER = "ERRO: NÃO É O LIDER"
)

var sectorFSM *RaftFSM

var LClock = shared.LamportClock{
	Time: 0,
	Mu:   sync.Mutex{},
}

var (
	peers      []string
	sigPort    int
	raftNode   *raft.Raft
	brokerAddr string
)
