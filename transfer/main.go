package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
)

func main() {
	// Definição das Flags de entrada
	broker := flag.String("broker", "tcp://localhost:1883", "Endereço do broker MQTT (ex: tcp://host:1883)")
	fromAlias := flag.String("from", "manager_setor_a", "Apelido do país/setor na keyring local (origem)")
	toAddr := flag.String("to", "", "Endereço bech32 de destino (cosmos1...)")
	amount := flag.String("amount", "100stake", "Quantidade e denominação (ex: 100stake, 50token)")
	flag.Parse()

	// Validação de segurança
	if *toAddr == "" {
		log.Fatal("\033[1;31m[ERRO]\033[0m O endereço de destino (-to) é obrigatório.")
	}

	BROKER_IP := *broker
	// Gera um ID de cliente único para evitar conflitos no broker
	CLIENT_ID := fmt.Sprintf("transfer-client-%d", time.Now().UnixNano())
	TOPIC := "finance/transfer"

	// Cria e conecta o cliente MQTT reutilizando o construtor do projeto
	client, err := shared.MakeClient(BROKER_IP, CLIENT_ID, nil, true)
	if err != nil {
		log.Fatalf("\033[1;31m[ERRO]\033[0m Falha ao conectar no broker MQTT: %v", err)
	}

	// Monta o payload JSON
	req := shared.TransferRequest{
		FromAlias: *fromAlias,
		ToAddress: *toAddr,
		Amount:    *amount,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("\033[1;31m[ERRO]\033[0m Erro ao serializar payload de transferência: %v", err)
	}

	fmt.Printf("\033[1;94m[CLIENTE]\033[0m Iniciando requisição de transferência via MQTT...\n")
	fmt.Printf(" -> Origem (Apelido): %s\n", req.FromAlias)
	fmt.Printf(" -> Destino: %s\n", req.ToAddress)
	fmt.Printf(" -> Valor: %s\n", req.Amount)

	// Publica a mensagem no tópico (QoS 1 garante entrega)
	token := client.Publish(TOPIC, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		log.Fatalf("\033[1;31m[ERRO]\033[0m Falha ao publicar transferência: %v", token.Error())
	}

	fmt.Println("\033[1;32m[SUCESSO]\033[0m Transferência enviada ao Setor Manager.")

	// Desconexão limpa
	time.Sleep(500 * time.Millisecond)
	client.Disconnect(250)
}
