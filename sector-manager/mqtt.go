package main

import (
	"encoding/json"
	"fmt"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var onMissionDoneHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Missão concluída: %s\n", string(msg.Payload()))
	result := shared.MissionResult{}

	if err := json.Unmarshal(msg.Payload(), &result); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	sensorID := shared.ExtractSensorID(result.OccurrenceID)

	token := client.Publish(fmt.Sprintf("sensors/%s/solved", sensorID), 1, false, []byte("DONE"))
	token.Wait()

}

var onIncidentHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Nova ocorrência: %s\n", string(msg.Payload()))

	incident := shared.Incident{}
	if err := json.Unmarshal(msg.Payload(), &incident); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	cmd := shared.DroneCommand{
		OccurrenceID: incident.OccurrenceID,
		Action:       "oil",
		Timestamp:    incident.Timestamp,
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		fmt.Printf("Erro ao marshal comando: %v\n", err)
		return
	}

	sendCommand(client, payload)

}
