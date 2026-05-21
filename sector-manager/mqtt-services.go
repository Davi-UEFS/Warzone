package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/raft"
)

// Struct usada para enviar um pedido de join para o líder via TCP
type joinReq struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

// Struct usada para encaminhar um alerta recebido por MQTT para o líder via TCP.
// Contém o alerta original e o setor de origem para que o líder possa processar corretamente.
type forwardedAlert struct {
	Alert        shared.Alert `json:"alert"`
	OriginSector string       `json:"origin_sector"`
}

// Handler chamado quando o cliente MQTT se conecta ao broker.
// Ele se inscreve nos tópicos relevantes para receber alertas,
//
//	resultados de missões, registros de drones e heartbeats.
var onConnect = func(client mqtt.Client) {
	fmt.Println("\033[1;94m[LOCAL]:\033[0m Conectado ao broker local")
	fmt.Println("\033[1;94m[LOCAL]:\033[0m Se inscrevendo nos tópicos...")

	client.Subscribe("sensors/+/incidents", 1, onAlertHandler)
	client.Subscribe("drones/+/done", 1, onDoneHandler)
	client.Subscribe("drones/register", 1, onNewDroneHandler)
	client.Subscribe("drones/+/heartbeat", 1, onHeartbeatHandler)
}

// Handler chamado quando um resultado de missão é publicado por um drone no tópico.
// Ele processa o resultado e atualiza o estado do sistema. Se o nó atual não for o líder, ele encaminha o resultado para o líder via TCP.
//
// Atualiza o relógio de Lamport.
var onDoneHandler = func(client mqtt.Client, msg mqtt.Message) {

	var result shared.DoneInfo

	if err := json.Unmarshal(msg.Payload(), &result); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao unmarshal payload: %v\n", err)
		return
	}

	if raftNode.State() != raft.Leader {
		fmt.Println("\033[1;94m[LOCAL]:\033[0m Sou seguidor, encaminhando resultado para o líder via TCP...")

		dispatchForwarding(shared.HeaderCommand{
			Operation: FORWARD_DONE,
			Payload:   msg.Payload(),
		}, "Conclusão de Missão", nil)

		return
	}

	LClock.CompareAndUpdate(result.LCTime)
	LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Done recebido do drone (MQTT)\n", LClock.GetTime())
	}

	cmd := shared.HeaderCommand{
		Operation:   OP_RMVREQ,
		Payload:     msg.Payload(),
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao aplicar comando no Raft: %v\n", err)
	} else {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Drone %s liberado da missão %s\n", result.DroneID, result.RequisitionID)
	}
}

// Gera um ID único para o incidente com base no ID do sensor e um número aleatório.
//
// Returns:
//   - string: O ID do incidente no formato "inc--SENSOR_ID--RANDOM", onde RANDOM é um número de 6 dígitos.
func createIncidentID(SENSOR_ID string) string {
	randomPart := rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(1000000)
	return fmt.Sprintf("inc--%s--%06d", SENSOR_ID, randomPart)
}

// Handler chamado quando um alerta é publicado por um sensor no tópico MQTT.
// Ele processa o alerta e cria uma requisição correspondente. Se o nó atual não for o líder, ele encaminha o alerta para o líder via TCP.
//
// Atualiza o relógio de Lamport.
var onAlertHandler = func(client mqtt.Client, msg mqtt.Message) {

	fmt.Println("\033[1;94m[LOCAL]:\033[0m Novo alerta chegou por MQTT")

	alert := shared.Alert{}
	if err := json.Unmarshal(msg.Payload(), &alert); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao unmarshal payload: %v\n", err)
		return
	}

	LClock.CompareAndUpdate(alert.LamportTime)

	if raftNode.State() != raft.Leader {
		fmt.Println("\033[1;94m[LOCAL]:\033[0m Sou seguidor, encaminhando alerta para o líder via TCP...")

		forwardPayload, _ := json.Marshal(forwardedAlert{
			Alert:        alert,
			OriginSector: sectorFSM.GetSector(),
		})

		dispatchForwarding(shared.HeaderCommand{
			Operation: FORWARD_ALR,
			Payload:   forwardPayload,
		}, "Alerta de Incidente", nil)

		return
	}

	LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Novo Alerta Recebido (MQTT)\n", LClock.GetTime())
	}

	reqID := createIncidentID(alert.SensorID)

	requisition := shared.Requisition{
		ID:           reqID,
		Priority:     PRIOTIRIES[alert.Type], //Prioridade baseada no tipo de alerta
		Type:         alert.Type,
		Coord:        alert.Coordinate,
		OriginSector: sectorFSM.GetSector(),
		LamportTime:  LClock.GetTime(),
		CreatedAt:    time.Now().Unix(),
	}

	newPayload, _ := json.Marshal(requisition)

	cmd := shared.HeaderCommand{
		Operation:   OP_ADDREQ,
		Payload:     newPayload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	// ----------------------------------------------------
	// Simulação de latência para o sensor "sensor-lento"
	// Esta seção não é necessária para o funcionamento.
	// Simula um atraso para testar a sincronização de Lamport.
	// ----------------------------------------------------
	if DebugMode && alert.SensorID == "sensor-lento" {
		log.Printf("[DEBUG-LAMPORT] Interceptado alerta do %s. Retardando envio ao Raft por 10 segundos...\n", alert.SensorID)

		go func(cmdBytesToApply []byte, reqID string) {
			time.Sleep(10 * time.Second)
			log.Printf("[DEBUG-LAMPORT] Tempo esgotado! Injetando %s no Raft.\n", reqID)
			raftNode.Apply(cmdBytesToApply, 5*time.Second)
		}(cmdBytes, requisition.ID)

		return
	}

	future := raftNode.Apply(cmdBytes, 5*time.Second)
	if err := future.Error(); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao aplicar comando ADDI: %v\n", err)
		return
	}

	fmt.Println("\033[1;94m[LOCAL]:\033[0m Incidente replicado com sucesso no cluster")

}

