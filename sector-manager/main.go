package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
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

func setupRaft(dir, id, addr string, fsm *RaftFSM, bootstrap bool) (*raft.Raft, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	transport, err := raft.NewTCPTransport(addr, tcpAddr, 3, 10*time.Second, os.Stderr)
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
				Address: transport.LocalAddr(),
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
	raftAddrFlag := flag.String("raft", "127.0.0.1:10001", "Endereço Raft")
	dataDirFlag := flag.String("dir", "data/node1", "Diretório de dados")
	bootstrapFlag := flag.Bool("bootstrap", false, "Iniciar como líder")
	peersFlag := flag.String("peers", "", "Endereços dos outros nós (host:porta ou host, separados por vírgula)")
	sigAddrFlag := flag.String("sig", "127.0.0.1:9123", "Endereço do servidor de sinalização")
	brokerAddrFlag := flag.String("broker", "tcp://127.0.0.1:1883", "Endereço do broker MQTT")
	flag.Parse()

	var peers []string

	if *peersFlag != "" {
		peers = strings.Split(*peersFlag, ",")
	}

	// --- Inicialização do Raft ---
	fsm := &RaftFSM{
		DroneMap:     make(map[string]shared.Drone),
		IncidentList: []shared.Incident{},
	}

	raftNode, err := setupRaft(*dataDirFlag, *nodeIDFlag, *raftAddrFlag, fsm, *bootstrapFlag)
	if err != nil {
		fmt.Printf("Erro ao iniciar Raft: %v\n", err)
		return
	}

	go startSignalingServer(raftNode, *sigAddrFlag, *brokerAddrFlag)

	// --- Inicialização do MQTT ---
	client, _ := shared.MakeClient(*brokerAddrFlag, *nodeIDFlag)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Erro MQTT: %v\n", token.Error())
	} else {
		go MQTTJoinHandler(raftNode, client)
	}

	if !*bootstrapFlag {
		fmt.Println("Prcurando líder na lista de peers...")
		leaderInfo := searchForLeaderInfo(peers)

		if leaderInfo.RaftAddr == "" {
			fmt.Println("Não foi possível encontrar o líder")
			return
		}

		joinLeaderViaMQTT(leaderInfo.BrokerAddr, *nodeIDFlag, *raftAddrFlag)
	}

	//mainMenu(client)

	fmt.Printf("Nó %s (Raft: %s) em execução...\n", *nodeIDFlag, *raftAddrFlag)
	select {}
}
