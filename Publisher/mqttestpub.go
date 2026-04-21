package main

import (
	"fmt"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	BROKER_IP = "tcp://localhost:1883"
	TOPIC     = "test/topic"
	CLIENT_ID = "mqtt_test_publisher"
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Conectado ao broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Println("Conexao perdida com o broker")
}

func main() {

	opts := mqtt.NewClientOptions()
	opts.AddBroker(BROKER_IP)
	opts.SetClientID(CLIENT_ID)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	for {
		message := generateRandomMessages()

		token := client.Publish(TOPIC, 0, false, message)
		token.Wait()
		fmt.Println("Mensagem publicada. Aguardando 1s...")
		time.Sleep(time.Second)
	}

}

func generateRandomMessages() string {

	messages := []string{
		"Hello, World!",
		"Greetings from Go!",
		"MQTT is awesome!",
		"Random message incoming!",
		"Go is fun!",
	}
	return messages[rand.Intn(len(messages))]
}
