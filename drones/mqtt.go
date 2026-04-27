package main

import (
	"fmt"
	"os"

	"github.com/Davi-UEFS/Warzone/shared/functions"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client = functions.MakeClient(
	os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID"))

var MISSION_TOPIC = fmt.Sprintf("drones/%s/missions", os.Getenv("CLIENT_ID"))
var MISSION_DONE_TOPIC = fmt.Sprintf("drones/%s/missions/done", os.Getenv("CLIENT_ID"))
var payloadChannel = make(chan []byte)

var commandHandler = func(client mqtt.Client, msg mqtt.Message) {
	payloadChannel <- msg.Payload()

}
