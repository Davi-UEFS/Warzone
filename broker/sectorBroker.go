// 1) The broker. We create an embedded MQTT server using mochi-mqtt,
// add a TCP listener on the standard MQTT port 1883, and start serving.
// This single process handles all message routing, QoS, retained messages,
// and Last Will & Testament.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

func main() {
	ID := flag.String("id", "tcp1", "ID do Mochi MQTT")
	PORT := flag.Int("port", 1883, "Porta do MQTT")
	flag.Parse()
	server := mqtt.New(nil)

	// Allow all connections (no auth for demo purposes)
	_ = server.AddHook(new(auth.AllowHook), nil)

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
