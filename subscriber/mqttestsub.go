package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	BROKER_IP = "tcp://localhost:1883"
	TOPIC     = "setor/A/sensors/+"
	CLIENT_ID = "mqtt_test_subscriber"
)

var mqttChan = make(chan mqtt.Message)

var mqttMessageHandler mqtt.MessageHandler = func(client mqtt.Client, message mqtt.Message) {
	mqttChan <- message

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

func main() {

	opts := mqtt.NewClientOptions()
	opts.AddBroker(BROKER_IP)
	opts.SetClientID(CLIENT_ID)
	opts.SetDefaultPublishHandler(mqttMessageHandler)
	opts.OnConnect = connectHandler2
	opts.OnConnectionLost = connectLostHandler2

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		finalChan := processMsg(ctx, mqttChan)
		for range finalChan {

		}
	}()

	token := client.Subscribe(TOPIC, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic: %s\n", TOPIC)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	cancel()

	fmt.Println("Unsubscribing and disconnecting...")
	client.Unsubscribe(TOPIC)
	client.Disconnect(250)

	wg.Wait()

}
