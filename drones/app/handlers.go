package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// missionHandler coloca o payload no canal interno para processamento posterior.
func (app *DroneApp) missionHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Missão recebida!")
	app.PayloadChannel <- msg.Payload()
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
