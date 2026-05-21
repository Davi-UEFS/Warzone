package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DroneApp comporta todos os metadados do Drone.
//
// Info é a struct do Drone que o setor enxerga.
//
// Todo o resto são metadados para controlar a aplicação.
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
	heartbeatTopic   string
	regErrorTopic    string
	DebugMode        bool

	// Mutex + flag para impedir reconnect concorrente.
	ReconnectMu  sync.Mutex
	Reconnecting bool
}

// NewDroneApp monta o estado inicial do drone.
//
// Params:
//   - id: ID único do drone (ex: "drone1").
//   - brokers: Lista de brokers para failover (ex: ["broker1:1883", "broker2:1883"]).
//   - debugMode: Flag para ativar modo de depuração.
//
// Returns:
//   - Ponteiro para a struct DroneApp inicializada.
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
		heartbeatTopic:   fmt.Sprintf("drones/%s/heartbeat", id),
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

// startHeartbeat é a rotina que envia os pulsos para o setor.
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
//
// Se conecta aos tópicos MQTT e atualiza o broker atual do cliente.
// Também envia o pedido de registro ao setor.
func (app *DroneApp) onConnect(client mqtt.Client) {
	fmt.Printf("Conectado ao broker %s.\n", app.Brokers[app.CurrentIdx])

	app.Info.CurrentBroker = app.Brokers[app.CurrentIdx]
	client.Subscribe(app.missionTopic, 1, app.missionHandler)
	client.Subscribe(app.regErrorTopic, 1, app.regErrorHandler)
	app.register(client)
	app.PrintDashboard("Conectado ao Broker com sucesso. Aguardando patrulhamento...")
}

// onLost é chamado quando o broker cai ou a conexão é perdida.
//
// Inicia o processo de failover para tentar se conectar ao próximo broker da lista.
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

// sendHeartbeat envia no tópico o ID e Bateria atuais do drone.
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

	token := app.Client.Publish(app.heartbeatTopic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Erro ao enviar heartbeat: %v\n", token.Error())
	}
}

// Helper para padronizar o tamanho das strings no painel para que a borda fique reta
func pad(s string, l int) string {
	if len(s) >= l {
		return s[:l]
	}
	return s + strings.Repeat(" ", l-len(s))
}

// PrintDashboard desenha a interface do drone no terminal
func (app *DroneApp) PrintDashboard(action string) {
	mission := app.Info.CurrentMission
	if mission == shared.NONE {
		mission = "Nenhuma (Aguardando)"
	}

	statusColor := "\033[1;32m" // Verde
	if app.Info.Status == shared.DRONE_BUSY {
		statusColor = "\033[1;31m" // Vermelho
	}

	brokerAddr := "Desconectado"
	if app.CurrentIdx >= 0 && app.CurrentIdx < len(app.Brokers) {
		brokerAddr = app.Brokers[app.CurrentIdx]
	}

	// Desenha o bloco visual
	fmt.Println("\n\033[1;34m╔══════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Printf("\033[1;34m║\033[0m   \033[1;37mDRONE ID    :\033[0m %s \033[1;34m    ║\033[0m\n", pad(app.ID, 40))
	fmt.Println("\033[1;34m╠══════════════════════════════════════════════════════════════╣\033[0m")
	fmt.Printf("\033[1;34m║\033[0m   \033[1;36mBroker Atual:\033[0m %s \033[1;34m    ║\033[0m\n", pad(brokerAddr, 40))
	fmt.Printf("\033[1;34m║\033[0m   \033[1;32mBateria     :\033[0m %s \033[1;34m    ║\033[0m\n", pad(fmt.Sprintf("%d%%", app.Info.BatteryLevel), 40))
	fmt.Printf("\033[1;34m║\033[0m   \033[1;33mStatus      :\033[0m %s%s\033[0m \033[1;34m    ║\033[0m\n", statusColor, pad(string(app.Info.Status), 40))
	fmt.Printf("\033[1;34m║\033[0m   \033[1;35mMissão Atual:\033[0m %s \033[1;34m    ║\033[0m\n", pad(mission, 40))
	fmt.Println("\033[1;34m╚══════════════════════════════════════════════════════════════╝\033[0m")

	// Imprime a ação atual que disparou a atualização do painel
	if action != "" {
		fmt.Printf("\033[1;37m>> %s\033[0m\n\n", action)
	}
}
