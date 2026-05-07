package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func mainMenu(client mqtt.Client) {
	scanner := bufio.NewScanner(os.Stdin)
	var i int = 1

	for {
		//showMenu()		//TODO: ERRO AQUI
		scanner.Scan()
		option := scanner.Text()

		switch option {
		case "1":
			fmt.Println("EM BREVE...")

		case "2":
			cmd := shared.DroneCommand{
				OccurrenceID: fmt.Sprintf("cmd-%d", i),
				Action:       "oil",
				Timestamp:    time.Now(),
			}
			payload, _ := json.Marshal(cmd)
			sendCommand(client, payload)

		case "3":
			fmt.Print("ID do sensor: ")
			scanner.Scan()
			sensorID := scanner.Text()

			payload, _ := json.Marshal("DONE")

			topic := fmt.Sprintf("sensors/%s/solved", sensorID)
			token := client.Publish(topic, 2, false, payload)
			token.Wait()
			if token.Error() != nil {
				fmt.Println("Erro ao publicar:", token.Error())
			} else {
				fmt.Printf("→ Ocorrência marcada como resolvida no sensor %s\n", sensorID)
			}

		case "4":
			fmt.Println("Saindo...")
			client.Disconnect(250)
			return

		default:
			fmt.Println("Opção inválida")
		}

		i++
	}
}

// normalizePeerAddr assegura que o endereço dado está no formato IP + Porta
// Se não possuir porta, adiciona a porta dada.

func normalizePeerAddr(peer string, defaultPort int) string {
	trimmed := strings.TrimSpace(peer)
	if trimmed == "" {
		return ""
	}

	if _, _, err := net.SplitHostPort(trimmed); err == nil {
		return trimmed
	}

	return net.JoinHostPort(trimmed, strconv.Itoa(defaultPort))
}

func main() {

	// Identifica este nó no cluster.
	nodeIDFlag := flag.String("id", "node1", "ID único deste nó")
	// Define o host base usado para calcular os endereços do Raft e do SIG.
	hostFlag := flag.String("host", "127.0.0.1", "Host base para Raft e SIG (modo host)")
	// Porta usada pelo serviço Raft deste nó.
	raftPortFlag := flag.Int("raft-port", 10001, "Porta Raft")
	// Host do broker MQTT que será utilizado para publicar e assinar mensagens.
	brokerHostFlag := flag.String("broker-host", "127.0.0.1", "Host do broker MQTT")
	// Porta do broker MQTT.
	brokerPortFlag := flag.Int("broker-port", 1883, "Porta do broker MQTT")

	// Diretório onde o estado do Raft será persistido.
	dataDirFlag := flag.String("dir", "data/node1", "Diretório de dados")
	// Define se este nó deve iniciar como líder e montar o cluster localmente.
	bootstrapFlag := flag.Bool("bootstrap", false, "Iniciar como líder")
	// Lista de peers usada para descobrir o líder quando este nó não é bootstrap.
	peersFlag := flag.String("peers", "", "Endereços dos peers (SIG do líder). Separe por vírgula")
	flag.Parse()

	// Porta do serviço de sinalização, calculada a partir da porta do Raft.
	sigPort := *raftPortFlag + 1000
	// Endereço completo do Raft usando host base e porta configurada.
	raftAddr := net.JoinHostPort(*hostFlag, strconv.Itoa(*raftPortFlag))
	// Endereço completo do serviço SIG usando o mesmo host e a porta derivada.
	sigAddr := net.JoinHostPort(*hostFlag, strconv.Itoa(sigPort))
	// Lista de peers informada na flag, separada por vírgula.
	peers := strings.Split(*peersFlag, ",")
	// Endereço do broker MQTT normalizado para o formato esperado pela aplicação.
	brokerAddr := shared.NormalizeBrokerAddr(net.JoinHostPort(*brokerHostFlag, strconv.Itoa(*brokerPortFlag)))

	// --- Inicialização do Raft ---
	fsm := &RaftFSM{
		DroneMap:     make(map[string]shared.Drone),
		IncidentList: []shared.Incident{},
	}

	raftNode, err := setupRaft(*dataDirFlag, *nodeIDFlag, raftAddr, fsm, *bootstrapFlag)
	if err != nil {
		fmt.Printf("Erro ao iniciar Raft: %v\n", err)
		return
	}

	go startSignaling(raftNode, sigAddr)

	// --- Inicialização do MQTT ---
	client, err := shared.MakeClient(brokerAddr, *nodeIDFlag)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Erro MQTT: %v\n", token.Error())
	}

	if !*bootstrapFlag {
		fmt.Println("Procurando líder na lista de peers...")
		leaderInfo := searchForLeaderInfo(peers, sigPort)

		if leaderInfo.RaftAddr == "" {
			fmt.Println("Não foi possível encontrar o líder")
			return
		}

		req := joinReq{
			ID:   *nodeIDFlag,
			Addr: raftAddr,
		}

		reqPayload, err := json.Marshal(req)
		if err != nil {
			fmt.Printf("Erro ao serializar join request: %v\n", err)
			return
		}

		cmd := shared.HeaderCommand{
			Operation: JOIN,
			Payload:   reqPayload,
		}

		if err := sendJoinRequest(leaderInfo.SigAddr, cmd); err != nil {
			fmt.Printf("Erro ao enviar join request: %v\n", err)
			return
		}

		fmt.Println("Join request enviado, aguardando replicação...")

	}
	fmt.Printf("Nó %s em execução\n", *nodeIDFlag)
	fmt.Printf("Raft: %s | SIG: %s | Broker: %s\n", raftAddr, sigAddr, brokerAddr)
	select {}
}
