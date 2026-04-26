package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func sendCommand(client mqtt.Client, payload []byte) {
	topic := "drones/drone01/commands"
	client.Publish(topic, 2, false, payload)
}
