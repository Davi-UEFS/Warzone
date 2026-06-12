package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
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
}

// onDoneHandler processa o laudo do drone e envia a transação imutável para a blockchain
var onDoneHandler = func(client mqtt.Client, msg mqtt.Message) {
	var result shared.DoneInfo

	if err := json.Unmarshal(msg.Payload(), &result); err != nil {
		fmt.Printf("\033[1;91m[MQTT ERROR]:\033[0m Falha ao decodificar DoneInfo: %v\n", err)
		return
	}

	fmt.Printf("\033[1;94m[MQTT]:\033[0m Drone %s concluiu a requisição %s\n", result.DroneID, result.RequisitionID)

	// Libera o drone no estado local imediatamente
	sectorState.Mu.Lock()
	if drone, exists := sectorState.DroneMap[result.DroneID]; exists {
		drone.Status = shared.DRONE_IDLE
	}
	sectorState.Mu.Unlock()

	// 1. Envia o laudo para a blockchain
	go enviarLaudoParaBlockchain(result.RequisitionID, result.DroneID, "Missao concluida via MQTT")

	// 2. Remove a requisição da fila ativa e liberta o drone na blockchain (RmvReq)
	go enviarRmvReqParaBlockchain(result.RequisitionID, result.DroneID, "Concluido com sucesso")
}

func createIncidentID(SENSOR_ID string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomPart := r.Int63n(1000000)
	return fmt.Sprintf("inc--%s--%06d", SENSOR_ID, randomPart)
}

// onAlertHandler recebe o alerta do sensor e registra a nova requisição na blockchain
var onAlertHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("\033[1;94m[MQTT]:\033[0m Novo alerta de sensor recebido")

	var alert shared.Alert
	if err := json.Unmarshal(msg.Payload(), &alert); err != nil {
		fmt.Printf("\033[1;91m[MQTT ERROR]:\033[0m Erro ao decodificar Alerta: %v\n", err)
		return
	}

	reqID := createIncidentID(alert.SensorID)
	go enviarRequisicaoParaBlockchain(reqID, alert)
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

	// Salva na memória local
	sectorState.Mu.Lock()
	sectorState.DroneMap[drone.ID] = &drone
	sectorState.Mu.Unlock()

	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Drone %s registrado na RAM. Sincronizando com Blockchain...\n", drone.ID)

	// Registra o drone no Censo Global da Blockchain
	// Convertendo a bateria para string (ajuste se sua struct de bateria for diferente)
	batteryStr := fmt.Sprintf("%f", drone.BatteryLevel)
	go enviarRegDroneParaBlockchain(drone.ID, sectorID, batteryStr)
}

// onHeartbeatHandler atualiza o timestamp de vida do drone (TTL)
var onHeartbeatHandler = func(client mqtt.Client, msg mqtt.Message) {
	droneID := string(msg.Payload())

	sectorState.Mu.Lock()
	if drone, exists := sectorState.DroneMap[droneID]; exists {
		drone.LastSeen = time.Now().Unix()
	}
	sectorState.Mu.Unlock()
}
