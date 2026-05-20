package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var typesMap = map[int]string{
	1: shared.FIRE, 2: shared.OIL, 3: shared.WRECKAGE, 4: shared.INSPECTION, 5: shared.UNKNOWN_OBJECT, 6: shared.BOTTLENECK,
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\033[1;35m====================================================\033[0m")
	fmt.Println("\033[1;35m       WARZONE - SIMULADOR INTERATIVO DE CAOS       \033[0m")
	fmt.Println("\033[1;35m====================================================\033[0m")

	brokerInput := ""
	for {
		fmt.Print("\033[1;33mDigite o endereço completo do Broker (ex: tcp://192.168.1.10:1883): \033[0m")
		input, _ := reader.ReadString('\n')
		brokerInput = strings.TrimSpace(input)
		if brokerInput != "" {
			break
		}
		fmt.Println("\033[1;31mO endereço do Broker não pode ser vazio!\033[0m")
	}

	opts := mqtt.NewClientOptions().AddBroker(brokerInput).SetClientID("warzone-chaos-simulator")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("\033[1;31mErro ao conectar no Broker %s: %v\033[0m\n", brokerInput, token.Error())
		return
	}
	fmt.Printf("\033[1;32mConectado com sucesso ao Broker em %s!\033[0m\n\n", brokerInput)

	for {
		showMenu()
		fmt.Print("\033[1;33mEscolha uma opção: \033[0m")
		optionStr, _ := reader.ReadString('\n')
		option, _ := strconv.Atoi(strings.TrimSpace(optionStr))

		switch option {
		case 1:
			enviarAlertaManual(client, reader, "sensor-manual-01")
		case 2:
			enviarEmLote(client, reader)
		case 3:
			enviarSensorLento(client)
		case 4:
			simularCondicaoDeCorrida(brokerInput)
		case 5:
			testeEstresseSensores(client, reader)
		case 6:
			testeEstresseDronesAutonomos(client, reader, brokerInput)
		case 7:
			fmt.Println("Saindo do simulador. Até logo!")
			return
		default:
			fmt.Println("Opção inválida. Tente novamente.")
		}
		fmt.Println("\nPressione ENTER para continuar...")
		reader.ReadString('\n')
	}
}

func showMenu() {
	fmt.Println("\033[1;34m--- MENU DE SIMULAÇÕES DIRETAS ---\033[0m")
	fmt.Println("1- Enviar alerta único (Manual)")
	fmt.Println("2- Enviar alertas em lote (Sequencial)")
	fmt.Println("3- Enviar sensor lento (Gatilho de Latência 10s)")
	fmt.Println("4- Forçar Condição de Corrida (Takeover de Sessão Concorrente)")
	fmt.Println("5- Teste de estresse (Inundação de Alertas de Sensores)")
	fmt.Println("6- Injetar enxame de Drones Autônomos (Processam e Liberam Missões)")
	fmt.Println("7- Sair")
}

// 1. ENVIAR ALERTA MANUAL
func enviarAlertaManual(client mqtt.Client, reader *bufio.Reader, id string) {
	fmt.Println("\nTipos: 1-Fogo, 2-Óleo, 3-Mantimentos, 4-Inspeção, 5-Objeto Suspeito, 6-Tráfego")
	fmt.Print("Escolha o tipo (1-6): ")
	tStr, _ := reader.ReadString('\n')
	tIdx, _ := strconv.Atoi(strings.TrimSpace(tStr))

	typeName, exists := typesMap[tIdx]
	if !exists {
		typeName = shared.INSPECTION
	}

	alert := shared.Alert{
		SensorID:    id,
		Coordinate:  shared.Coordinate{Latitude: rand.Intn(500), Longitude: rand.Intn(500)},
		Type:        typeName,
		LamportTime: 1,
	}

	payload, _ := json.Marshal(alert)
	topic := fmt.Sprintf("sensors/%s/incidents", id)
	client.Publish(topic, 1, false, payload).Wait()
	fmt.Printf("\033[1;32m[SUCESSO]\033[0m Alerta enviado no tópico '%s' (Tipo: %s)\n", topic, typeName)
}

// 2. ENVIAR EM LOTE
func enviarEmLote(client mqtt.Client, reader *bufio.Reader) {
	fmt.Print("Quantos alertas deseja enviar em lote? ")
	qtdStr, _ := reader.ReadString('\n')
	qtd, _ := strconv.Atoi(strings.TrimSpace(qtdStr))

	fmt.Println("Enviando lote...")
	for i := 1; i <= qtd; i++ {
		alert := shared.Alert{
			SensorID:    fmt.Sprintf("sensor-lote-%d", i),
			Coordinate:  shared.Coordinate{Latitude: rand.Intn(500), Longitude: rand.Intn(500)},
			Type:        typesMap[rand.Intn(6)+1],
			LamportTime: i * 2,
		}
		payload, _ := json.Marshal(alert)
		topic := fmt.Sprintf("sensors/%s/incidents", alert.SensorID)
		client.Publish(topic, 1, false, payload).Wait()
	}
	fmt.Printf("\033[1;32m[SUCESSO]\033[0m %d alertas enviados sequencialmente.\n", qtd)
}

// 3. ENVIAR SENSOR LENTO (10s)
func enviarSensorLento(client mqtt.Client) {
	alert := shared.Alert{
		SensorID:    "sensor-lento",
		Coordinate:  shared.Coordinate{Latitude: 111, Longitude: 222},
		Type:        shared.FIRE,
		LamportTime: 10,
	}
	payload, _ := json.Marshal(alert)
	client.Publish("sensors/sensor-lento/incidents", 1, false, payload).Wait()
	fmt.Println("\033[1;33m[GATILHO]\033[0m Alerta do 'sensor-lento' publicado! O Manager configurado com -debug vai retê-lo por 10s.")
}

// 4. CONDIÇÃO DE CORRIDA
func simularCondicaoDeCorrida(brokerTarget string) {
	fmt.Printf("\n\033[1;33mSimulando ataque de condição de corrida em %s...\033[0m\n", brokerTarget)
	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			opts := mqtt.NewClientOptions().AddBroker(brokerTarget).SetClientID("drone-01")
			c := mqtt.NewClient(opts)
			if token := c.Connect(); token.Wait() && token.Error() == nil {
				fmt.Printf("[Thread-%d] Conectou e roubou a sessão do ID 'drone-01'!\n", id)
				c.Publish("drones/drone-01/heartbeat", 1, false, []byte(`{"id":"drone-01","battery_level":99}`)).Wait()
				time.Sleep(100 * time.Millisecond)
				c.Disconnect(250)
			}
		}(i)
	}
	wg.Wait()
}

