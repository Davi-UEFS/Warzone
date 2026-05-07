package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/raft"
)

type joinReq struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

var onMissionDoneHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Missão concluída: %s\n", string(msg.Payload())) //TODO: EDITAR DEPOIS
	result := shared.Requisition{}

	if err := json.Unmarshal(msg.Payload(), &result); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	sensorID := shared.ExtractSensorID(result.OccurrenceID)

	token := client.Publish(fmt.Sprintf("sensors/%s/solved", sensorID), 1, false, []byte("DONE"))
	token.Wait()

}

var onIncidentHandler = func(client mqtt.Client, msg mqtt.Message, raftNode *raft.Raft) {
	// TODO: NAO ESTOU VERIFICANDO SE E LIDER RAFT. OBSERVAR MELHOR DEPOIS
	fmt.Printf("Nova ocorrência: %s\n", string(msg.Payload()))
	incident := shared.Incident{}
	if err := json.Unmarshal(msg.Payload(), &incident); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	LClock.CompareAndUpdate(incident.LamportTime)
	incident.LamportTime = LClock.GetTime()

	newPayload, _ := json.Marshal(incident)

	cmd := shared.HeaderCommand{
		Operation: OP_ADDI,
		Payload:   newPayload,
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)
	if err := future.Error(); err != nil {
		fmt.Printf("Erro ao aplicar comando ADDI: %v\n", err)
		return
	}

	fmt.Println("Incidente replicado com sucesso no cluster")

}

func sendCommand(client mqtt.Client, payload []byte) {
	//TODO: HARDCODED
	topic := "drones/drone01/missions"
	client.Publish(topic, 2, false, payload)
}
