package main

import (
	"sync"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	raft "github.com/hashicorp/raft"
)

// Constantes usadas para comandos do Raft.
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

// Constantes usadas para os tipos de mensagens de encaminhamento.
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

// PRIOTIRIES é um mapa que define a prioridade de cada tipo de requisição.
var PRIOTIRIES = map[string]int{
	shared.OIL:            4,
	shared.FIRE:           5,
	shared.WRECKAGE:       3,
	shared.INSPECTION:     1,
	shared.UNKNOWN_OBJECT: 2,
	shared.BOTTLENECK:     2,
}

// Variáveis globais usadas pelo setor manager.
var (
	sectorFSM    *RaftFSM            //TODO: DEPRECATED. TIRAR DEPOIS
	LClock       shared.LamportClock //TODO: DEPRECATED. TIRAR DEPOIS
	peers        []string            //TODO: DEPRECATED. TIRAR DEPOIS
	sigPort      int                 //TODO: DEPRECATED. TIRAR DEPOIS
	globalClient mqtt.Client
	raftNode     *raft.Raft //TODO: DEPRECATED. TIRAR DEPOIS
	brokerAddr   string
	DebugMode    bool
)

// LocalState substitui a antiga FSM.
type LocalState struct {
	Mu               sync.Mutex
	PendingReqsQueue ReqHeap
	DroneMap         map[string]*shared.Drone
}

// Inicializamos o estado local
var sectorState = &LocalState{
	DroneMap: make(map[string]*shared.Drone),
}
