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
//
// Params:
//   - addr: o endereço do broker, que pode ser apenas "host:porta" ou já incluir o protocolo (ex: "tcp://host:porta").
//
// Returns:
//   - string: o endereço do broker formatado corretamente para uso com o cliente MQTT. Se o endereço já incluir um protocolo, ele é retornado sem alterações.
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

// MakeClient cria e conecta um cliente MQTT usando as opções fornecidas.
//
// Params:
//   - brokerIP: o endereço do broker MQTT (ex: "tcp://host:1883").
//   - clientID: o ID do cliente MQTT.
//   - onConnect: uma função opcional que é chamada quando a conexão é estabelecida. Se nil, uma função padrão é usada.
//   - autoRec: um booleano que indica se o cliente deve tentar se reconectar automaticamente em caso de perda de conexão.
//
// Returns:
//   - mqtt.Client: o cliente MQTT criado ou nil em caso de erro.
//   - error: um erro caso a conexão falhe, ou nil se a conexão for bem-sucedida.
func MakeClient(brokerIP, clientID string, onConnect func(mqtt.Client), autoRec bool) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerIP)
	opts.SetClientID(clientID)
	opts.SetCleanSession(false)
	opts.AutoReconnect = autoRec

	if onConnect != nil {
		opts.OnConnect = onConnect
	} else {
		opts.OnConnect = func(client mqtt.Client) {
			fmt.Printf("Conectado ao broker %s\n", brokerIP)
		}
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

// WaitForShutdown aguarda um sinal de interrupção (Ctrl+C) para desconectar o cliente MQTT do broker de forma limpa.
//
// Params:
//   - client: o cliente MQTT que deve ser desconectado quando um sinal de interrupção for recebido.
func WaitForShutdown(client mqtt.Client) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\nDesconectando do broker...")
	client.Disconnect(250)
}
