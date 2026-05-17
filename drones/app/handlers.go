package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// missionHandler coloca o payload no canal interno para processamento posterior.
func (app *DroneApp) missionHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Missão recebida!")
	app.PayloadChannel <- msg.Payload()
}

func (app *DroneApp) regErrorHandler(client mqtt.Client, msg mqtt.Message) {
	var errorMsg shared.RegErrorMessage
	if err := json.Unmarshal(msg.Payload(), &errorMsg); err != nil {
		fmt.Printf("Erro ao desserializar mensagem de erro de registro: %v\n", err)
		return
	}

	fmt.Printf("Erro de registro recebido: %s\n", errorMsg.Error)

	fmt.Println("Aguardando 3 segundos a eleição do Raft terminar...")

	go func() {
		time.Sleep(3 * time.Second)
		fmt.Println("Reenviando pedido de registro...")

		app.register(client)
	}()
}

// handleAction consome missões e executa a ação correspondente.
func (app *DroneApp) handleAction(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload := <-app.PayloadChannel:
			var command shared.DroneMission
			log.Println("Payload recebido para processamento")

			if err := json.Unmarshal(payload, &command); err != nil {
				fmt.Printf("Erro ao desserializar pacote: %v\n", err)
				continue
			}

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
	}
}
