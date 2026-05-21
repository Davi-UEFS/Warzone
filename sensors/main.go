package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// createAlertPayload cria o payload do alerta a ser enviado ao setor manager.
// Ele inclui o ID do sensor, a coordenada gerada aleatoriamente, o tipo de alerta e o tempo de Lamport atual.
//
// Params:
//   - SENSOR_TYPE: o tipo do sensor que gerou o alerta.
//   - SENSOR_ID: o ID do sensor que gerou o alerta.
//
// Returns:
//   - []byte: o payload do alerta em formato JSON pronto.
//   - error: um erro caso a serialização do alerta falhe.
func createAlertPayload(SENSOR_TYPE int, SENSOR_ID string) ([]byte, error) {

	alert := shared.Alert{
		SensorID:    SENSOR_ID,
		Coordinate:  generateRandomCoordinate(), //TODO: GERAR COORDENADAS MELHOR DEPOIS
		Type:        getSensorTypeString(SENSOR_TYPE),
		LamportTime: LClock.GetTime(),
	}

	return json.Marshal(alert)

}

// getSensorTypeString converte o número do tipo de sensor em uma string correspondente ao tipo de alerta.
//
// Params:
//   - sensorType: o número do tipo de sensor.
//
// Returns:
//   - string: a string correspondente ao tipo de alerta. Retorna "INSPECTION" como padrão para números inválidos.
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

// getSensorGenerator chama a função de geração de alerta correspondente ao tipo de sensor fornecido.
//
// Params:
//   - sensorType: o número do tipo de sensor.
//
// Returns:
//   - bool: true se o alerta for gerado, false caso contrário. Retorna false para números de sensor inválidos.
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

///////////////////////////////////////////////
////////////// LAMPORT CLOCK  /////////////////
///////////////////////////////////////////////

var LClock = &shared.LamportClock{
	Time: 0,
	Mu:   sync.Mutex{},
}

func main() {
	// Flags
	broker := flag.String("broker", "tcp://localhost:1883", "Endereço do broker MQTT (ex: tcp://host:1883)")
	sensorID := flag.String("id", "sensor-01", "ID do cliente/sensor")
	sensorType := flag.Int("type", 0, "Tipo de sensor")
	flag.Parse()

	BROKER_IP, CLIENT_ID, SENSOR_TYPE := *broker, *sensorID, *sensorType
	TOPIC := fmt.Sprintf("sensors/%s/incidents", CLIENT_ID) // Tópico para publicar os alertas, usando o ID do sensor para criar um tópico único.

	client, _ := shared.MakeClient(BROKER_IP, CLIENT_ID, nil, true) // Cria o cliente MQTT. Não usa um onConnect personalizado e possui AutoReconnect do Paho.

	trigger := false

	// Fica em loop gerando alertas em intervalos de 1 segundo.

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
