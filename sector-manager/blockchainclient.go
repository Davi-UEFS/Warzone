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

// Qualquer tx enviada para a blockchain precisa pagar uma taxa mínima (fees) para ser aceita no bloco.
const FEES = "20stake"

// BlockchainMission mapeia como a blockchain devolve os dados. Devemos converter para o formato interno (shared.Requisition) antes de usar.
type BlockchainMission struct {
	Id              string `json:"id"`
	Sector          string `json:"sector"`
	Status          string `json:"status"`
	Priority        int    `json:"priority,string"`
	ReqType         string `json:"reqType"`
	Coord           string `json:"coord"`
	AssignedDroneId string `json:"assignedDroneId"`
}

// BlockchainDrone mapeia como a blockchain devolve os dados. Devemos converter para o formato interno (shared.Drone) antes de usar.
type BlockchainDrone struct {
	DroneId          string `json:"drone_id"`
	Sector           string `json:"sector"`
	Status           string `json:"status"`
	Battery          string `json:"battery"` // Blockchain devolve string
	CurrentMissionId string `json:"currentMissionId"`
}

// RequisitionsResponse mapeia a resposta REST da blockchain para requisitions
type RequisitionsResponse struct {
	Missions []BlockchainMission `json:"mission"`
}

// DronesResponse mapeia a resposta REST da blockchain para drones
type DronesResponse struct {
	APIReturnedDrones []BlockchainDrone `json:"drone"`
}

// Retorna o caminho do binário da blockchain (warzone-cored) a partir da variável de ambiente WARZONE_BIN.
func getBinPath() string {
	bin := os.Getenv("WARZONE_BIN")
	if bin == "" {
		bin = "/root/warzone-cored"
	}
	return bin
}

// Retorna o chain-id da blockchain a partir da variável de ambiente CHAIN_ID, ou usa o padrão "warzone-rede" se não estiver definido.
func getChainID() string {
	chainID := os.Getenv("CHAIN_ID")
	if chainID == "" {
		chainID = "warzone-rede"
	}
	return chainID
}

// Retorna a URL do RPC da blockchain a partir da variável de ambiente BLOCKCHAIN_RPC_URL, ou usa o padrão "http://localhost:26657" se não estiver definido.
func getRPCURL() string {
	rpc := os.Getenv("BLOCKCHAIN_RPC_URL")
	if rpc == "" {
		rpc = "http://localhost:26657"
	}
	return rpc
}

// Mutex global de transação para evitar "Sequence Mismatch" no Cosmos SDK
var txMutex sync.Mutex

// extrairTxHash extrai o txhash do output da CLI do Cosmos SDK
func extrairTxHash(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "txhash:") {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "txhash:"))
		}
	}
	return ""
}

// verificarTx aguarda o bloco confirmar e verifica se a tx foi aceita pelo keeper.
// Retorna (sucesso, raw_log) para facilitar o diagnóstico de erros.
func verificarTx(txhash string) (bool, string) {
	if txhash == "" {
		return false, "txhash vazio — tx pode não ter chegado à mempool"
	}

	// Aguarda ~1 bloco para a tx ser incluída (~5s no Tendermint)
	time.Sleep(6500 * time.Millisecond)

	binPath := getBinPath()
	cmd := exec.Command(binPath, "query", "tx", txhash,
		"--node", getRPCURL(),
		"--output", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Sprintf("erro ao consultar tx: %v", err)
	}

	var result struct {
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Sprintf("erro ao parsear resposta: %v", err)
	}

	if result.Code != 0 {
		return false, result.RawLog
	}
	return true, ""
}

