package main

import (
	"container/heap"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// normalizePeerAddr garante que o endereço do peer esteja no formato "host:port".
// Se o peer já tiver uma porta, retorna como está. Caso contrário, adiciona a porta padrão.
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

// Filtro para mensagens do Raft
type RaftStatesWriter struct{}

// Filtra mensagens de log do Raft e imprime mensagens customizadas para o usuário.
func (w *RaftStatesWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if strings.Contains(msg, "entering leader state") {
		return os.Stderr.Write([]byte("Atenção. Você agora é um líder!!\n"))
	}
	if strings.Contains(msg, "entering follower state") {
		return os.Stderr.Write([]byte("Entrando no modo seguidor (Follower)\n"))
	}
	if strings.Contains(msg, "entering candidate state") {
		return os.Stderr.Write([]byte("Tentando ganhar eleição...\n"))
	}

	if strings.Contains(msg, "Rollback failed: tx closed") {
		return os.Stderr.Write(nil)
	}

	return os.Stderr.Write(p)
}

func main() {
	// Flags de configuração
	nodeIDFlag := flag.String("id", "node1", "ID único deste nó")
	hostFlag := flag.String("host", "127.0.0.1", "Host base para Raft e SIG")
	raftPortFlag := flag.Int("raft-port", 10001, "Porta Raft")
	brokerPortFlag := flag.Int("broker-port", 1883, "Porta do broker MQTT")
	dashboardPortFlag := flag.Int("dashboard-port", 8080, "Porta do dashboard HTTP")
	dataDirFlag := flag.String("dir", "data/node1", "Diretório de dados")
	bootstrapFlag := flag.Bool("bootstrap", false, "Iniciar como líder")
	peersFlag := flag.String("peers", "", "Endereços dos peers (SIG do líder). Separe por vírgula")
	debugFlag := flag.Bool("debug", false, "Ativa simulações de latência, aging e logs de Lamport")
	flag.Parse()

	DebugMode = *debugFlag
	// Endereços
	sigPort = *raftPortFlag + 1000
	raftAddr := net.JoinHostPort(*hostFlag, strconv.Itoa(*raftPortFlag))
	sigAddr := net.JoinHostPort(*hostFlag, strconv.Itoa(sigPort))
	peers = strings.Split(*peersFlag, ",")

	// Como o broker é embutido, o IP é sempre o localhost (LEMBRAR DE TIRAR A FLAG DE HOST DEPOIS)
	brokerAddr := shared.NormalizeBrokerAddr(net.JoinHostPort("127.0.0.1", strconv.Itoa(*brokerPortFlag)))

	// --- INICIALIZAÇÃO DO RAFT ---

	log.SetOutput(&RaftStatesWriter{})

	LClock = shared.LamportClock{
		Time:  0,
		Mu:    sync.Mutex{},
		Debug: DebugMode,
	}

	// --- Inicializa FSM do Raft
	sectorFSM = &RaftFSM{
		Mu:               sync.Mutex{},
		DroneMap:         make(map[string]shared.Drone),
		PendingReqsQueue: ReqHeap{},
		InProgressReqs:   map[string]shared.Requisition{},
		EventsChan:       make(chan MissionPublishEvent, 4096),
	}

	// Inicializa heap da fila de requisições
	heap.Init(&sectorFSM.PendingReqsQueue)

	var err error
	alreadyInDB := false

	raftNode, alreadyInDB, err = setupRaft(*dataDirFlag, *nodeIDFlag, raftAddr, sectorFSM, *bootstrapFlag)
	if err != nil {
		log.Fatalf("Erro ao iniciar Raft: %v\n", err)
	}

	fmt.Println("Aguardando Raft estabilizar...")
	time.Sleep(2 * time.Second)

	go startSignaling(raftNode, sigAddr)

	// --- INICIALIZA O BROKER EMBUTIDO ---
	startEmbeddedBroker(*brokerPortFlag)

	// Aguarda um segundo para garantir que a porta TCP do broker foi aberta
	time.Sleep(1 * time.Second)

	// --- INICIALIZAÇÃO DO CLIENTE MQTT LOCAL (PAHO) ---

	globalClient, err = shared.MakeClient(brokerAddr, *nodeIDFlag+"-client", onConnect, false)
	if err != nil {
		log.Fatalf("Erro ao conectar Paho MQTT local: %v\n", err)
	}

	sectorFSM.Sector = *nodeIDFlag

	go publishToDrones(sectorFSM.EventsChan, globalClient)

	// --- 4. LÓGICA DE JOIN NO CLUSTER ---

	if alreadyInDB {
		fmt.Printf("\nEste nó é pre-existente. Foi carregado do DB do disco.\n")
	} else {
		if !*bootstrapFlag {
			fmt.Println("Procurando líder na lista de peers...")
			leaderInfo := searchForLeaderInfo(peers, sigPort)

			if leaderInfo.RaftAddr == "" {
				fmt.Println("Não foi possível encontrar o líder.")
			} else {
				req := joinReq{
					ID:   *nodeIDFlag,
					Addr: raftAddr,
				}

				reqPayload, _ := json.Marshal(req)

				cmd := shared.HeaderCommand{
					Operation: JOIN,
					Payload:   reqPayload,
				}

				if err := sendJoinRequest(leaderInfo.SigAddr, cmd); err != nil {
					fmt.Printf("Erro ao enviar join request: %v\n", err)
				} else {
					fmt.Println("Join request enviado, aguardando replicação...")
				}

			}
		}
	}

	fmt.Printf("NÓ %s EM EXECUÇÃO\n", *nodeIDFlag)
	fmt.Printf("Raft: %s | Endereço de escuta: %s | Broker Embutido: :%d\n", raftAddr, sigAddr, *brokerPortFlag)

	go startDispatcher()
	go startDashboardServer(*dashboardPortFlag)

	select {}
}
