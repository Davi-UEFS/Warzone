package main

import (
	"sync"

	"github.com/Davi-UEFS/Warzone/shared"
)

const (
	OP_ADDI      = "ADD_INCIDENT"
	OP_RMVI      = "REMOVE_INCIDENT"
	OP_ASSIGN    = "ASSIGN_DRONE"
	OP_DEASSIGN  = "DEASSIGN_DRONE"
	OP_UPDATEDRB = "UPDATE_DRONE_BROKER"
)

const (
	QUERY          = "QUERY_LEADER"
	JOIN           = "JOIN_CLUSTER"
	FORWARD        = "FORWARD_INCIDENT"
	SUCCESS        = "SUCESSO: OPERAÇÃO CONCLUÍDA"
	ERR_NOT_LEADER = "ERRO: NÃO É O LIDER"
)

var LClock = shared.LamportClock{
	Time: 0,
	Mu:   sync.Mutex{},
}
