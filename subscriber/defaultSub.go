package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\nDesconectando do broker...")
	client.Disconnect(250)
}
