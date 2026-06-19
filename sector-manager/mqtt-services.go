package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// onConnect inscreve o Manager nos tópicos relevantes do Setor
var onConnect = func(client mqtt.Client) {
	fmt.Println("\033[1;94m[MQTT]:\033[0m Conectado ao broker local")
	client.Subscribe("sensors/+/incidents", 1, onAlertHandler)
	client.Subscribe("drones/+/done", 1, onDoneHandler)
	client.Subscribe("drones/register", 1, onNewDroneHandler)
	client.Subscribe("drones/+/heartbeat", 1, onHeartbeatHandler)
	client.Subscribe("finance/transfer", 1, onTransferHandler)
}

// onTransferHandler permite que você dispare transferências via MQTT
var onTransferHandler = func(client mqtt.Client, msg mqtt.Message) {
	var req shared.TransferRequest
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		fmt.Printf("\033[1;91m[MQTT ERROR]:\033[0m Falha ao decodificar TransferRequest: %v\n", err)
		return
	}

	fmt.Printf("\033[1;94m[MQTT]:\033[0m Recebido pedido de transferência de %s para %s\n", req.FromAlias, req.ToAddress)

	// Chama a função que criamos no blockchainclient.go
	go func() {
		err := enviarTransferenciaParaBlockchain(req.FromAlias, req.ToAddress, req.Amount)
		if err != nil {
			fmt.Printf("\033[1;91m[MQTT ERROR]:\033[0m Falha na transferência via Blockchain: %v\n", err)
		} else {
			fmt.Printf("\033[1;92m[MQTT]:\033[0m Transferência executada com sucesso!\n")
		}
	}()
}

// onDoneHandler processa o laudo do drone e envia a transação imutável para a blockchain
var onDoneHandler = func(client mqtt.Client, msg mqtt.Message) {
	var result shared.DoneInfo

	if err := json.Unmarshal(msg.Payload(), &result); err != nil {
		fmt.Printf("\033[1;91m[MQTT ERROR]:\033[0m Falha ao decodificar DoneInfo: %v\n", err)
		return
	}

	fmt.Printf("\033[1;94m[MQTT]:\033[0m Drone %s concluiu a requisição %s no terreno. Iniciando burocracia blockchain...\n", result.DroneID, result.RequisitionID)

	// 1. Extrai APENAS O NÚMERO do ID (Transforma "inc--Setor-A--0" em "0")
	partes := strings.Split(result.RequisitionID, "--")
	idNumerico := partes[len(partes)-1]

	// 2. Agrupa as transações NUMA ÚNICA goroutine.
	go func() {
		// A. Envia o laudo e espera
		enviarLaudoParaBlockchain(idNumerico, result.DroneID, "Missao concluida via MQTT")

		// B. Intervalo de segurança para garantir que o bloco do laudo foi processado
		time.Sleep(2 * time.Second)

		// C. Remove a requisição antiga na blockchain (O que forçaria o IDLE na rede)
		enviarRmvReqParaBlockchain(idNumerico, result.DroneID, "Concluido com sucesso")

		// =========================================================================
		// A CORREÇÃO: O drone SÓ FICA LIVRE na RAM local após a blockchain fechar a missão!
		// =========================================================================
		GlobalState.Mu.Lock()
		if drone, exists := GlobalState.DroneMap[result.DroneID]; exists {
			drone.Status = shared.DRONE_IDLE
		}
		GlobalState.Mu.Unlock()

		fmt.Printf("\033[1;32m[MANAGER]\033[0m Burocracia da requisição %s finalizada. Drone %s liberado para nova missão!\n", idNumerico, result.DroneID)
	}()
}

// onAlertHandler recebe o alerta do sensor e regista a nova requisição na blockchain
var onAlertHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("\033[1;94m[MQTT]:\033[0m Novo alerta de sensor recebido")

	var alert shared.Alert
	if err := json.Unmarshal(msg.Payload(), &alert); err != nil {
		fmt.Printf("\033[1;91m[MQTT ERROR]:\033[0m Erro ao descodificar Alerta: %v\n", err)
		return
	}

	// Enviamos apenas o alerta para a rede. A Blockchain gera o ID oficial!
	go enviarRequisicaoParaBlockchain(alert)
}

// onNewDroneHandler registra o drone na memória local E na blockchain
var onNewDroneHandler = func(client mqtt.Client, msg mqtt.Message) {
	var drone shared.Drone

	if err := json.Unmarshal(msg.Payload(), &drone); err != nil {
		fmt.Printf("\033[1;91m[MQTT ERROR]:\033[0m Erro ao decodificar registro de Drone: %v\n", err)
		return
	}

	sectorID := os.Getenv("SECTOR_ID")
	if sectorID == "" {
		sectorID = "Setor-A"
	}
	drone.SetPhysicalLocation(sectorID, brokerAddr)
	drone.LastSeen = time.Now().Unix()
	drone.Status = shared.DRONE_IDLE
	drone.Verified = true // O drone acabou de falar connosco no MQTT, logo é real!

	// Salva no NOVO COFRE local
	GlobalState.Mu.Lock()
	GlobalState.DroneMap[drone.ID] = &drone
	GlobalState.Mu.Unlock()

	//TODO: DEBUG
	fmt.Println(drone)

	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Drone %s registrado na RAM. Sincronizando com Blockchain...\n", drone.ID)

	// Registra o drone no Censo Global da Blockchain
	// Convertendo a bateria para string
	batteryStr := fmt.Sprintf("%d", drone.BatteryLevel)
	go enviarRegDroneParaBlockchain(drone.ID, sectorID, batteryStr)
}

// onHeartbeatHandler atualiza o timestamp de vida do drone (TTL)
var onHeartbeatHandler = func(client mqtt.Client, msg mqtt.Message) {
	var droneHeartbeat shared.DroneHeartbeat
	if err := json.Unmarshal(msg.Payload(), &droneHeartbeat); err != nil {
		fmt.Printf("\033[1;91m[MQTT ERROR]:\033[0m Erro ao decodificar Heartbeat: %v\n", err)
		return
	}

	GlobalState.Mu.Lock() // Tranca o NOVO cofre

	if drone, exists := GlobalState.DroneMap[droneHeartbeat.ID]; exists {
		drone.LastSeen = time.Now().Unix()
		drone.Verified = true // O drone provou que está vivo!
		drone.BatteryLevel = droneHeartbeat.BatteryLevel
	}

	GlobalState.Mu.Unlock() // Destranca imediatamente
}
