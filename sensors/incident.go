package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var mqttChan = make(chan mqtt.Message)

var mqttMessageHandler mqtt.MessageHandler = func(client mqtt.Client, message mqtt.Message) {
	mqttChan <- message

}

func getEnviromentVariables() (string, string) {
	return os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID")
}

func processMsg(ctx context.Context, input <-chan mqtt.Message) chan mqtt.Message {

	out := make(chan mqtt.Message)

	go func() {
		defer close(out)
		for {
			select {
			case msg, ok := <-input:
				if !ok {
					return
				}
				fmt.Printf("Mensagem recebida: %s do topico: %s\n", msg.Payload(), msg.Topic())
				out <- msg

			case <-ctx.Done():
				return
			}

		}
	}()

	return out

}

var connectHandler2 mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Conectado ao broker")
}

var connectLostHandler2 mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Println("Conexao perdida com o broker")
}

func generateIncident() bool {
	chance := rand.Float64()
	if chance <= 0.10 {
		return true
	}

	return false

}

func makeTopic() string {
	return fmt.Sprintf("setor/%s/sensors/%s", os.Getenv("SECTOR"), os.Getenv("CLIENT_ID"))
}

func main() {

	opts := mqtt.NewClientOptions()
	BROKER_IP, CLIENT_ID := getEnviromentVariables()
	TOPIC := makeTopic()

	opts.AddBroker(BROKER_IP)
	opts.SetClientID(CLIENT_ID)
	opts.SetDefaultPublishHandler(mqttMessageHandler)
	opts.OnConnect = connectHandler2
	opts.OnConnectionLost = connectLostHandler2

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	var trigger bool
	trigger = false

	token := client.Publish(TOPIC, 1, false, fmt.Sprintf("DISPAROU: %v\n", trigger))
	token.Wait()

	for {
		if !trigger {
			trigger = generateIncident()
		} else {
			token := client.Publish(TOPIC, 1, false, fmt.Sprintf("DISPAROU: %v\n", trigger))
			token.Wait()
			trigger = false
			fmt.Println("Evento disparou. Comecando de novo")
		}

		time.Sleep(time.Second)

	}

}
