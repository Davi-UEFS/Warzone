package main

import (
	"fmt"
	"os"

	"github.com/Davi-UEFS/Warzone/shared/functions"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func handleMessage(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Tópico: %s | Payload: %s\n", msg.Topic(), msg.Payload())
	// sua lógica aqui
}

func main() {
	client := functions.MakeClient(
		os.Getenv("BROKER_IP"),
		os.Getenv("CLIENT_ID"),
	)

	client.Subscribe("sensors/+", 1, handleMessage)
	functions.WaitForShutdown(client)
}
