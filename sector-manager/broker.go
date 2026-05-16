package main

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	// Importações do Mochi MQTT (Broker Embutido)
	mochimqtt "github.com/mochi-mqtt/server/v2"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

// ==========================================
// 1. LÓGICA DO BROKER EMBUTIDO (MOCHI MQTT)
// ==========================================

// 1. Mapa Global Thread-Safe para guardar os sensores que estão fisicamente conectados
var ConnectedSensors sync.Map

// 2. Estrutura do nosso Intercetor (Hook) Customizado
type SensorTrackerHook struct {
	mqtt.HookBase
}

func (h *SensorTrackerHook) ID() string {
	return "sensor-tracker"
}

func (h *SensorTrackerHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
	}, []byte{b})
}

// Quando um cliente termina o handshake MQTT com sucesso
func (h *SensorTrackerHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	// Assumimos que o ID do cliente MQTT dos sensores começa com "sensor" (ex: "sensor-01")
	if strings.HasPrefix(cl.ID, "sensor") {
		ConnectedSensors.Store(cl.ID, true)
		fmt.Printf("📡 Sensor Local Conectado: %s\n", cl.ID)
	}
	return nil
}

// Quando a conexão TCP cai ou o cliente envia um pacote DISCONNECT
func (h *SensorTrackerHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	if strings.HasPrefix(cl.ID, "sensor") {
		ConnectedSensors.Delete(cl.ID)
		fmt.Printf("🔌 Sensor Local Desconectado: %s\n", cl.ID)
	}
}

// startEmbeddedBroker inicia o broker na mesma thread do Manager
func startEmbeddedBroker(port int) {
	quietLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Só vai imprimir se for WARN ou ERROR
	}))

	server := mochimqtt.New(&mochimqtt.Options{
		Logger: quietLogger,
	})

	_ = server.AddHook(new(auth.AllowHook), nil)
	_ = server.AddHook(new(SensorTrackerHook), nil)

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
