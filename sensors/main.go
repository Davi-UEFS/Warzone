package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func getEnviromentVariables() (string, string, string) {
	return os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID"), os.Getenv("SENSOR_TYPE")
}

func createIncidentID(CLIENT_ID string) string {
	cnt := commandCounter.Add(1)
	return fmt.Sprintf("%s-%d", CLIENT_ID, cnt)
}

func createIncidentPayload(SENSOR_TYPE string, priority int, incidentID string) ([]byte, error) {

	incident := shared.Incident{
		ID:          incidentID,
		Priority:    priority,
		LamportTime: LClock.GetTime(),
		Coord: shared.Coordinate{
			Longitude: 100,
			Latitude:  100,
		},
	}

	return json.Marshal(incident)

}

/////////////////////////////////////////////////////
//////////////////// VARS ///////////////////////////
/////////////////////////////////////////////////////

var LClock = &shared.LamportClock{
	Time: 0,
	Mu:   sync.Mutex{},
}

var commandCounter atomic.Int64

var solvedSignal = make(chan struct{}, 1)

var onSolvedHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Incidente resolvido")

	var solvedInfo shared.SolvedInfo
	if err := json.Unmarshal(msg.Payload(), &solvedInfo); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	LClock.CompareAndUpdate(solvedInfo.LCTime)
	fmt.Printf("Lamport clock atualizado: %d\n", LClock.GetTime())

	solvedSignal <- struct{}{}

}

//////////////////////////////////////////

func main() {

	BROKER_IP, CLIENT_ID, SENSOR_TYPE := getEnviromentVariables()
	TOPIC := fmt.Sprintf("sensors/%s/incidents", CLIENT_ID)

	client, _ := shared.MakeClient(BROKER_IP, CLIENT_ID)
	client.Subscribe(fmt.Sprintf("sensors/%s/solved", CLIENT_ID), 1, onSolvedHandler)

	var trigger bool
	trigger = false

	for {
		if !trigger {
			trigger = generateIncident()
		} else {
			LClock.Tick()
			incidentID := createIncidentID(CLIENT_ID)
			payload, _ := createIncidentPayload(SENSOR_TYPE, 1, incidentID) //TODO: PRIORITY HARDCODED

			token := client.Publish(TOPIC, 1, false, payload)
			token.Wait()

			fmt.Println("Ocorrência enviada ao gerenciador.")

			<-solvedSignal
			trigger = false

			fmt.Println("Ocorrência resolvida. Começando de novo...")

		}

		time.Sleep(time.Second)

	}

}
