package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DroneApp representa a aplicação do drone e todo o estado necessário
// para conexão MQTT, clock lógico e processamento das missões.
type DroneApp struct {
	ID             string
	Info           shared.Drone
	Brokers        []string
	CurrentIdx     int
	Client         mqtt.Client
	LClock         *shared.LamportClock
	ReconnectChan  chan bool
	PayloadChannel chan []byte

	// Mutex + flag para impedir reconnect concorrente.
	ReconnectMu  sync.Mutex
	Reconnecting bool
}

// missionHandler é chamado quando uma missão chega no tópico MQTT do drone.
// Ele apenas coloca o payload no canal interno para ser processado depois.
func (app *DroneApp) missionHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Missão recebida!")
	app.PayloadChannel <- msg.Payload()
}

// handleAction fica ouvindo o canal de payloads e executa as ações
// associadas ao tipo de missão recebido.
func (app *DroneApp) handleAction(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case payload := <-app.PayloadChannel:
			var command shared.DroneMission
			log.Println("Payload recebido para processamento") //TODO: DEBUG

			// Converte o JSON recebido para estrutura tipada.
			if err := json.Unmarshal(payload, &command); err != nil {
				fmt.Printf("Erro ao desserializar pacote: %v", err)
				continue
			}

			// Executa a tarefa conforme o tipo de missão.
			switch command.Type {
			case shared.FIRE:
				app.CarryWater(command)

			case shared.OIL:
				app.DrainOil(command)

			case shared.BOTTLENECK:
				app.OptimizeRoute(command)

			case shared.WRECKAGE:
				app.RetrieveGoods(command)

			case shared.INSPECTION:
				app.PerformInspection(command)

			case shared.UNKNOWN_OBJECT:
				app.IdentifyObject(command)
			default:
				fmt.Printf("Tipo de missão desconhecido: %s\n", command.Type)
			}

		}
	}
}

// makeResult monta o payload de resposta ao término da missão.
// Ele também atualiza o clock de Lamport com base no tempo recebido.
func (app *DroneApp) makeResult(command shared.DroneMission) ([]byte, error) {
	app.LClock.CompareAndUpdate(command.LamportTime)

	result := shared.DoneInfo{
		RequisitionID: command.RequisitionID,
		DroneID:       command.AssignedDrone,
		LCTime:        app.LClock.GetTime(),
	}
	return json.Marshal(result)
}

func (app *DroneApp) drainBattery(value int) {
	app.Info.BatteryLevel -= value
	if app.Info.BatteryLevel < 0 {
		fmt.Println("Recarregando")
		app.Info.BatteryLevel = 100
		//TODO: POSSO IMPLEMENTAR UM TEMPO DE RECARGA AQUI SE QUISER DEIXAR MAIS REALISTA
	}

}
