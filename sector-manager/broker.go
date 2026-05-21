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

// Hook personalizado utilizado para identificar quando um sensor se conecta ao MQTT.
// O hook implementa os métodos ID(), Provides(), OnConnect() e OnDisconnect() conforme 
// exigido pela interface de hooks do MochiMQTT.
type SensorTrackerHook struct {
	mqtt.HookBase
}

// Define o ID para o Hook
func (h *SensorTrackerHook) ID() string {
	return "sensor-tracker"
}

// Define os eventos no qual o Hook se interessa.
// Neste caso, são o OnConnect e onDisconnect
func (h *SensorTrackerHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
	}, []byte{b})
}

// Quando um cliente termina o handshake MQTT com sucesso, verifica se seu ID contém "sensor".
// Se sim, guarda no mapa e imprime na tela.
func (h *SensorTrackerHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	if strings.HasPrefix(cl.ID, "sensor") {
		ConnectedSensors.Store(cl.ID, true)
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Sensor Local Conectado: %s\n", cl.ID)
	}
	return nil
}

// Quando a conexão TCP cai ou o cliente envia um pacote DISCONNECT, verifica se seu ID contém "sensor".
// Se sim, guarda no mapa e imprime na tela.
func (h *SensorTrackerHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	if strings.HasPrefix(cl.ID, "sensor") {
		ConnectedSensors.Delete(cl.ID)
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Sensor Local Desconectado: %s\n", cl.ID)
	}
}

// startEmbeddedBroker inicia o broker embutido do setor.
// Params:
// port: A porta utilizada pelo broker MQTT
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

	log.Printf("\033[1;94m[LOCAL]:\033[0m Broker MQTT  iniciado na porta: %d", port)
}