// fetchRequisitionsFromBlockchain faz o polling e já converte os dados para o tipo esperado.
func fetchRequisitionsFromBlockchain() ([]shared.Requisition, error) {
	urlsEnv := os.Getenv("BLOCKCHAIN_REST_URLS")
	if urlsEnv == "" {
		urlsEnv = "http://localhost:1317"
	}

	endpoints := strings.Split(urlsEnv, ",")
	var lastErr error

	for _, ip := range endpoints {
		url := strings.TrimSpace(ip) + "/blockchain/warzone/requisicoes" // Endpoint REST da blockchain para requisitions

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

		var data RequisitionsResponse // Estrutura para armazenar a resposta JSON com as requisições.
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("erro no parse do JSON: %v. Body: %s", err, string(body))
		}

		// Converte os dados da blockchain para o formato interno (shared.Requisition)
		var reqs []shared.Requisition
		for _, bm := range data.Missions {
			if bm.Status != shared.PENDING {
				continue
			}

			coordStrParts := strings.Split(bm.Coord, ",")
			lat, _ := strconv.Atoi(coordStrParts[0])
			lng := 0
			if len(coordStrParts) > 1 {
				lng, _ = strconv.Atoi(coordStrParts[1])
			}

			novaReq := shared.Requisition{
				ID:           fmt.Sprintf("inc--%s--%s", bm.Sector, bm.Id),
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
	// Se nenhum endpoint respondeu com sucesso, retorna o último erro encontrado.
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
		url := strings.TrimSpace(ip) + "/blockchain/warzone/drones" // Endpoint REST da blockchain para drones

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

		var data DronesResponse // Estrutura para armazenar a resposta JSON com os drones.
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("erro no parse do JSON (Drone): %v. Body: %s", err, string(body))
		}

		var drones []shared.Drone // Converte os dados da blockchain para o formato interno (shared.Drone)
		for _, blockDrone := range data.APIReturnedDrones {
			batteryLvl, _ := strconv.Atoi(blockDrone.Battery)
			status := shared.DroneStatus(blockDrone.Status)
			if status == "" {
				status = shared.DRONE_IDLE
			}

			novoDrone := shared.Drone{
				ID:             blockDrone.DroneId,
				BatteryLevel:   batteryLvl,
				Status:         status,
				CurrentSector:  blockDrone.Sector,
				CurrentMission: blockDrone.CurrentMissionId,
			}
			drones = append(drones, novoDrone)
		}

		return drones, nil
	}

	return nil, fmt.Errorf("todos os nós falharam. Último erro: %v", lastErr)
}

// Pega o nome da carteira do setor a partir da variável de ambiente WALLET_NAME, ou usa o padrão "manager_setor_a" se não estiver definido.
func getSectorWalletName() string {
	wallet := os.Getenv("WALLET_NAME")
	if wallet == "" {
		return "manager_setor_a"
	}
	return wallet
}

// --- FUNÇÕES DE TRANSAÇÃO BLOCKCHAIN (CLI) ---

// enviarAssignDroneParaBlockchain envia a tx e verifica se foi confirmada no bloco.
// Retorna error se a tx foi rejeitada (ex: duplo despacho detectado pelo keeper).
func enviarAssignDroneParaBlockchain(missionID string, droneID string) error {
	txMutex.Lock()
	defer txMutex.Unlock()

	binPath := getBinPath()
	wallet := getSectorWalletName()

	partes := strings.Split(missionID, "--")
	idNumerico := partes[len(partes)-1]

	cmd := exec.Command(binPath, "tx", "warzone", "assign-drone", idNumerico, droneID, // Argumentos usados para fazer a lógica dentro do keeper da blockchain.
		"--from", wallet, // Carteira do setor que está enviando a tx
		"--home", KeyringDir, // Diretório do keyring (onde estão as chaves privadas)
		"--keyring-backend", "test", // Backend do keyring (test para desenvolvimento)
		"--chain-id", getChainID(), // Chain ID da blockchain (warzone-rede)
		"--node", getRPCURL(), // URL do nó RPC da blockchain
		"--fees", FEES, // Valor da taxa de transação (fees)
		"--broadcast-mode", "sync", // Modo de broadcast da tx (sync para aguardar a inclusão no bloco)
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao enviar assign-drone %s.\nOutput: %s\n", droneID, string(output))
		return fmt.Errorf("falha ao enviar tx: %v", err)
	}

	// Extrai o txhash e verifica confirmação no bloco para segurança.
	txhash := extrairTxHash(string(output))
	log.Printf("\033[1;34m[BLOCKCHAIN]\033[0m Assign-drone enviado. TxHash: %s. Aguardando confirmação...\n", txhash)

	// Verifica se a tx foi aceita no bloco (evita duplo despacho)
	if ok, rawLog := verificarTx(txhash); !ok {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Duplo despacho detectado para drone %s! Motivo: %s\n", droneID, rawLog)
		return fmt.Errorf("tx rejeitada: %s", rawLog)
	}

	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Missão %s atribuída ao drone %s confirmada no bloco!\n", idNumerico, droneID)
	return nil
}

