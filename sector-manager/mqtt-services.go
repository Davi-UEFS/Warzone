package main

import (
	"encoding/json"
	"fmt"

	"github.com/Davi-UEFS/Warzone/shared"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/raft"
)

type joinReq struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

var onMissionDoneHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Missão concluída: %s\n", string(msg.Payload())) //TODO: EDITAR DEPOIS
	result := shared.Requisition{}

	if err := json.Unmarshal(msg.Payload(), &result); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	sensorID := shared.ExtractSensorID(result.OccurrenceID)

	token := client.Publish(fmt.Sprintf("sensors/%s/solved", sensorID), 1, false, []byte("DONE"))
	token.Wait()

}

var onIncidentHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Nova ocorrência: %s\n", string(msg.Payload()))

	incident := shared.Incident{}
	if err := json.Unmarshal(msg.Payload(), &incident); err != nil {
		fmt.Printf("Erro ao unmarshal payload: %v\n", err)
		return
	}

	cmd := shared.DroneCommand{
		OccurrenceID: incident.ID, //TODO: EDITAR DEPOIS
		Action:       "oil",
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		fmt.Printf("Erro ao marshal comando: %v\n", err)
		return
	}

	sendCommand(client, payload)

}

func sendCommand(client mqtt.Client, payload []byte) {
	//TODO: HARDCODED
	topic := "drones/drone01/missions"
	client.Publish(topic, 2, false, payload)
}

func joinLeaderViaMQTT(brokerAddr, nodeID string, raftAddr string) error {
	client, err := shared.MakeClient(brokerAddr, "temporary_node")

	if err != nil {
		return err
	}

	defer client.Disconnect(250)

	req := joinReq{
		ID:   nodeID,
		Addr: raftAddr,
	}

	payload, err := json.Marshal(req)

	if err != nil {
		return err
	}

	token := client.Publish("/clusters/join", 2, false, payload)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("erro ao publicar join: %v", token.Error())
	}

	fmt.Printf("✔ Pedido de join enviado ao broker %s\n", brokerAddr)
	return nil

}

func MQTTJoinHandler(raftNode *raft.Raft, client mqtt.Client) {
	client.Subscribe("/clusters/join", 2, func(client mqtt.Client, msg mqtt.Message) {
		// 1. Só o líder processa mudanças na topologia
		if raftNode.State() != raft.Leader {
			return
		}

		var req joinReq
		if err := json.Unmarshal(msg.Payload(), &req); err != nil {
			fmt.Printf("Erro ao desserializar join: %v\n", err)
			return
		}

		fmt.Printf("→ Processando pedido de join: Nó %s em %s\n", req.ID, req.Addr)

		// 2. AddVoter é assíncrono, você PRECISA checar o futuro
		future := raftNode.AddVoter(raft.ServerID(req.ID), raft.ServerAddress(req.Addr), 0, 0)
		if err := future.Error(); err != nil {
			fmt.Printf("Falha ao adicionar nó %s ao consenso: %v\n", req.ID, err)
			return
		}

		fmt.Printf("Nó %s integrado com sucesso!\n", req.ID)
	})
}
