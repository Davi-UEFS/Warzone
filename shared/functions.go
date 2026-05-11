package shared

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// NormalizeBrokerAddr adiciona a URL TCP ao endereço:porta dado.
func NormalizeBrokerAddr(addr string) string {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		return trimmed
	}
	return "tcp://" + trimmed
}

func MakeClient(brokerIP, clientID string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerIP)
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.OnConnect = func(client mqtt.Client) {
		fmt.Println("Conectado ao broker")
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		fmt.Println("Conexao perdida com o broker")
	}
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("Erro MQTT: %v ", token.Error())
	}
	return client, nil
}

func WaitForShutdown(client mqtt.Client) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\nDesconectando do broker...")
	client.Disconnect(250)
}

func ExtractSensorID(ocurrenceID string) string {
	parts := strings.Split(ocurrenceID, "-")
	return parts[1]
}
