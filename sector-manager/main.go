package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

func main() {
	// 1. Leitura de Flags de configuração
	nodeIDFlag := flag.String("id", "node1", "ID único deste nó/setor")
	brokerPortFlag := flag.Int("broker-port", 1883, "Porta do broker MQTT")
	dashboardPortFlag := flag.Int("dashboard-port", 8080, "Porta do dashboard HTTP")
	debugFlag := flag.Bool("debug", false, "Ativa simulações de latência, aging e logs extras")
	flag.Parse()

	DebugMode = *debugFlag

	// Como o broker é embutido, usamos sempre 127.0.0.1
	brokerAddr := shared.NormalizeBrokerAddr(net.JoinHostPort("127.0.0.1", strconv.Itoa(*brokerPortFlag)))

	// 2. Inicializa o Broker MQTT Embutido
	startEmbeddedBroker(*brokerPortFlag)

	// 3. Inicializa o Cliente MQTT Local (Paho)
	var err error
	globalClient, err = shared.MakeClient(brokerAddr, *nodeIDFlag+"-client", onConnect, false)
	if err != nil {
		log.Fatalf("\033[1;31m[ERRO CRÍTICO]\033[0m Falha ao conectar cliente MQTT: %v\n", err)
	}

	fmt.Printf("Broker:            Dashboard:           Debug Mode:\n")
	fmt.Printf("%s              %d                    %t\n", brokerAddr, *dashboardPortFlag, *debugFlag)

	// 4. Inicia as rotinas paralelas (Dashboard e o Coração do Sistema)
	//go startDashboardServer(*dashboardPortFlag)
	go startManagerLoop()

	// 5. Trava a thread principal para o servidor não morrer
	select {}
}

// startManagerLoop substitui o antigo startDispatcher.
// Ele é o maestro que chama os nossos novos ficheiros refatorados no tempo exato.
func startManagerLoop() {
	pollingTicker := time.NewTicker(2 * time.Second)   // Olheiro: Vai à Blockchain
	watchdogTicker := time.NewTicker(10 * time.Second) // Ceifeiro: Procura mortos
	agingTicker := time.NewTicker(20 * time.Second)    // Fila: Aplica o envelhecimento
	dispatchTicker := time.NewTicker(1 * time.Second)  // Maestro: Tenta despachar

	log.Println("\033[1;32m[MANAGER]\033[0m Orquestrador principal iniciado!")

	for {
		select {
		case <-pollingTicker.C:
			// Puxa a verdade global (poller.go)
			SyncStateWithBlockchain()

		case <-watchdogTicker.C:
			// Corta as cabeças dos inativos (watchdog.go)
			RunWatchdog()

		case <-agingTicker.C:
			// Envelhece as missões na fila (state.go)
			GlobalState.Mu.Lock()
			GlobalState.PendingReqsQueue.ApplyAging(time.Now().Unix(), 20, 1)
			GlobalState.Mu.Unlock()

		case <-dispatchTicker.C:
			// Cruza os dados e envia os drones (dispatcher.go)
			// Rodar a cada 1 segundo garante respostas ultrarrápidas,
			// sem depender da lentidão da Blockchain!
			ProcessRequisitions()
		}
	}
}
