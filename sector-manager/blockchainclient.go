package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// RequisitionsResponse mapeia a resposta da API para a lista de missões
type RequisitionsResponse struct {
	Requisitions []shared.Requisition `json:"requisition"` // Ajuste a tag json de acordo com o retorno da sua API (geralmente é singular no Cosmos)
}

// DronesResponse mapeia a resposta da API para a lista do censo de drones
type DronesResponse struct {
	Drones []shared.Drone `json:"drone"` // Padrão gerado pelo Ignite
}

// fetchRequisitionsFromBlockchain faz o polling no nó Tendermint com fallback de IPs.
func fetchRequisitionsFromBlockchain() ([]shared.Requisition, error) {
	urlsEnv := os.Getenv("BLOCKCHAIN_REST_URLS")
	if urlsEnv == "" {
		urlsEnv = "http://localhost:1317"
	}

	endpoints := strings.Split(urlsEnv, ",")
	var lastErr error

	for _, ip := range endpoints {
		url := strings.TrimSpace(ip) + "/warzone/warzone/req"

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

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler resposta REST: %v", err)
		}

		var data RequisitionsResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("erro no parse do JSON (Req): %v", err)
		}

		return data.Requisitions, nil
	}

	return nil, fmt.Errorf("todos os nós falharam. Último erro: %v", lastErr)
}

// fetchDronesFromBlockchain faz o polling do estado global dos drones na Blockchain.
func fetchDronesFromBlockchain() ([]shared.Drone, error) {
	urlsEnv := os.Getenv("BLOCKCHAIN_REST_URLS")
	if urlsEnv == "" {
		urlsEnv = "http://localhost:1317"
	}

	endpoints := strings.Split(urlsEnv, ",")
	var lastErr error

	for _, ip := range endpoints {
		url := strings.TrimSpace(ip) + "/warzone/warzone/drone"

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

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler resposta REST: %v", err)
		}

		var data DronesResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("erro no parse do JSON (Drone): %v", err)
		}

		return data.Drones, nil
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

func enviarAssignDroneParaBlockchain(droneID string, reqID string) {
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	cmd := exec.Command(binPath, "tx", "warzone", "assign-drone", reqID, droneID, "--from", wallet, "--chain-id", "warzone", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao atribuir drone %s: %v\nOutput: %s\n", droneID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Drone %s ocupado na blockchain pela carteira %s!\n", droneID, wallet)
}

func enviarLaudoParaBlockchain(reqID string, droneID string, relatorio string) {
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	cmd := exec.Command(binPath, "tx", "warzone", "submit-laudo", reqID, droneID, relatorio, "concluido", "--from", wallet, "--chain-id", "warzone", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao submeter laudo %s: %v\nOutput: %s\n", reqID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Laudo da %s salvo no bloco pela carteira %s!\n", reqID, wallet)
}

func enviarRequisicaoParaBlockchain(reqID string, alert shared.Alert) {
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	coordStr := fmt.Sprintf("%f,%f", alert.Coordinate.Latitude, alert.Coordinate.Longitude)

	cmd := exec.Command(binPath, "tx", "warzone", "add-req", reqID, fmt.Sprintf("%d", alert.Type), coordStr, os.Getenv("SECTOR_ID"), "--from", wallet, "--chain-id", "warzone", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao criar req %s: %v\nOutput: %s\n", reqID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Alerta %s transformado em requisição!\n", reqID)
}

func enviarRegDroneParaBlockchain(droneID string, sector string, battery string) {
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	// Ajuste a ordem dos argumentos conforme criado no seu scaffold (droneID, sector, battery)
	cmd := exec.Command(binPath, "tx", "warzone", "reg-drone", droneID, sector, battery, "--from", wallet, "--chain-id", "warzone", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao registrar drone %s: %v\nOutput: %s\n", droneID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Drone %s registrado globalmente com sucesso!\n", droneID)
}

func enviarRmvReqParaBlockchain(missionID string, droneID string, laudo string) {
	binPath := os.ExpandEnv("$HOME/go/bin/warzone-cored")
	wallet := getWalletName()

	// Ajuste a ordem dos argumentos conforme criado no seu scaffold (missionID, droneID, laudo)
	cmd := exec.Command(binPath, "tx", "warzone", "rmv-req", missionID, droneID, laudo, "--from", wallet, "--chain-id", "warzone", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao remover requisição %s: %v\nOutput: %s\n", missionID, err, string(output))
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Missão %s finalizada na blockchain. Drone %s livre!\n", missionID, droneID)
}
