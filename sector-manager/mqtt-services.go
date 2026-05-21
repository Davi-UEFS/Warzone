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

type joinReq struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

type forwardedAlert struct {
	Alert        shared.Alert `json:"alert"`
	OriginSector string       `json:"origin_sector"`
}

var onConnect = func(client mqtt.Client) {
	fmt.Println("\033[1;94m[LOCAL]:\033[0m Conectado ao broker local")
	fmt.Println("\033[1;94m[LOCAL]:\033[0m Se inscrevendo nos tópicos...")

	client.Subscribe("sensors/+/incidents", 1, onAlertHandler)
	client.Subscribe("drones/+/done", 1, onDoneHandler)
	client.Subscribe("drones/register", 1, onNewDroneHandler)
	client.Subscribe("drones/+/heartbeat", 1, onHeartbeatHandler)
}

var onDoneHandler = func(client mqtt.Client, msg mqtt.Message) {

	var result shared.DoneInfo

	if err := json.Unmarshal(msg.Payload(), &result); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao unmarshal payload: %v\n", err)
		return
	}

	if raftNode.State() != raft.Leader {
		fmt.Println("\033[1;94m[LOCAL]:\033[0m Sou seguidor, encaminhando resultado para o líder via TCP...")

		leaderInfo := searchForLeaderInfo(peers, sigPort)
		if err := forwardCommand(leaderInfo.SigAddr, shared.HeaderCommand{
			Operation: FORWARD_DONE,
			Payload:   msg.Payload(),
		}); err != nil {
			fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao encaminhar resultado: %v\n", err)
		}
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

func createIncidentID(SENSOR_ID string) string {
	randomPart := rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(1000000)
	return fmt.Sprintf("inc--%s--%06d", SENSOR_ID, randomPart)
}

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

		leaderInfo := searchForLeaderInfo(peers, sigPort)

		forwardPayload, err := json.Marshal(forwardedAlert{
			Alert:        alert,
			OriginSector: sectorFSM.GetSector(),
		})
		if err != nil {
			fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao serializar forward alert: %v\n", err)
			return
		}

		if err := forwardCommand(leaderInfo.SigAddr, shared.HeaderCommand{
			Operation: FORWARD_ALR,
			Payload:   forwardPayload,
		}); err != nil {
			fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro ao encaminhar alerta: %v\n", err)
		}

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
	// Simula um atraso na rede para testar a resiliência do sistema e a sincronização de Lamport.
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

		leaderInfo := searchForLeaderInfo(peers, sigPort)

		if err := forwardCommand(leaderInfo.SigAddr, shared.HeaderCommand{
			Operation: FORWARD_REG,
			Payload:   payload,
		}); err != nil {
			tellRegError(drone.ID, "Sem líder no momento. Aguarde...")
		}

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
