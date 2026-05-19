package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DroneApp concentra o estado e o comportamento do drone.
type DroneApp struct {
	ID             string
	Info           shared.Drone
	Brokers        []string
	CurrentIdx     int
	Client         mqtt.Client
	LClock         *shared.LamportClock
	ReconnectChan  chan bool
	PayloadChannel chan []byte

	missionTopic     string
	missionDoneTopic string
	regErrorTopic    string
	DebugMode        bool

	// Mutex + flag para impedir reconnect concorrente.
	ReconnectMu  sync.Mutex
	Reconnecting bool
}

// NewDroneApp monta o estado inicial do drone.
func NewDroneApp(id string, brokers []string, debugMode bool) *DroneApp {
	return &DroneApp{
		ID:      id,
		Brokers: brokers,
		LClock: &shared.LamportClock{
			Time: 0,
		},
		Info: shared.Drone{
			ID:             id,
			BatteryLevel:   100,
			Status:         shared.DRONE_IDLE,
			CurrentMission: shared.NONE,
		},
		ReconnectChan:    make(chan bool),
		PayloadChannel:   make(chan []byte, 4096),
		missionTopic:     fmt.Sprintf("drones/%s/mission", id),
		missionDoneTopic: fmt.Sprintf("drones/%s/done", id),
		regErrorTopic:    fmt.Sprintf("drones/%s/reg_error", id),
		DebugMode:        debugMode,
	}
}

// Run inicia o ciclo de vida do drone.
func (app *DroneApp) Run() {
	if err := app.connectWithFailover(); err != nil {
		fmt.Printf("Falha ao conectar: %v\n", err)
		return
	}

	ctx := context.Background()
	go app.handleAction(ctx)
	go app.startHeartbeat()

	select {}
}

func (app *DroneApp) startHeartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if app.Client != nil && app.Client.IsConnected() {
			app.sendHeartbeat()
		}
	}
}

// onConnect é chamado pelo client MQTT toda vez que a conexão é bem-sucedida.
func (app *DroneApp) onConnect(client mqtt.Client) {
	fmt.Printf("Conectado ao broker %s.\n", app.Brokers[app.CurrentIdx])

	app.Info.CurrentBroker = app.Brokers[app.CurrentIdx]
	client.Subscribe(app.missionTopic, 1, app.missionHandler)
	client.Subscribe(app.regErrorTopic, 1, app.regErrorHandler)
	app.register(client)
}

// onLost é chamado quando o broker cai ou a conexão é perdida.
func (app *DroneApp) onLost(client mqtt.Client, err error) {
	fmt.Printf("Conexão perdida: %v. Iniciando failover...\n", err)
	go app.connectWithFailover()
}

// register publica no tópico de registro a informação atual do drone.
func (app *DroneApp) register(client mqtt.Client) {
	app.LClock.Tick()

	// TODO: DEBUG_MODE_LAMPORT_TICK
	if app.DebugMode {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m TICK (+1): Relógio = %d | Ação: Pedido de Registro de Drone\n", app.LClock.GetTime())
	}

	// ... resto do código (payload, etc)

	payload, err := json.Marshal(app.Info)
	if err != nil {
		fmt.Printf("Erro ao preparar registro: %v\n", err)
		return
	}

	token := client.Publish("drones/register", 1, true, payload)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Falha ao enviar registro: %v\n", token.Error())
	} else {
		fmt.Printf("Registro enviado: Drone %s no Broker %s\n", app.ID, app.Info.CurrentBroker)
	}
}

func (app *DroneApp) sendHeartbeat() {
	data := shared.DroneHeartbeat{
		ID:           app.ID,
		BatteryLevel: app.Info.BatteryLevel,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Erro ao preparar heartbeat: %v\n", err)
		return
	}

	topic := fmt.Sprintf("drones/%s/heartbeat", app.ID)
	token := app.Client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Erro ao enviar heartbeat: %v\n", token.Error())
	}
}
