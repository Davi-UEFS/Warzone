package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
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

func setupRaft(dir, id, raftAddr string, fsm *RaftFSM, bootstrap bool) (*raft.Raft, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)

	filtered := &shared.FilteredWriter{
		Output: os.Stderr,
		Filters: []string{
			"dial tcp",
			"failed to appendEntries to",
		},
	}

	config.Logger = hclog.New(&hclog.LoggerOptions{
		Name:   "raft",
		Level:  hclog.Error,
		Output: filtered,
	})

	tcpAddr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, err
	}

	transport, err := raft.NewTCPTransport(raftAddr, tcpAddr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dir, "log.db"))
	if err != nil {
		return nil, err
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dir, "stable.db"))
	if err != nil {
		return nil, err
	}

	snapshots, err := raft.NewFileSnapshotStore(dir, 3, os.Stderr)
	if err != nil {
		return nil, err
	}

	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		return nil, err
	}

	if bootstrap {
		cfg := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: raft.ServerAddress(raftAddr),
			}},
		}
		if err := raftNode.BootstrapCluster(cfg).Error(); err != nil && err != raft.ErrCantBootstrap {
			return nil, err
		}
	}

	return raftNode, nil
}

func main() {
	nodeIDFlag := flag.String("id", "node1", "ID único deste nó")
	hostFlag := flag.String("host", "127.0.0.1", "Host base para Raft e SIG (modo host)")
	raftPortFlag := flag.Int("raft-port", 10001, "Porta Raft")
	sigPortFlag := flag.Int("sig-port", 9123, "Porta do servidor de sinalização")
	brokerHostFlag := flag.String("broker-host", "127.0.0.1", "Host do broker MQTT")
	brokerPortFlag := flag.Int("broker-port", 1883, "Porta do broker MQTT")

	raftAddrFlag := flag.String("raft", "", "Endereço Raft (host:porta). Se vazio usa host+porta")
	sigAddrFlag := flag.String("sig", "", "Endereço SIG (host:porta). Se vazio usa host+porta")
	brokerAddrFlag := flag.String("broker", "", "Endereço do broker MQTT (tcp://host:porta). Se vazio usa broker-host+porta")

	dataDirFlag := flag.String("dir", "data/node1", "Diretório de dados")
	bootstrapFlag := flag.Bool("bootstrap", false, "Iniciar como líder")
	peersFlag := flag.String("peers", "", "Endereços dos peers (SIG do líder). Separe por vírgula")
	flag.Parse()

	var peers []string
	if *peersFlag != "" {
		peers = strings.Split(*peersFlag, ",")
	}

	raftAddr := *raftAddrFlag
	if raftAddr == "" {
		raftAddr = net.JoinHostPort(*hostFlag, strconv.Itoa(*raftPortFlag))
	}

	sigAddr := *sigAddrFlag
	if sigAddr == "" {
		sigAddr = net.JoinHostPort(*hostFlag, strconv.Itoa(*sigPortFlag))
	}

	brokerAddr := *brokerAddrFlag
	if brokerAddr == "" {
		brokerAddr = net.JoinHostPort(*brokerHostFlag, strconv.Itoa(*brokerPortFlag))
	}
	brokerAddr = shared.NormalizeBrokerAddr(brokerAddr)

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

	go startSignalingServer(raftNode, sigAddr, brokerAddr)

	// --- Inicialização do MQTT ---
	client, err := shared.MakeClient(brokerAddr, *nodeIDFlag)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Erro MQTT: %v\n", token.Error())
	} else {
		go MQTTJoinHandler(raftNode, client)
	}

	if !*bootstrapFlag {
		fmt.Println("Procurando líder na lista de peers...")
		leaderInfo := searchForLeaderInfo(peers, *sigPortFlag)

		if leaderInfo.RaftAddr == "" {
			fmt.Println("Não foi possível encontrar o líder")
			return
		}

		joinLeaderViaMQTT(leaderInfo.BrokerAddr, *nodeIDFlag, raftAddr)
	}

	fmt.Printf("Nó %s em execução\n", *nodeIDFlag)
	fmt.Printf("Raft: %s | SIG: %s | Broker: %s\n", raftAddr, sigAddr, brokerAddr)
	select {}
}
