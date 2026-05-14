package main

import (
	"bytes"
	"log"
	"strconv"

	// Importações do Mochi MQTT (Broker Embutido)
	mochimqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

// ==========================================
// 1. LÓGICA DO BROKER EMBUTIDO (MOCHI MQTT)
// ==========================================

type ConnectionLoggerHook struct {
	mochimqtt.HookBase
}

func (h *ConnectionLoggerHook) ID() string {
	return "connection-logger"
}

func (h *ConnectionLoggerHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mochimqtt.OnConnect,
		mochimqtt.OnDisconnect,
	}, []byte{b})
}

func (h *ConnectionLoggerHook) OnConnect(cl *mochimqtt.Client, pk packets.Packet) error {
	h.Log.Info("Dispositivo conectado ao Broker",
		"client_id", cl.ID,
		"remote", cl.Net.Remote,
	)
	return nil
}

func (h *ConnectionLoggerHook) OnDisconnect(cl *mochimqtt.Client, err error, expire bool) {
	h.Log.Info("Dispositivo desconectado do Broker",
		"client_id", cl.ID,
		"remote", cl.Net.Remote,
	)
}

// startEmbeddedBroker inicia o broker na mesma thread do Manager
func startEmbeddedBroker(port int) {
	server := mochimqtt.New(nil)

	_ = server.AddHook(new(auth.AllowHook), nil)
	_ = server.AddHook(new(ConnectionLoggerHook), nil)

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
