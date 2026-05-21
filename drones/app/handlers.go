package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Davi-UEFS/Warzone/shared"
)

// handleAction consome missões e executa a ação correspondente.
func (app *DroneApp) handleAction(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload := <-app.PayloadChannel:
			var command shared.DroneMission

			if err := json.Unmarshal(payload, &command); err != nil {
				fmt.Printf("Erro ao desserializar pacote: %v\n", err)
				continue
			}

			app.Info.CurrentMission = command.RequisitionID
			app.Info.Status = shared.DRONE_BUSY

			app.PrintDashboard(fmt.Sprintf("ALERTA: Nova missão recebida! (Tipo: %s)", command.Type))
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
			app.PrintDashboard("Missão finalizada! Drone liberado e retornando ao estado IDLE.")
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
