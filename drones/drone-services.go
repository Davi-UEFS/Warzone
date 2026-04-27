package main

import (
	"context"
	"encoding/json"

	"github.com/Davi-UEFS/Warzone/shared/types"
)

func handleAction(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return

		case payload := <-payloadChannel:

			var command types.DroneCommand

			if err := json.Unmarshal(payload, &command); err != nil {
				//TODO: POSSO AVISAR NUM PRINT AQUI
				continue
			}

			switch command.Action {
			case "water":
				carryWater(command)

			case "oil":
				drainOil(command)
			}

		}
	}
}

func makeResult(command types.DroneCommand) ([]byte, error) {
	result := types.MissionResult{
		OccurrenceID: command.OccurrenceID,
		Action:       command.Action,
		Status:       "DONE",
	}
	return json.Marshal(result)
}

func notifyDone(payload []byte) {
	client.Publish(MISSION_DONE_TOPIC, 1, false, payload)
}
