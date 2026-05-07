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
	result := shared.CommandTemporary{}

	if err := json.Unmarshal(msg.Payload(), &result); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	sensorID := shared.ExtractSensorID(result.OccurrenceID)

	token := client.Publish(fmt.Sprintf("sensors/%s/solved", sensorID), 1, false, []byte("DONE"))
	token.Wait()

}

func createIncidentID(SENSOR_ID string) string {
	return fmt.Sprintf("inc-%s-%d", SENSOR_ID, time.Now().Unix())
}

var onAlertHandler = func(client mqtt.Client, msg mqtt.Message, raftNode *raft.Raft) {

	fmt.Printf("Nova ocorrência: %s\n", string(msg.Payload()))
	alert := shared.Alert{}
	if err := json.Unmarshal(msg.Payload(), &alert); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	LClock.CompareAndUpdate(alert.LamportTime)

	if raftNode.State() != raft.Leader {
		fmt.Println("Sou seguidor, encaminhando alerta para o líder via TCP...")

		leaderInfo := searchForLeaderInfo(peers, sigPort)

		if err := forwardAlert(leaderInfo.SigAddr, shared.HeaderCommand{
			Operation: FORWARD,
			Payload:   msg.Payload(),
		}); err != nil {
			fmt.Printf("Erro ao encaminhar alerta: %v\n", err)
		}

		return
	}

	LClock.Tick()

	reqID := createIncidentID(alert.SensorID)

	requisition := shared.Requisition{
		ID:           reqID,
		Priority:     1, //TODO: DEFINIR PRIORITY MELHOR DEPOIS
		Coord:        alert.Coordinate,
		OriginSector: sectorFSM.GetSector(), //TODO: DEFINIR SECTOR MELHOR DEPOIS
		LamportTime:  LClock.GetTime(),
	}

	newPayload, _ := json.Marshal(requisition)

	cmd := shared.HeaderCommand{
		Operation:   OP_ADDR,
		Payload:     newPayload,
		LamportTime: LClock.GetTime(),
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
