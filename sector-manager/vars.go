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

// Dicionário de cobrança: Nome do País -> Endereço Público na Blockchain
var EnderecosPaises = map[string]string{
	"alice": "cosmos1w0dl36f7uumjqxn9899h077jups7083a06ex2l", // Carteira de teste da Alice
	"bob":   "cosmos1rp9qpnj2t75z8708gkgsx7xadwm9vewkwvrs2d", // Carteira de teste do Bob
}

// Variáveis globais usadas pelo setor manager.
var (
	globalClient mqtt.Client
	brokerAddr   string
	DebugMode    bool
)