// 5. TESTE DE ESTRESSE (SENSORS)
func testeEstresseSensores(client mqtt.Client, reader *bufio.Reader) {
	fmt.Print("Quantos alertas simultâneos no ataque de estresse? ")
	qtdStr, _ := reader.ReadString('\n')
	qtd, _ := strconv.Atoi(strings.TrimSpace(qtdStr))

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < qtd; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			alert := shared.Alert{
				SensorID:    fmt.Sprintf("stress-sensor-%d", idx),
				Coordinate:  shared.Coordinate{Latitude: rand.Intn(500), Longitude: rand.Intn(500)},
				Type:        shared.OIL,
				LamportTime: idx,
			}
			payload, _ := json.Marshal(alert)
			topic := fmt.Sprintf("sensors/%s/incidents", alert.SensorID)
			client.Publish(topic, 1, false, payload).Wait()
		}(i)
	}
	wg.Wait()
	fmt.Printf("\033[1;32m[TESTE CONCLUÍDO]\033[0m %d mensagens processadas em %v\n", qtd, time.Since(start))
}

// 6. INJETAR DRONES FUNCIONAIS AUTÔNOMOS (Cria Drones vivos com ciclo de vida completo)
func testeEstresseDronesAutonomos(client mqtt.Client, reader *bufio.Reader, brokerTarget string) {
	fmt.Print("Quantos drones deseja injetar? ")
	qtdStr, _ := reader.ReadString('\n')
	qtd, _ := strconv.Atoi(strings.TrimSpace(qtdStr))

	fmt.Printf("\033[1;34mInstanciando %d drones funcionais em segundo plano...\033[0m\n", qtd)

	for i := 1; i <= qtd; i++ {
		go func(idx int) {
			droneID := fmt.Sprintf("virtual-drone-%02d", idx)

			// Inicializa o relógio lógico individual do drone virtuais
			localClock := &shared.LamportClock{Time: 0}

			// Conexão exclusiva estável do Drone Virtual
			opts := mqtt.NewClientOptions().AddBroker(brokerTarget).SetClientID(droneID + "-client")
			droneClient := mqtt.NewClient(opts)
			if token := droneClient.Connect(); token.Wait() && token.Error() != nil {
				return
			}

			defer func() {
				time.Sleep(5 * time.Minute)
				droneClient.Disconnect(250)
				fmt.Printf("\033[1;31m[DRONE %s]\033[0m Desligado após teste de estresse.\n", droneID)
			}()

			// 1. Enviar Registro Inicial para o cluster Raft entrar em consenso
			droneInfo := shared.Drone{
				ID:             droneID,
				BatteryLevel:   100,
				Status:         shared.DRONE_IDLE,
				CurrentBroker:  brokerTarget,
				CurrentMission: shared.NONE,
			}
			regPayload, _ := json.Marshal(droneInfo)
			droneClient.Publish("drones/register", 1, false, regPayload).Wait()

			// 2. Loop Assíncrono de Heartbeats (Evita remoção pelo Watchdog)
			go func(client mqtt.Client, id string) {
				ticker := time.NewTicker(4 * time.Second)
				defer ticker.Stop()
				hbTopic := fmt.Sprintf("drones/%s/heartbeat", id)
				for range ticker.C {
					if !droneClient.IsConnected() {
						return
					}
					hbData := shared.DroneHeartbeat{ID: id, BatteryLevel: 100}
					hbPayload, _ := json.Marshal(hbData)
					client.Publish(hbTopic, 1, false, hbPayload)
				}
			}(droneClient, droneID)

			// 3. Inscrever-se no tópico de recepção de Missões atribuídas pelo líder
			missionTopic := fmt.Sprintf("drones/%s/mission", droneID)
			droneClient.Subscribe(missionTopic, 1, func(c mqtt.Client, msg mqtt.Message) {
				var mission shared.DroneMission
				if err := json.Unmarshal(msg.Payload(), &mission); err != nil {
					return
				}

				// Executa a lógica de sincronização temporal de Lamport recebendo o carimbo do cluster
				localClock.CompareAndUpdate(mission.LamportTime)

				fmt.Printf("\n\033[1;33m[DRONE %s]\033[0m Atribuído para Incidente: %s. Processando ação (%s)...\n",
					droneID, mission.RequisitionID, mission.Type)

				// Simula o tempo de execução da missão (3s)
				time.Sleep(3 * time.Second)

				// Incrementa o relógio local do drone indicando a conclusão interna do trabalho
				localClock.Tick()

				// Monta o payload de Done Info para se auto-liberar
				doneInfo := shared.DoneInfo{
					RequisitionID: mission.RequisitionID,
					DroneID:       droneID,
					LCTime:        localClock.GetTime(),
				}

				donePayload, _ := json.Marshal(doneInfo)
				doneTopic := fmt.Sprintf("drones/%s/done", droneID)

				// Publica no barramento. O onDoneHandler do Manager vai capturar e atualizar a FSM Raft!
				c.Publish(doneTopic, 1, false, donePayload).Wait()
				fmt.Printf("\033[1;32m[DRONE %s]\033[0m Missão %s concluída e enviada com Lamport: %d. Drone livre!\n",
					droneID, mission.RequisitionID, localClock.GetTime())
			}).Wait()

		}(i)
	}

	fmt.Printf("\033[1;32m[SUCESSO]\033[0m %d drones ativos e ouvindo missões. Serão derrubados em 5 minutos.\n", qtd)
}
