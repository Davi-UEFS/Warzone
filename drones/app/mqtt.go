package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// newDroneMQTTClient cria um cliente MQTT já configurado.
//
// Params:
// clientID: O nome do cliente MQTT
// broker: O broker MQTT
// onConnect: A função/handler chamada ao conectar.
// onLost: A função/handler chamada ao perder a conexão.
//
// Returns:
// O cliente MQTT criado e erro.
func newDroneMQTTClient(clientID string, broker string, onConnect func(mqtt.Client), onLost func(mqtt.Client, error)) (mqtt.Client, error) {
	if clientID == "" {
		return nil, fmt.Errorf("clientID vazio")
	}
	if broker == "" {
		return nil, fmt.Errorf("broker vazio")
	}

	opts := mqtt.NewClientOptions()
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(false)

	broker = shared.NormalizeBrokerAddr(broker)
	if broker != "" {
		opts.AddBroker(broker)
	}

	opts.OnConnect = onConnect
	opts.OnConnectionLost = onLost

	return mqtt.NewClient(opts), nil
}

// connectWithFailover tenta conectar em um dos brokers informados.
// Utiliza round-robin para ciclar tentativas.
//
// Returns:
// Erro se a lista de brokers estiver vazia
func (app *DroneApp) connectWithFailover() error {
	if len(app.Brokers) == 0 {
		return fmt.Errorf("lista de brokers vazia")
	}

	app.ReconnectMu.Lock()
	if app.Reconnecting {
		app.ReconnectMu.Unlock()
		return nil
	}
	app.Reconnecting = true
	app.ReconnectMu.Unlock()

	defer func() {
		app.ReconnectMu.Lock()
		app.Reconnecting = false
		app.ReconnectMu.Unlock()
	}()

	startIdx := app.CurrentIdx
	attempts := 0

	for {
		broker := shared.NormalizeBrokerAddr(app.Brokers[app.CurrentIdx])
		fmt.Printf("Tentando conectar em %s...\n", broker)

		if app.Client != nil && app.Client.IsConnected() {
			app.Client.Disconnect(250)
		}

		client, err := newDroneMQTTClient(app.ID, broker, app.onConnect, app.onLost)
		if err != nil {
			fmt.Printf("Erro ao criar client: %v\n", err)
		} else {
			token := client.Connect()
			token.Wait()

			if token.Error() == nil {
				app.Client = client
				fmt.Printf("Conectado em %s\n", broker)
				return nil
			}

			fmt.Printf("Falha em %s: %v\n", broker, token.Error())
		}

		app.CurrentIdx = (app.CurrentIdx + 1) % len(app.Brokers)
		attempts++

		if app.CurrentIdx == startIdx && attempts >= len(app.Brokers) {
			time.Sleep(2 * time.Second)
		}
	}
}

// missionHandler coloca o payload no canal interno para processamento posterior.
func (app *DroneApp) missionHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Missão recebida!")
	app.PayloadChannel <- msg.Payload()
}

// regErrorHandler escuta a resposta do setor acerca do registro enviado.
// Tenta novamente após 3 segundos.
func (app *DroneApp) regErrorHandler(client mqtt.Client, msg mqtt.Message) {
	var errorMsg shared.RegErrorMessage
	if err := json.Unmarshal(msg.Payload(), &errorMsg); err != nil {
		fmt.Printf("Erro ao desserializar mensagem de erro de registro: %v\n", err)
		return
	}

	fmt.Printf("Erro de registro recebido: %s\n", errorMsg.Error)

	fmt.Println("Aguardando 3 segundos a eleição do Raft terminar...")

	go func() {
		time.Sleep(3 * time.Second)
		fmt.Println("Reenviando pedido de registro...")

		app.register(client)
	}()
}

// notifyDone publica no tópico de conclusão da missão.
//
// Params:
// payload: O dado a ser enviado por MQTT.
func (app *DroneApp) notifyDone(payload []byte) {
	for {
		if app.Client != nil && app.Client.IsConnected() {
			token := app.Client.Publish(app.missionDoneTopic, 1, false, payload)
			token.Wait()
			if token.Error() == nil {
				fmt.Printf("Resultado da missão publicado no broker %s\n", app.Brokers[app.CurrentIdx])
				return
			}
			fmt.Printf("Erro ao publicar done: %v — tentando novamente...\n", token.Error())
		} else {
			fmt.Println("Não conectado ao broker — aguardando reconexão para publicar resultado...")
		}
		time.Sleep(500 * time.Millisecond)
	}
}
