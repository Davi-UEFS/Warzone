package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DroneApp representa a aplicação do drone e todo o estado necessário
// para conexão MQTT, clock lógico e processamento das missões.
type DroneApp struct {
	ID             string
	Info           shared.Drone
	Brokers        []string
	CurrentIdx     int
	Client         mqtt.Client
	LClock         *shared.LamportClock
	ReconnectChan  chan bool
	PayloadChannel chan []byte

	// Mutex + flag para impedir reconnect concorrente.
	ReconnectMu  sync.Mutex
	Reconnecting bool
}

// missionHandler é chamado quando uma missão chega no tópico MQTT do drone.
// Ele apenas coloca o payload no canal interno para ser processado depois.
func (app *DroneApp) missionHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("[Drone %s] Missão recebida!\n", app.ID)
	app.PayloadChannel <- msg.Payload()
}

// handleAction fica ouvindo o canal de payloads e executa as ações
// associadas ao tipo de missão recebido.
func (app *DroneApp) handleAction(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case payload := <-app.PayloadChannel:
			var command shared.DroneMission
			log.Println("Payload recebido para processamento") //TODO: DEBUG

			// Converte o JSON recebido para estrutura tipada.
			if err := json.Unmarshal(payload, &command); err != nil {
				fmt.Printf("Erro ao desserializar pacote: %v", err)
				continue
			}

			// Executa a tarefa conforme o tipo de missão.
			switch command.Type {
			case shared.WATER:
				app.CarryWater(command)

			case shared.OIL:
				app.DrainOil(command)
			}
		}
	}
}

// makeResult monta o payload de resposta ao término da missão.
// Ele também atualiza o clock de Lamport com base no tempo recebido.
func (app *DroneApp) makeResult(command shared.DroneMission) ([]byte, error) {
	app.LClock.CompareAndUpdate(command.LamportTime)

	result := shared.DoneInfo{
		RequisitionID: command.RequisitionID,
		DroneID:       command.AssignedDrone,
		LCTime:        app.LClock.GetTime(),
	}
	return json.Marshal(result)
}

// notifyDone publica no tópico de conclusão da missão.
func (app *DroneApp) notifyDone(payload []byte) {
	// Retry loop: if broker is down or publish fails, wait for reconnection and retry.
	for {
		if app.Client != nil && app.Client.IsConnected() {
			token := app.Client.Publish(MISSION_DONE_TOPIC, 1, false, payload)
			token.Wait()
			if token.Error() == nil {
				fmt.Printf("[Drone %s] Resultado da missão publicado no broker %s\n", app.ID, app.Brokers[app.CurrentIdx])
				return
			}
			fmt.Printf("Erro ao publicar done: %v — tentando novamente...\n", token.Error())
		} else {
			fmt.Printf("[Drone %s] Não conectado ao broker — aguardando reconexão para publicar resultado...\n", app.ID)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
