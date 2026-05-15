package main

import (
	"log"
	"log/slog"
	"os"
	"strconv"

	// Importações do Mochi MQTT (Broker Embutido)
	mochimqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

// ==========================================
// 1. LÓGICA DO BROKER EMBUTIDO (MOCHI MQTT)
// ==========================================

// startEmbeddedBroker inicia o broker na mesma thread do Manager
func startEmbeddedBroker(port int) {
	quietLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Só vai imprimir se for WARN ou ERROR
	}))

	server := mochimqtt.New(&mochimqtt.Options{
		Logger: quietLogger,
	})

	_ = server.AddHook(new(auth.AllowHook), nil)

	address := ":" + strconv.Itoa(port)

	tcp := listeners.NewTCP(listeners.Config{
		ID:      "embedded-broker",
		Address: address,
	})

	err := server.AddListener(tcp)
	if err != nil {
		log.Fatalf("Erro ao configurar listener do broker embutido: %v", err)
	}

	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatalf("Erro ao iniciar broker embutido: %v", err)
		}
	}()

	log.Printf("Broker MQTT  iniciado na porta: %d", port)
}
