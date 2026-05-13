package main

import "fmt"

type FSMEvent struct {
	Topic    string
	QoS      byte
	Retained bool
	Payload  []byte
}

type eventPublisher interface {
	Publish(topic string, qos byte, retained bool, payload []byte) error
}

func (fsm *RaftFSM) emitEvent(event FSMEvent) {
	if fsm.EventSink == nil {
		return
	}

	select {
	case fsm.EventSink <- event:
	default:
		fmt.Printf("Fila de eventos cheia. Evento descartado: %s\n", event.Topic)
	}
}

func startFSMEventPublisher(events <-chan FSMEvent, publisher eventPublisher) {
	go func() {
		for event := range events {
			if err := publisher.Publish(event.Topic, event.QoS, event.Retained, event.Payload); err != nil {
				fmt.Printf("Falha ao publicar evento MQTT (%s): %v\n", event.Topic, err)
			}
		}
	}()
}
