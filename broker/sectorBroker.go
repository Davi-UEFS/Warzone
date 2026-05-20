package main

/////////////////////////////////////////////////////////////////
// ESTE CODIGO NAO ESTA MAIS SENDO USADO.
// MANTIVE APENAS PARA POSSIBILIDADES FUTURAS
/////////////////////////////////////////////////////////////////

import (
	"bytes"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

type ConnectionLoggerHook struct {
	mqtt.HookBase
}

func (h *ConnectionLoggerHook) ID() string {
	return "connection-logger"
}

func (h *ConnectionLoggerHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
	}, []byte{b})
}

func (h *ConnectionLoggerHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	h.Log.Info("Dispositivo conectado",
		"client_id", cl.ID,
		"remote", cl.Net.Remote,
		"username", pk.Connect.Username,
	)
	return nil
}

func (h *ConnectionLoggerHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	if err != nil {
		h.Log.Info("Dispositivo desconectado",
			"client_id", cl.ID,
			"remote", cl.Net.Remote,
			"expire", expire,
			"error", err,
		)
		return
	}

	h.Log.Info("Dispositivo desconectado",
		"client_id", cl.ID,
		"remote", cl.Net.Remote,
		"expire", expire,
	)
}

func main() {
	ID := flag.String("id", "tcp1", "ID do Mochi MQTT")
	PORT := flag.Int("port", 1883, "Porta do MQTT")
	flag.Parse()

	server := mqtt.New(nil)

	_ = server.AddHook(new(auth.AllowHook), nil)
	_ = server.AddHook(new(ConnectionLoggerHook), nil)

	address := ":" + strconv.Itoa(*PORT)

	tcp := listeners.NewTCP(listeners.Config{
		ID:      *ID,
		Address: address,
	})
	err := server.AddListener(tcp)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Printf("MQTT Broker started on :%d", *PORT)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down broker...")
	server.Close()
}
