package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Davi-UEFS/Warzone/shared/types"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var MISSION_TOPIC = fmt.Sprintf("drones/%s/commands", os.Getenv("CLIENT_ID"))
var payloadChannel = make(chan []byte)

var commandHandler = func(client mqtt.Client, msg mqtt.Message) {
	payloadChannel <- msg.Payload()

}

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
				carryWater()

			case "oil":
				drainOil()
			}

		}
	}
}
