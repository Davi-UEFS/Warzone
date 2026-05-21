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

			app.Info.SetBusy(command.RequisitionID)

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

// makeResult monta o payload de resposta ao terminar a missão.
//
// Params:
//   - command: missão que foi executada, usada para extrair informações e atualizar o relógio de Lamport.
//
// Returns:
//   - []byte: Payload JSON contendo as informações do resultado da missão, pronto para ser enviado ao broker.
//   - error: Erro caso haja falha na serialização do resultado.
func (app *DroneApp) makeResult(command shared.DroneMission) ([]byte, error) {
	app.LClock.CompareAndUpdate(command.LamportTime)

	result := shared.DoneInfo{
		RequisitionID: command.RequisitionID,
		DroneID:       command.AssignedDrone,
		LCTime:        app.LClock.GetTime(),
	}
	return json.Marshal(result)
}

// drainBattery simula o uso de bateria do drone.
//
// Params:
//   - value: quantidade de bateria a ser drenada. Se o nível de bateria cair abaixo de 0, o drone recarrega automaticamente.
func (app *DroneApp) drainBattery(value int) {
	app.Info.BatteryLevel -= value
	if app.Info.BatteryLevel < 0 {
		fmt.Println("Recarregando")
		app.Info.BatteryLevel = 100
	}
}
