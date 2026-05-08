package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	raft "github.com/hashicorp/raft"
)

func startDispatcher(raftNode *raft.Raft) {
	ticker := time.NewTicker(2 * time.Second)

	for range ticker.C {
		if raftNode.State() == raft.Leader {
			processRequisitions(raftNode)
		}

	}
}

func processRequisitions(raftNode *raft.Raft) {

	sectorFSM.Mu.Lock()

	if len(sectorFSM.PendingReqsQueue) == 0 {
		sectorFSM.Mu.Unlock()
		return
	}

	var freeDroneID string

	for id, drone := range sectorFSM.DroneMap {
		if drone.Status == shared.DRONE_IDLE {
			freeDroneID = id
			break
		}
	}

	if freeDroneID != "" {
		req := sectorFSM.PendingReqsQueue[0]
		sectorFSM.Mu.Unlock()

		dispatch(raftNode, freeDroneID, req)

	} else {
		sectorFSM.Mu.Unlock()
	}

}

func dispatch(raftNode *raft.Raft, droneID string, req shared.Requisition) {

	LClock.Tick()

	mission := shared.DroneMission{
		RequisitionID: req.ID,
		Type:          "oil", //TODO: Definir tipo com base na requisição
		Coordinate:    req.Coord,
		LamportTime:   LClock.GetTime(),
	}

	payload, _ := json.Marshal(mission)

	cmd := shared.HeaderCommand{
		Operation:   OP_ASSIGN,
		Payload:     payload,
		LamportTime: LClock.GetTime(),
	}

	cmdBytes, _ := json.Marshal(cmd)

	future := raftNode.Apply(cmdBytes, 5*time.Second)

	if err := future.Error(); err != nil {
		fmt.Printf("Erro ao aplicar comando no Raft: %v\n", err)
	} else {
		fmt.Printf("Requisição %s atribuída ao drone %s\n", req.ID, droneID)
	}

}

/*
var onMissionDoneHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Missão concluída: %s\n", string(msg.Payload()))
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
		OccurrenceID: incident.ID,
		Action:       "oil",
		Timestamp:    incident.Timestamp,
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		fmt.Printf("Erro ao marshal comando: %v\n", err)
		return
	}

	sendCommand(client, payload)

}
*/
