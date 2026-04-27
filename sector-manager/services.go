package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func sendCommand(client mqtt.Client, payload []byte) {
	//TODO: HARDCODED
	topic := "drones/drone01/missions"
	client.Publish(topic, 2, false, payload)
}
