package main

import (
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// newDroneMQTTClient cria um cliente MQTT já configurado com:
// - clientID
// - lista de brokers
// - callbacks de conexão e perda de conexão
func newDroneMQTTClient(clientID string, brokers []string, onConnect func(mqtt.Client), onLost func(mqtt.Client, error)) (mqtt.Client, error) {
	if clientID == "" {
		return nil, fmt.Errorf("clientID vazio")
	}
	if len(brokers) == 0 {
		return nil, fmt.Errorf("nenhum broker informado")
	}

	// Configurações do client MQTT.
	// Observação: auto-reconnect foi desativado porque o failover será manual.
	opts := mqtt.NewClientOptions()
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(false)
	opts.SetConnectRetry(false)
	opts.SetKeepAlive(10 * time.Second)
	opts.SetPingTimeout(5 * time.Second)

	// Adiciona os brokers informados.
	for _, broker := range brokers {
		broker = shared.NormalizeBrokerAddr(broker)
		if broker != "" {
			opts.AddBroker(broker)
		}
	}

	// Registra os callbacks de evento.
	opts.OnConnect = onConnect
	opts.OnConnectionLost = onLost

	client := mqtt.NewClient(opts)
	return client, nil
}

// connectWithFailover tenta conectar no broker atual.
// Se falhar, avança para o próximo broker da lista até conseguir.
func (app *DroneApp) connectWithFailover() error {
	if len(app.Brokers) == 0 {
		return fmt.Errorf("lista de brokers vazia")
	}

	// Evita concorrência entre várias tentativas simultâneas de reconnect.
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
		// Normaliza o endereço do broker atual.
		broker := shared.NormalizeBrokerAddr(app.Brokers[app.CurrentIdx])
		fmt.Printf("Tentando conectar em %s...\n", broker)

		// Se houver uma conexão anterior ativa, encerra antes de trocar.
		if app.Client != nil && app.Client.IsConnected() {
			app.Client.Disconnect(250)
		}

		// Cria um client novo apontando somente para o broker atual.
		client, err := newDroneMQTTClient(app.ID, []string{broker}, app.onConnect, app.onLost)
		if err != nil {
			fmt.Printf("Erro ao criar client: %v\n", err)
		} else {
			// Conecta efetivamente.
			token := client.Connect()
			token.Wait()

			if token.Error() == nil {
				// Conexão bem-sucedida: salva o client ativo.
				app.Client = client
				fmt.Printf("Conectado em %s\n", broker)
				return nil
			}

			fmt.Printf("Falha em %s: %v\n", broker, token.Error())
		}

		// Próximo broker da lista.
		app.CurrentIdx = (app.CurrentIdx + 1) % len(app.Brokers)
		attempts++

		// Se já tentou todos e voltou ao início, espera um pouco antes de tentar de novo.
		if app.CurrentIdx == startIdx && attempts >= len(app.Brokers) {
			time.Sleep(2 * time.Second)
		}
	}
}

// notifyDone publica no tópico de conclusão da missão.
func (app *DroneApp) notifyDone(payload []byte) {
	// Retry loop: if broker is down or publish fails, wait for reconnection and retry.
	for {
		if app.Client != nil && app.Client.IsConnected() {
			token := app.Client.Publish(MISSION_DONE_TOPIC, 1, false, payload)
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
