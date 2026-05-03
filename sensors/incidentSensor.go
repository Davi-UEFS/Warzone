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

func getEnviromentVariables() (string, string, string) {
	return os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID"), os.Getenv("SENSOR_TYPE")
}

var solvedSignal = make(chan struct{}, 1)

var onSolvedHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Incidente resolvido")
	solvedSignal <- struct{}{}

}

// Retorna incidente com chance de 10%
func generateIncident() bool {
	chance := rand.Float64()
	if chance <= 0.1 {
		return true
	}

	return false

}

func createIncidentID(CLIENT_ID string) string {
	cnt := commandCounter.Add(1)
	return fmt.Sprintf("%s-%d", CLIENT_ID, cnt)
}

func createIncidentPayload(SENSOR_TYPE string, incidentID string) ([]byte, error) {

	incident := shared.Incident{
		ID:        incidentID,
		Message:   INCIDENT_MESSAGES[SENSOR_TYPE],
		Timestamp: time.Now(),
	}

	return json.Marshal(incident)

}

func main() {

	BROKER_IP, CLIENT_ID, SENSOR_TYPE := getEnviromentVariables()
	TOPIC := fmt.Sprintf("sensors/%s/incidents", CLIENT_ID)

	client := shared.MakeClient(BROKER_IP, CLIENT_ID)
	client.Subscribe(fmt.Sprintf("sensors/%s/solved", CLIENT_ID), 1, onSolvedHandler)

	var trigger bool
	trigger = false

	for {
		if !trigger {
			trigger = generateIncident()
		} else {
			incidentID := createIncidentID(CLIENT_ID)
			payload, _ := createIncidentPayload(SENSOR_TYPE, incidentID)
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