// enviarLaudoParaBlockchain envia o laudo da missão para a blockchain e aguarda confirmação no bloco.
func enviarLaudoParaBlockchain(reqID string, droneID string, relatorio string) {
	txMutex.Lock()
	defer txMutex.Unlock()

	binPath := getBinPath()
	wallet := getSectorWalletName()

	// Aqui fica tudo igual então não irei comentar.

	cmd := exec.Command(binPath, "tx", "warzone", "submit-laudo", reqID, droneID, relatorio, "concluido",
		"--from", wallet,
		"--home", KeyringDir,
		"--keyring-backend", "test",
		"--chain-id", getChainID(),
		"--node", getRPCURL(),
		"--fees", FEES,
		"--broadcast-mode", "sync",
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao submeter laudo %s.\nOutput: %s\n", reqID, string(output))
		return
	}

	txhash := extrairTxHash(string(output))
	if ok, rawLog := verificarTx(txhash); !ok {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Laudo da missão %s rejeitado! Motivo: %s\n", reqID, rawLog)
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Laudo da missão %s confirmado no bloco! TxHash auditável: %s\n", reqID, txhash)
}

// enviarRequisicaoParaBlockchain envia a requisição de alerta para a blockchain e aguarda confirmação no bloco.
// A cobrança de tx do país é feita aqui.
func enviarRequisicaoParaBlockchain(alert shared.Alert) {
	txMutex.Lock()
	defer txMutex.Unlock()

	binPath := getBinPath()
	wallet := getSectorWalletName()
	sector := os.Getenv("SECTOR_ID")

	if sector == "" {
		sector = "Setor-A"
	}

	// Precisamos converter os tipos para string, pois a CLI do Cosmos SDK só aceita strings como argumentos.
	coordStr := fmt.Sprintf("%d,%d", alert.Coordinate.Longitude, alert.Coordinate.Latitude)
	reqType := fmt.Sprintf("%s", alert.Type)
	priority := strconv.Itoa(PRIOTIRIES[alert.Type])

	enderecoPagante, existe := EnderecosPaises[alert.Country]
	if !existe {
		log.Printf("\033[1;33m[WARNING]\033[0m País %s não encontrado no dicionário de endereços. Abortando...\n", alert.Country)
		return
	}

	cmd := exec.Command(
		binPath, "tx", "warzone", "add-req", sector, priority, reqType, coordStr,
		alert.ID,                   // IMPORTANTE !!!!!: O alertID enviado é apenas para ela verificar requisições duplicadas. A blockchain vai gerar o ID oficial da requisição.
		"--payer", enderecoPagante, // Unico diferencial. A carteira do país a ser cobrada pela tx de requisição.
		"--from", wallet,
		"--keyring-backend", "test",
		"--home", KeyringDir,
		"--chain-id", getChainID(),
		"--node", getRPCURL(),
		"--fees", FEES,
		"--broadcast-mode", "sync",
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao criar requisição no bloco.\nOutput: %s\n", string(output))
		return
	}

	// Pode falhar por diversos motivos, como req já adicionada, falta de saldo, etc.
	txhash := extrairTxHash(string(output))
	if ok, rawLog := verificarTx(txhash); !ok {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Requisição do alerta %s rejeitada! Motivo: %s\n", alert.ID, rawLog)
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Alerta transformado em requisição! TxHash: %s\n", txhash)
}

// enviarRegDroneParaBlockchain envia a tx de registro do drone para a blockchain e aguarda confirmação no bloco.
func enviarRegDroneParaBlockchain(droneID string, sector string, battery string) {
	txMutex.Lock()
	defer txMutex.Unlock()

	binPath := getBinPath()
	wallet := getSectorWalletName()

	cmd := exec.Command(binPath, "tx", "warzone", "reg-drone", droneID, sector, battery,
		"--from", wallet,
		"--home", KeyringDir,
		"--keyring-backend", "test",
		"--chain-id", getChainID(),
		"--node", getRPCURL(),
		"--fees", FEES,
		"--broadcast-mode", "sync",
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao registrar drone %s.\nOutput: %s\n", droneID, string(output))
		return
	}

	txhash := extrairTxHash(string(output))
	if ok, rawLog := verificarTx(txhash); !ok {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Registro do drone %s rejeitado! Motivo: %s\n", droneID, rawLog)
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Drone %s registrado globalmente com sucesso!\n", droneID)
}

// enviarRmvReqParaBlockchain envia a tx de remoção da requisição para a blockchain e aguarda confirmação no bloco.
// Na prática, ela não remove uma requisicão, mas sim marca como concluída (DONE).
func enviarRmvReqParaBlockchain(missionID string, droneID string, laudo string) {
	txMutex.Lock()
	defer txMutex.Unlock()

	binPath := getBinPath()
	wallet := getSectorWalletName()

	cmd := exec.Command(binPath, "tx", "warzone", "rmv-req", missionID, droneID, laudo,
		"--from", wallet,
		"--home", KeyringDir,
		"--keyring-backend", "test",
		"--chain-id", getChainID(),
		"--node", getRPCURL(),
		"--fees", FEES,
		"--broadcast-mode", "sync",
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao remover requisição %s.\nOutput: %s\n", missionID, string(output))
		return
	}

	txhash := extrairTxHash(string(output))
	if ok, rawLog := verificarTx(txhash); !ok {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Remoção da missão %s rejeitada! Motivo: %s\n", missionID, rawLog)
		return
	}
	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Missão %s finalizada na blockchain. Drone %s livre!\n", missionID, droneID)
}

// enviarReportDeadDroneParaBlockchain envia a tx de report de drone morto para a blockchain e aguarda confirmação no bloco.
func enviarReportDeadDroneParaBlockchain(droneID string) {
	txMutex.Lock()
	defer txMutex.Unlock()

	binPath := getBinPath()
	wallet := getSectorWalletName()

	cmd := exec.Command(binPath, "tx", "warzone", "report-dead-drone", droneID,
		"--from", wallet,
		"--home", KeyringDir,
		"--keyring-backend", "test",
		"--chain-id", getChainID(),
		"--node", getRPCURL(),
		"--fees", FEES,
		"--broadcast-mode", "sync",
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha ao reportar drone inativo %s.\nOutput: %s\n", droneID, string(output))
		return
	}

	txhash := extrairTxHash(string(output))
	if ok, rawLog := verificarTx(txhash); !ok {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Report de drone morto %s rejeitado! Motivo: %s\n", droneID, rawLog)
		return
	}
	fmt.Printf("\033[1;31m[BLOCKCHAIN]\033[0m Drone %s reportado como inativo na blockchain.\n", droneID)
}

// obterEnderecoPorApelido consulta a keyring para converter o apelido (ex: "key_a")
// no endereço bech32 real (ex: "cosmos1...") que a blockchain exige no bank send.
func obterEnderecoPorApelido(apelido string) string {
	binPath := getBinPath()
	// Consulta o endereço público associado ao apelido no banco de chaves local
	cmd := exec.Command(binPath, "keys", "show", apelido, "--address", "--keyring-backend", "test", "--home", KeyringDir)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[BLOCKCHAIN ERROR] Não foi possível encontrar endereço para o apelido %s: %v\n", apelido, err)
		return ""
	}
	return strings.TrimSpace(string(output))
}

// enviarTransferenciaParaBlockchain transfere 'valor' (ex: "1000stake")
// de uma carteira interna (apelidoOrigem) para um endereço destino (cosmos1...)
func enviarTransferenciaParaBlockchain(apelidoOrigem string, destinoAddr string, valor string) error {
	txMutex.Lock()
	defer txMutex.Unlock()

	binPath := getBinPath()
	// Converte o apelido de origem para o endereço real para o comando bank send
	enderecoOrigem := obterEnderecoPorApelido(apelidoOrigem)
	if enderecoOrigem == "" {
		return fmt.Errorf("apelido de origem inválido ou não encontrado na keyring")
	}

	// Comando bank send: origem (endereco) -> destino (endereco) -> valor
	// O --from usa o apelido para que a CLI saiba qual chave privada usar para assinar
	cmd := exec.Command(binPath, "tx", "bank", "send", enderecoOrigem, destinoAddr, valor,
		"--from", apelidoOrigem,
		"--node", getRPCURL(),
		"--home", KeyringDir,
		"--keyring-backend", "test",
		"--chain-id", getChainID(),
		"--fees", FEES,
		"--broadcast-mode", "sync",
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Falha na transferência de %s.\nOutput: %s\n", valor, string(output))
		return fmt.Errorf("falha ao enviar tx de transferência: %v", err)
	}

	// Extrai o hash e aguarda confirmação no bloco
	txhash := extrairTxHash(string(output))
	if ok, rawLog := verificarTx(txhash); !ok {
		log.Printf("\033[1;31m[BLOCKCHAIN ERROR]\033[0m Transferência rejeitada no bloco! Motivo: %s\n", rawLog)
		return fmt.Errorf("transferência rejeitada: %s", rawLog)
	}

	fmt.Printf("\033[1;32m[BLOCKCHAIN]\033[0m Transferência de %s realizada com sucesso! TxHash: %s\n", valor, txhash)
	return nil
}
