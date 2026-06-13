package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// BlockchainMission mapeia EXATAMENTE como a blockchain devolve os dados
type BlockchainMission struct {
	Id              string `json:"id"`
	Sector          string `json:"sector"`
	Status          string `json:"status"`
	Priority        int    `json:"priority,string"`
	ReqType         string `json:"reqType"`
	Coord           string `json:"coord"`
	AssignedDroneId string `json:"assignedDroneId"`
}

// BlockchainDrone mapeia EXATAMENTE como a blockchain devolve os dados
type BlockchainDrone struct {
	DroneId string `json:"drone_id"`
	Sector  string `json:"sector"`
	Status  string `json:"status"`
	Battery string `json:"battery"` // Blockchain devolve string
}

type RequisitionsResponse struct {
	Missions []BlockchainMission `json:"mission"`
}

type DronesResponse struct {
	APIReturnedDrones []BlockchainDrone `json:"drone"`
}

// Mutex global de transação para evitar "Sequence Mismatch" no Cosmos SDK quando muitas transações são enviadas em paralelo.
var txMutex sync.Mutex

// fetchRequisitionsFromBlockchain faz o polling e CONVERTE os dados
func fetchRequisitionsFromBlockchain() ([]shared.Requisition, error) {
	urlsEnv := os.Getenv("BLOCKCHAIN_REST_URLS")
	if urlsEnv == "" {
		urlsEnv = "http://localhost:1317"
	}

	endpoints := strings.Split(urlsEnv, ",")
	var lastErr error

	for _, ip := range endpoints {
		url := strings.TrimSpace(ip) + "/Davi-UEFS/warzone-core/warzone/v1/mission"

		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("código %d recebido do nó %s", resp.StatusCode, ip)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("erro ao ler resposta REST: %v", err)
		}

		// Deserializa usando a struct intermediária
		var data RequisitionsResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("erro no parse do JSON: %v. Body: %s", err, string(body))
		}

		// Converte para a sua shared.Requisition original
		var reqs []shared.Requisition
		for _, bm := range data.Missions {
			// Apenas missões PENDING importam para o Dispatcher
			if bm.Status != shared.PENDING {
				continue
			}

			// Simula uma coordenada baseada na string enviada
			coordStrParts := strings.Split(bm.Coord, ",")
			lat, _ := strconv.Atoi(coordStrParts[0]) // Extração simplificada
			lng := 0
			if len(coordStrParts) > 1 {
				lng, _ = strconv.Atoi(coordStrParts[1])
			}

			novaReq := shared.Requisition{
				ID:           fmt.Sprintf("inc--%s--%s", bm.Sector, bm.Id), // Cria um ID compativel
				Priority:     bm.Priority,
				Type:         bm.ReqType,
				OriginSector: bm.Sector,
				Coord: shared.Coordinate{
					Latitude:  lat,
					Longitude: lng,
				},
			}
			reqs = append(reqs, novaReq)
		}

		return reqs, nil
	}

	return nil, fmt.Errorf("todos os nós falharam. Último erro: %v", lastErr)
}

// fetchDronesFromBlockchain faz o polling e CONVERTE os dados
func fetchDronesFromBlockchain() ([]shared.Drone, error) {
	urlsEnv := os.Getenv("BLOCKCHAIN_REST_URLS")
	if urlsEnv == "" {
		urlsEnv = "http://localhost:1317"
	}

	endpoints := strings.Split(urlsEnv, ",")
	var lastErr error

	for _, ip := range endpoints {
		url := strings.TrimSpace(ip) + "/Davi-UEFS/warzone-core/warzone/v1/drone"

		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("código %d recebido do nó %s", resp.StatusCode, ip)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("erro ao ler resposta REST: %v", err)
		}

		// Deserializa usando a struct intermediária
		var data DronesResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("erro no parse do JSON (Drone): %v. Body: %s", err, string(body))
		}

		// Converte para a sua shared.Drone original
		var drones []shared.Drone
		for _, blockDrone := range data.APIReturnedDrones {
			// Converte a bateria de string para int
			batteryLvl, _ := strconv.Atoi(blockDrone.Battery)

			// Converte o status de string para o tipo DroneStatus
			status := shared.DroneStatus(blockDrone.Status)
			if status == "" {
				status = shared.DRONE_IDLE // Prevenção caso venha vazio
			}

			novoDrone := shared.Drone{
				ID:            blockDrone.DroneId,
				BatteryLevel:  batteryLvl,
				Status:        status,
				CurrentSector: blockDrone.Sector,
			}

			//TODO: DEBUG
			log.Printf("\033[1;35m[DEBUG DRONES]\033[0m Drone %s da Blockchain: Status=%s, Bateria=%d%%\n", blockDrone.DroneId, novoDrone.Status, novoDrone.BatteryLevel)

			drones = append(drones, novoDrone)
		}

		return drones, nil
	}

	return nil, fmt.Errorf("todos os nós falharam. Último erro: %v", lastErr)
}

