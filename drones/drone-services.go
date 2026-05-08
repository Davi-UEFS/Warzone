package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Davi-UEFS/Warzone/shared"
)

func handleAction(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return

		case payload := <-payloadChannel:

			var command shared.DroneMission

			if err := json.Unmarshal(payload, &command); err != nil {
				fmt.Printf("Erro ao desserializar pacote: %v", err)
				continue
			}

			switch command.Type {
			case shared.WATER:
				carryWater(command)

			case shared.OIL:
				drainOil(command)
			}

		}
	}
}

func makeResult(command shared.DroneMission) ([]byte, error) {

	LClock.CompareAndUpdate(command.LamportTime)

	result := shared.DoneInfo{
		RequisitionID: command.RequisitionID,
		DroneID:       command.AssignedDrone,
		LCTime:        LClock.GetTime(),
	}
	return json.Marshal(result)
}

func notifyDone(payload []byte) {
	client.Publish(MISSION_DONE_TOPIC, 1, false, payload)
}
