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

// ==========================================
// 2. LÓGICA DO SECTOR MANAGER E RAFT
// ==========================================

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
type MuteRaftDBWriter struct{}

func (w *MuteRaftDBWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if strings.Contains(msg, "Rollback failed: tx closed") {
		return len(p), nil
	}
	return os.Stderr.Write(p)
}

func main() {
	// Filtro para logs irritantes
	log.SetOutput(&MuteRaftDBWriter{})
	// Flags de configuração
	nodeIDFlag := flag.String("id", "node1", "ID único deste nó")
	hostFlag := flag.String("host", "127.0.0.1", "Host base para Raft e SIG")
	raftPortFlag := flag.Int("raft-port", 10001, "Porta Raft")
	brokerPortFlag := flag.Int("broker-port", 1883, "Porta do broker MQTT")
	dashboardPortFlag := flag.Int("dashboard-port", 8080, "Porta do dashboard HTTP")
	dataDirFlag := flag.String("dir", "data/node1", "Diretório de dados")
	bootstrapFlag := flag.Bool("bootstrap", false, "Iniciar como líder")
	peersFlag := flag.String("peers", "", "Endereços dos peers (SIG do líder). Separe por vírgula")
	flag.Parse()

	// --- 1. INICIALIZA O BROKER EMBUTIDO ---
	startEmbeddedBroker(*brokerPortFlag)

	// Aguarda um segundo para garantir que a porta TCP do broker foi aberta
	time.Sleep(1 * time.Second)

	// Endereços
	sigPort = *raftPortFlag + 1000
	raftAddr := net.JoinHostPort(*hostFlag, strconv.Itoa(*raftPortFlag))
	sigAddr := net.JoinHostPort(*hostFlag, strconv.Itoa(sigPort))
	peers = strings.Split(*peersFlag, ",")

	// Como o broker é embutido, o manager local sempre se conecta no localhost na porta do broker
	brokerAddr := shared.NormalizeBrokerAddr(net.JoinHostPort("127.0.0.1", strconv.Itoa(*brokerPortFlag)))

	// --- 2. INICIALIZAÇÃO DO RAFT ---
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
	raftNode, err = setupRaft(*dataDirFlag, *nodeIDFlag, raftAddr, sectorFSM, *bootstrapFlag)
	if err != nil {
		log.Fatalf("Erro ao iniciar Raft: %v\n", err)
	}

	go startSignaling(raftNode, sigAddr)

	// --- 3. INICIALIZAÇÃO DO CLIENTE MQTT LOCAL (PAHO) ---
	// Este client conecta-se ao broker que acabou de ser criado na própria máquina
	client, err := shared.MakeClient(brokerAddr, *nodeIDFlag+"-client", onConnect)
	if err != nil {
		log.Fatalf("Erro ao conectar Paho MQTT local: %v\n", err)
	}

	sectorFSM.Sector = *nodeIDFlag

	go goDrones(sectorFSM.EventsChan, client)

	// --- 4. LÓGICA DE JOIN NO CLUSTER ---
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

	fmt.Printf("NÓ %s EM EXECUÇÃO\n", *nodeIDFlag)
	fmt.Printf("Raft: %s | Endereço de escuta: %s | Broker Embutido: :%d\n", raftAddr, sigAddr, *brokerPortFlag)

	go startDispatcher()
	go startDashboardServer(*dashboardPortFlag)

	select {}
}
