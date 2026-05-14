package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// onConnect é chamado pelo client MQTT toda vez que a conexão
// com um broker é estabelecida com sucesso.
func (app *DroneApp) onConnect(client mqtt.Client) {
	fmt.Printf("[Drone %s] Conectado ao broker %s.\n", app.ID, app.Brokers[app.CurrentIdx])

	// Atualiza no estado do drone qual broker está sendo usado agora.
	// Isso é importante para o Raft e para debug.
	app.Info.CurrentBroker = app.Brokers[app.CurrentIdx]

	// Os tópicos dependem do ID do drone, então são montados aqui.
	MISSION_TOPIC = fmt.Sprintf("drones/%s/mission", app.ID)
	MISSION_DONE_TOPIC = fmt.Sprintf("drones/%s/done", app.ID)

	// O drone fica ouvindo as missões publicadas para ele.
	client.Subscribe(MISSION_TOPIC, 1, app.missionHandler)

	// Assim que conecta, ele registra sua presença no cluster.
	app.register()
}

// onLost é chamado quando o broker cai ou a conexão é perdida.
// Aqui a estratégia é iniciar o failover para o próximo broker.
func (app *DroneApp) onLost(client mqtt.Client, err error) {
	fmt.Printf("[Drone %s] Conexão perdida: %v. Iniciando failover...\n", app.ID, err)
	go app.connectWithFailover()
}

// Run inicia o ciclo de vida do drone:
// 1. tenta conectar com failover
// 2. inicia o consumidor de missões
// 3. mantém o processo vivo até receber SIGINT/SIGTERM
func (app *DroneApp) Run() {
	// Conecta inicialmente (ou tenta todos os brokers até conseguir).
	log.Println("teste")
	if err := app.connectWithFailover(); err != nil {
		fmt.Printf("Falha crítica ao conectar: %v\n", err)
		return
	}

	// Contexto do worker de ações.
	ctx := context.Background()
	go app.handleAction(ctx)

	// --- O HEARTBEAT (PULSO DE VIDA) ---
	// A cada 5 segundos, o drone avisa o cluster que está vivo e qual a sua bateria
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Só envia se a conexão MQTT estiver ativa no momento
			if app.Client != nil && app.Client.IsConnected() {
				app.sendHeartbeat()
			}
		}
	}()

	select {}
}

// register publica no tópico de registro a informação atual do drone.
// Isso permite que o cluster saiba que o drone está ativo e em qual broker ele está.
func (app *DroneApp) register() {
	// Tick no relógio de Lamport para marcar esse evento.
	app.LClock.Tick()

	// Serializa a estrutura do drone para JSON.
	payload, err := json.Marshal(app.Info)
	if err != nil {
		fmt.Printf("Erro ao preparar registro: %v\n", err)
		return
	}

	// Publica de forma persistente no cluster.
	// retained=true ajuda outros componentes a saberem o último estado.
	token := app.Client.Publish("drones/register", 1, true, payload)

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

func main() {
	// Flags de configuração do drone.
	idFlag := flag.String("id", "drone-01", "ID do drone")
	sectorBaseFlag := flag.String("sector", "Setor-A", "Setor/broker base do drone")
	brokersFlag := flag.String("brokers", "tcp://localhost:1883,tcp://localhost:1884", "Lista de brokers separados por vírgula")
	flag.Parse()

	droneID := *idFlag
	setorBase := *sectorBaseFlag
	brokers := strings.Split(*brokersFlag, ",")

	// Inicializa o estado da aplicação.
	app := &DroneApp{
		ID:      droneID,
		Brokers: brokers,
		LClock: &shared.LamportClock{
			Time: 0,
			Mu:   sync.Mutex{},
		},
		Info: shared.Drone{
			ID:            droneID,
			BatteryLevel:  100,
			Status:        shared.DRONE_IDLE,
			CurrentSector: setorBase,
		},
		ReconnectChan:  make(chan bool),
		PayloadChannel: make(chan []byte, 4096),
	}

	// Inicia o ciclo principal do drone em goroutine.
	go app.Run()

	// Espera sinal de interrupção para encerrar de forma graciosa.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("\nEncerrando atividades...")
}