// Handler chamado quando um novo drone se registra publicando no tópico MQTT.
// Ele processa o registro e adiciona o drone ao sistema. Se o nó atual não for o líder, ele encaminha o registro para o líder via TCP.
//
// Atualiza o relógio de Lamport.
var onNewDroneHandler = func(client mqtt.Client, msg mqtt.Message) {

	var drone shared.Drone

	if err := json.Unmarshal(msg.Payload(), &drone); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao unmarshal payload: %v\n", err)
		return
	}

	drone.SetPhysicalLocation(sectorFSM.GetSector(), brokerAddr)

	payload, _ := json.Marshal(drone)

	if raftNode.State() != raft.Leader {
		fmt.Println("\033[1;94m[LOCAL]:\033[0m Sou seguidor, encaminhando registro de drone para o líder via TCP...")

		dispatchForwarding(shared.HeaderCommand{
			Operation: FORWARD_REG,
			Payload:   payload,
		}, "Registro de Drone", func() {
			tellRegError(drone.ID, "Sem líder no momento. Aguarde e tente novamente.")
		})

		// Se um erro ocorrer, a função de callback irá informar o drone sobre o erro de registro.

		return
	}

	LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Novo Registro de Drone Recebido (MQTT)\n", LClock.GetTime())
	}

	cmd := shared.HeaderCommand{
		Operation:   OP_REGDRONE,
		Payload:     payload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)
	if err := future.Error(); err != nil {
		log.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao aplicar comando REGDRONE: %v\n", err)
		tellRegError(drone.ID, "Erro interno do Raft. Tente registrar novamente.")
		return
	}

	fmt.Println("\033[1;94m[LOCAL]:\033[0m Novo drone registrado com sucesso no cluster")

}

// Handler chamado quando um heartbeat é publicado por um drone no tópico MQTT.
// Ele processa o heartbeat e atualiza o estado do sistema. Se o nó atual não for o líder, ele encaminha o heartbeat para o líder via TCP.
var onHeartbeatHandler = func(client mqtt.Client, msg mqtt.Message) {

	if raftNode.State() != raft.Leader {
		// Encaminha para o líder via TCP
		leaderInfo := searchForLeaderInfo(peers, sigPort)
		if err := forwardCommand(leaderInfo.SigAddr, shared.HeaderCommand{
			Operation: FORWARD_HB,
			Payload:   msg.Payload(),
		}); err != nil {
			// Ignoramos o erro de log para não fludar o terminal
		}
		return
	}

	// É o líder, então aplica o pulso na FSM
	cmd := shared.HeaderCommand{
		Operation:   OP_HEARTBEAT,
		Payload:     msg.Payload(),
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)
	raftNode.Apply(cmdBytes, 1*time.Second)
}

// publishDrones escuta o canal de eventos de missão e publica os eventos para os drones via MQTT.
//
//	Ele é executado em uma goroutine separada para não bloquear o loop principal do MQTT.
//
// Params:
//   - eventsChan: Canal onde os eventos de missão.
//   - client: Cliente MQTT para publicar os eventos.
func publishToDrones(eventsChan chan MissionPublishEvent, client mqtt.Client) {
	for {
		event := <-eventsChan
		token := client.Publish(event.Topic, event.Qos, false, event.Payload)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao publicar evento para drone: %v\n", token.Error())
		} else {
			fmt.Printf("\033[1;94m[LOCAL]:\033[0m Evento publicado para drone no tópico %s\n", event.Topic)
		}
	}
}

// tellRegError publica uma mensagem de erro de registro para um drone específico no tópico MQTT.
//
// Params:
//   - droneID: ID do drone para o qual a mensagem de erro deve ser publicada.
//   - errMsg: Mensagem de erro a ser enviada.
func tellRegError(droneID string, errMsg string) {
	errorMessage := shared.RegErrorMessage{
		DroneID: droneID,
		Error:   errMsg,
	}

	payload, _ := json.Marshal(errorMessage)

	// TODO: PENSAR NUMA LOGICA MELHOR DEPOIS. USAR GLOBAL É PEBA.
	token := globalClient.Publish("drones/"+droneID+"/reg_error", 1, false, payload)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao publicar mensagem de erro de registro para drone %s: %v\n", droneID, token.Error())
	} else {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Mensagem de erro de registro publicada para drone %s\n", droneID)
	}
}
