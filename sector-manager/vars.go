package main

import (
	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
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
	globalClient mqtt.Client
	brokerAddr   string
	DebugMode    bool
)
