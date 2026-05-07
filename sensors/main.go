package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func getEnviromentVariables() (string, string, string) {
	return os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID"), os.Getenv("SENSOR_TYPE")
}

func createAlertPayload(SENSOR_TYPE, SENSOR_ID string) ([]byte, error) {

	alert := shared.Alert{
		SensorID:    SENSOR_ID,
		Coordinate:  generateRandomCoordinate(), //TODO: GERAR COORDENADAS MELHOR DEPOIS
		Type:        SENSOR_TYPE,
		LamportTime: LClock.GetTime(),
	}

	return json.Marshal(alert)

}

func generateRandomCoordinate() shared.Coordinate {
	return shared.Coordinate{
		Latitude:  rand.Intn(500),
		Longitude: rand.Intn(500),
	}
}

/////////////////////////////////////////////////////
//////////////////// VARS ///////////////////////////
/////////////////////////////////////////////////////

var LClock = &shared.LamportClock{
	Time: 0,
	Mu:   sync.Mutex{},
}

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
			trigger = generateAlert()
		} else {
			LClock.Tick()

			payload, _ := createAlertPayload(SENSOR_TYPE, CLIENT_ID)

			token := client.Publish(TOPIC, 1, false, payload)
			token.Wait()

			fmt.Println("Alerta enviado ao gerenciador.")

			<-solvedSignal
			trigger = false

			fmt.Println("Alerta resolvido. Começando de novo...")

		}

		time.Sleep(time.Second)

	}

}
