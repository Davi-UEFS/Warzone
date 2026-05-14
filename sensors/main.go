package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// flags are handled in main; removed getEnviromentVariables

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

// Nota: lógica de notificação ao sensor removida — sensores não aguardam mais solução.

//////////////////////////////////////////

func main() {
	// Flags
	broker := flag.String("broker", "tcp://localhost:1883", "Endereço do broker MQTT (ex: tcp://host:1883)")
	sensorID := flag.String("id", "sensor-01", "ID do cliente/sensor")
	sensorType := flag.String("type", "temperature", "Tipo de sensor")
	flag.Parse()

	BROKER_IP, CLIENT_ID, SENSOR_TYPE := *broker, *sensorID, *sensorType
	TOPIC := fmt.Sprintf("sensors/%s/incidents", CLIENT_ID)

	client, _ := shared.MakeClient(BROKER_IP, CLIENT_ID)
	// Nota: não nos inscrevemos em `sensors/<id>/solved` — sensores não aguardam confirmação.

	// Monitor de conexão: tenta recriar o client se necessário
	go func() {
		for {
			if client == nil {
				c, err := shared.MakeClient(BROKER_IP, CLIENT_ID)
				if err == nil {
					client = c
				}
			} else if !client.IsConnected() {
				fmt.Println("Conexão perdida com o broker — tentando reconectar...")
				if token := client.Connect(); token.Wait() && token.Error() != nil {
					fmt.Printf("Reconexão falhou: %v\n", token.Error())
					time.Sleep(2 * time.Second)
					continue
				}
				fmt.Println("Reconectado ao broker")
			}
			time.Sleep(1 * time.Second)
		}
	}()

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
			if token.Error() != nil {
				fmt.Printf("Erro ao publicar alerta: %v\n", token.Error())
			}

			fmt.Println("Alerta enviado ao gerenciador.")

			// Sensores não aguardam confirmação de resolução — continuam gerando alertas.
			trigger = false

		}

		time.Sleep(time.Second)

	}

}
