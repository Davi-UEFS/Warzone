package functions

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func MakeClient(brokerIP, clientID string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerIP)
	opts.SetClientID(clientID)
	opts.OnConnect = func(client mqtt.Client) {
		fmt.Println("Conectado ao broker")
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		fmt.Println("Conexao perdida com o broker")
	}
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return client
}

func WaitForShutdown(client mqtt.Client) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\nDesconectando do broker...")
	client.Disconnect(250)
}
