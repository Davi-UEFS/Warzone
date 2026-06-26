package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

// Estruturas auxiliares para a API da Blockchain
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type BalancesResponse struct {
	Balances []Coin `json:"balances"`
}

// CompanyBalance representa os fundos de um país no dashboard
type CompanyBalance struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Balance string `json:"balance"`
}

// DashboardState contém os campos para visualização em HTML.
type DashboardState struct {
	Pending     []shared.Requisition `json:"pending"`
	InProgress  []shared.Requisition `json:"in_progress"`
	Logs        []string             `json:"logs"`
	Drones      []shared.Drone       `json:"drones"` // Agora usa o Drone compartilhado diretamente
	Sensors     []string             `json:"sensors"`
	Balances    []CompanyBalance     `json:"balances"` // NOVO: Saldos da Blockchain
	GeneratedAt int64                `json:"generated_at"`
	Sector      string               `json:"sector"`
}

//go:embed GUI/dashboard.html
var dashboardHTML []byte

func dashboardIndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.URL.Path != "/" && r.URL.Path != "/dashboard" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(dashboardHTML)
}

func startDashboardServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", dashboardStateHandler)
	mux.HandleFunc("/", dashboardIndexHandler)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Dashboard disponível em http://localhost:%d\n", port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro no servidor do dashboard: %v\n", err)
	}
}

func dashboardStateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	state := buildDashboardState()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(state)
}

// buildDashboardState consolida dados locais (Memória) + dados globais (Blockchain)
// buildDashboardState consolida dados locais (Memória) + dados globais (Blockchain)
func buildDashboardState() DashboardState {
	state := DashboardState{
		GeneratedAt: time.Now().Unix(),
		Sector:      os.Getenv("SECTOR_ID"),
	}
	if state.Sector == "" {
		state.Sector = "Setor-Desconhecido"
	}

	// 1. DADOS DA BLOCKCHAIN: Busca Drones registrados na rede
	if drones, err := fetchDronesFromBlockchain(); err == nil {
		state.Drones = drones
	}

	// 2. DADOS DA BLOCKCHAIN: Busca Requisições (Missões)
	if reqs, err := fetchRequisitionsFromBlockchain(); err == nil {
		// A sua função fetchRequisitionsFromBlockchain já filtra as PENDENTES
		state.Pending = reqs
	}

	// 3. DADOS DA BLOCKCHAIN: Consulta Saldos dos Países
	state.Balances = fetchCompanyBalances()

	// 4. DADOS LOCAIS: Sensores e Logs (Lidos da Memória RAM do Manager)
	// Como os sensores se conectam via MQTT localmente, usamos as variáveis do Go.
	// Se você tiver as variáveis globais 'ConnectedSensors' e 'ActionLogs',
	// basta descomentar o bloco abaixo:

	/*
		sensorsList := make([]string, 0)
		ConnectedSensors.Range(func(key, value interface{}) bool {
			sensorsList = append(sensorsList, key.(string))
			return true
		})
		sort.Strings(sensorsList)
		state.Sensors = sensorsList

		// Adiciona os logs locais
		state.Logs = []string{"Sistema operando via Cosmos SDK. Consultando ledger..."}
	*/

	return state
}

// fetchCompanyBalances lê o arquivo paises.json e faz a consulta viva na blockchain
func fetchCompanyBalances() []CompanyBalance {
	var balances []CompanyBalance

	// Usa a função que já existe em vez de ler o arquivo manualmente
	if err := carregarEnderecosPaises(); err != nil {
		log.Printf("[WARN] fetchCompanyBalances: %v", err)
		return balances
	}

	nodeURL := os.Getenv("BLOCKCHAIN_REST_URLS")
	if nodeURL == "" {
		nodeURL = "http://localhost:1317"
	}
	nodeURL = strings.Split(nodeURL, ",")[0]

	for pais, address := range EnderecosPaises {
		url := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", nodeURL, address)
		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(url)
		saldoAtual := "FALHA API"
		if err == nil && resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			var data BalancesResponse
			json.Unmarshal(body, &data)
			saldoAtual = "0"
			for _, coin := range data.Balances {
				if coin.Denom == "stake" {
					saldoAtual = coin.Amount
					break
				}
			}
			resp.Body.Close()
		}
		balances = append(balances, CompanyBalance{
			Name:    strings.ToUpper(pais),
			Address: address,
			Balance: saldoAtual,
		})
	}

	sort.Slice(balances, func(i, j int) bool {
		return balances[i].Name < balances[j].Name
	})
	return balances
}
