package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Estruturas para ler a resposta da REST API do Cosmos
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type BalancesResponse struct {
	Balances []Coin `json:"balances"`
}

func main() {
	// 1. Localiza o ficheiro paises.json gerado pelo prepara_rede.sh
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("\033[1;31m[ERRO]\033[0m Falha ao obter diretório home: %v", err)
	}
	paisesPath := filepath.Join(homeDir, "warzone-data", "setorA", "paises.json")

	// 2. Lê o ficheiro JSON
	file, err := os.ReadFile(paisesPath)
	if err != nil {
		log.Fatalf("\033[1;31m[ERRO]\033[0m Ficheiro %s não encontrado. Já correu o make prepare?\nDetalhes: %v", paisesPath, err)
	}

	// 3. Converte o JSON num mapa (Dicionário de País -> Endereço)
	var paises map[string]string
	if err := json.Unmarshal(file, &paises); err != nil {
		log.Fatalf("\033[1;31m[ERRO]\033[0m Erro ao ler o formato do JSON: %v", err)
	}

	nodeURL := "http://localhost:1317"

	// Desenha o cabeçalho da tabela
	fmt.Println("\n\033[1;34m=================================================================================\033[0m")
	fmt.Printf("\033[1;32m %-18s | %-45s | %-12s\033[0m\n", "PAÍS (ALVO)", "ENDEREÇO PÚBLICO (BECH32)", "SALDO")
	fmt.Println("\033[1;34m=================================================================================\033[0m")

	// 4. Itera sobre cada país e consulta o saldo na blockchain
	for pais, address := range paises {
		url := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", nodeURL, address)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf(" %-18s | %-45s | \033[1;31mFALHA DE LIGAÇÃO\033[0m\n", strings.ToUpper(pais), address)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf(" %-18s | %-45s | \033[1;31mERRO NA API\033[0m\n", strings.ToUpper(pais), address)
			continue
		}

		var data BalancesResponse
		json.Unmarshal(body, &data)

		// Extrai a quantidade de 'stake'
		saldo := "0"
		for _, coin := range data.Balances {
			if coin.Denom == "stake" {
				saldo = coin.Amount
				break
			}
		}

		// Imprime a linha da tabela
		fmt.Printf(" %-18s | %-45s | \033[1;93m%s stake\033[0m\n", strings.ToUpper(pais), address, saldo)
	}
	fmt.Println("\033[1;34m=================================================================================\033[0m")
	fmt.Println("\033[1;90m * Fonte dos dados: Ledger Imutável (Node A REST API)\033[0m\n")
}
