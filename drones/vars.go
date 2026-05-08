package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client, _ = shared.MakeClient(
	os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID"))

var LClock = shared.LamportClock{
	Time: 0,
	Mu:   sync.Mutex{},
}

var MISSION_TOPIC = fmt.Sprintf("drones/%s/missions", os.Getenv("CLIENT_ID"))
var MISSION_DONE_TOPIC = fmt.Sprintf("drones/%s/done", os.Getenv("CLIENT_ID"))
var payloadChannel = make(chan []byte, 4096)

var commandHandler = func(client mqtt.Client, msg mqtt.Message) {
	payloadChannel <- msg.Payload()

}
