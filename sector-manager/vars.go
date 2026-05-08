package main

import (
	"sync"

	"github.com/Davi-UEFS/Warzone/shared"
)

const (
	OP_ADDR      = "ADD_REQUISITION"
	OP_RMVR      = "REMOVE_REQUISITION"
	OP_ASSIGN    = "ASSIGN_DRONE"
	OP_DEASSIGN  = "DEASSIGN_DRONE"
	OP_UPDATEDRB = "UPDATE_DRONE_BROKER"
)

const (
	QUERY          = "QUERY_LEADER"
	JOIN           = "JOIN_CLUSTER"
	FORWARD_ALR    = "FORWARD_ALERT"
	FORWARD_DONE   = "FORWARD_DONE"
	SUCCESS        = "SUCESSO: OPERAÇÃO CONCLUÍDA"
	ERR_NOT_LEADER = "ERRO: NÃO É O LIDER"
)

var sectorFSM = &RaftFSM{
	DroneMap:         make(map[string]shared.Drone),
	PendingReqsQueue: []shared.Requisition{},
}

var LClock = shared.LamportClock{
	Time: 0,
	Mu:   sync.Mutex{},
}

var (
	peers   []string
	sigPort int
)