// getWalletName puxa o nome da carteira dada por variável de ambiente.
func getWalletName() string {
	wallet := os.Getenv("WALLET_NAME")
	if wallet == "" {
		return "manager_setor_a"
	}
	return wallet
}

// --- FUNÇÕES DE TRANSAÇÃO BLOCKCHAIN (CLI) ---

func enviarAssignDroneParaBlockchain(missionID string, droneID string) {
	// Evita envios paralelos que causam "Sequence Mismatch" no Cosmos SDK
	txMutex.Lock()
	defer txMutex.Unlock()
	defer time.Sleep(2 * time.Second)
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	// 1. Extrai apenas o número final da string "inc--Setor-A--123"
	partes := strings.Split(missionID, "--")
	idNumerico := partes[len(partes)-1] // Pega a última parte (o número)

	// 2. Usa o idNumerico limpo ("0") no comando
	cmd := exec.Command(binPath, "tx", "warzone", "assign-drone", idNumerico, droneID, "--from", wallet, "--chain-id", "warzonecore", "-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao atribuir drone %s: %v\nOutput: %s\n", droneID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Missão %s atribuída ao drone %s com sucesso!\n", idNumerico, droneID)
}

func enviarLaudoParaBlockchain(reqID string, droneID string, relatorio string) {
	// Evita envios paralelos que causam "Sequence Mismatch" no Cosmos SDK
	txMutex.Lock()
	defer txMutex.Unlock()
	defer time.Sleep(2 * time.Second)
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	cmd := exec.Command(binPath, "tx", "warzone", "submit-laudo", reqID, droneID, relatorio, "concluido", "--from", wallet, "--chain-id", "warzonecore", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao submeter laudo %s: %v\nOutput: %s\n", reqID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Laudo da %s salvo no bloco pela carteira %s!\n", reqID, wallet)
}

func enviarRequisicaoParaBlockchain(alert shared.Alert) {
	// Evita envios paralelos que causam "Sequence Mismatch" no Cosmos SDK
	txMutex.Lock()
	defer txMutex.Unlock()
	defer time.Sleep(2 * time.Second)
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()
	sector := os.Getenv("SECTOR_ID")

	if sector == "" {
		sector = "Setor-A"
	}

	// A coordenada volta a ser limpa, sem X: ou Y:
	coordStr := fmt.Sprintf("%d,%d", alert.Coordinate.Longitude, alert.Coordinate.Latitude)
	reqType := fmt.Sprintf("%s", alert.Type)
	priority := strconv.Itoa(PRIOTIRIES[alert.Type]) // Converter por que o command line espera string

	// Comando perfeitamente alinhado com o autocli.go: [sector] [priority] [req-type] [coord]
	cmd := exec.Command(binPath, "tx", "warzone", "add-req", sector, priority, reqType, coordStr, "--from", wallet, "--chain-id", "warzonecore", "-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao criar req: %v\nOutput: %s\n", err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Alerta transformado em requisição com sucesso!\n")
}

func enviarRegDroneParaBlockchain(droneID string, sector string, battery string) {
	// Evita envios paralelos que causam "Sequence Mismatch" no Cosmos SDK
	txMutex.Lock()
	defer txMutex.Unlock()
	defer time.Sleep(2 * time.Second)
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	// Ajuste a ordem dos argumentos conforme criado no seu scaffold (droneID, sector, battery)
	cmd := exec.Command(binPath, "tx", "warzone", "reg-drone", droneID, sector, battery, "--from", wallet, "--chain-id", "warzonecore", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao registrar drone %s: %v\nOutput: %s\n", droneID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Drone %s registrado globalmente com sucesso!\n", droneID)
}

func enviarRmvReqParaBlockchain(missionID string, droneID string, laudo string) {
	// Evita envios paralelos que causam "Sequence Mismatch" no Cosmos SDK
	txMutex.Lock()
	defer txMutex.Unlock()
	defer time.Sleep(2 * time.Second)
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	// Comando alinhado: [mission-id] [drone-id] [laudo]
	cmd := exec.Command(binPath, "tx", "warzone", "rmv-req", missionID, droneID, laudo, "--from", wallet, "--chain-id", "warzonecore", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao remover requisição %s: %v\nOutput: %s\n", missionID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Missão %s finalizada na blockchain. Drone %s livre!\n", missionID, droneID)
}

func enviarReportDeadDroneParaBlockchain(droneID string) {
	// Evita envios paralelos que causam "Sequence Mismatch" no Cosmos SDK
	txMutex.Lock()
	defer txMutex.Unlock()
	defer time.Sleep(2 * time.Second)
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	// O comando usa report-dead-drone e precisa apenas do ID do drone
	cmd := exec.Command(binPath, "tx", "warzone", "report-dead-drone", droneID, "--from", wallet, "--chain-id", "warzonecore", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao reportar drone inativo %s: %v\nOutput: %s\n", droneID, err, string(output))
		return
	}
	fmt.Printf("\033[1;31m[BLOCKCHAIN]\033[0m Comando de morte enviado. Resposta da rede: %s\n", string(output))
}
