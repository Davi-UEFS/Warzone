package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// flags are handled in main; removed getEnviromentVariables

func createAlertPayload(SENSOR_TYPE int, SENSOR_ID string) ([]byte, error) {

	alert := shared.Alert{
		SensorID:    SENSOR_ID,
		Coordinate:  generateRandomCoordinate(), //TODO: GERAR COORDENADAS MELHOR DEPOIS
		Type:        getSensorTypeString(SENSOR_TYPE),
		LamportTime: LClock.GetTime(),
	}

	return json.Marshal(alert)

}

func getSensorTypeString(sensorType int) string {
	switch sensorType {
	case 1:
		return shared.FIRE
	case 2:
		return shared.OIL
	case 3:
		return shared.WRECKAGE
	case 4:
		return shared.INSPECTION
	case 5:
		return shared.UNKNOWN_OBJECT
	case 6:
		return shared.BOTTLENECK
	default:
		// Retorna um valor padrão ou erro caso o número seja inválido
		return shared.INSPECTION
	}
}

func getSensorGenerator(sensorType int) bool {
	switch sensorType {
	case 1:
		return generateFireAlert()
	case 2:
		return generateOilAlert()
	case 3:
		return generateWreckageAlert()
	case 4:
		return generateInspectionAlert()
	case 5:
		return generateUnknownObjectAlert()
	case 6:
		return generateBottleneckAlert()
	default:
		// Retorna um valor padrão ou erro caso o número seja inválido
		return false
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
	sensorType := flag.Int("type", 0, "Tipo de sensor")
	flag.Parse()

	BROKER_IP, CLIENT_ID, SENSOR_TYPE := *broker, *sensorID, *sensorType
	TOPIC := fmt.Sprintf("sensors/%s/incidents", CLIENT_ID)

	client, _ := shared.MakeClient(BROKER_IP, CLIENT_ID)

	trigger := false

	for {
		if !trigger {
			trigger = getSensorGenerator(SENSOR_TYPE)
		} else {
			LClock.Tick()

			payload, _ := createAlertPayload(SENSOR_TYPE, CLIENT_ID)

			token := client.Publish(TOPIC, 1, false, payload)
			token.Wait()
			if token.Error() != nil {
				fmt.Printf("Erro ao publicar alerta: %v\n", token.Error())
			}

			fmt.Println("Alerta enviado ao gerenciador.")

			trigger = false

		}

		time.Sleep(time.Second)

	}

}
