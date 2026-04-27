package main

import (
	"context"

	"github.com/Davi-UEFS/Warzone/shared/functions"
)

func main() {
	// ESCUTA AS REQUISICOES
	client.Subscribe(MISSION_TOPIC, 2, commandHandler)

	ctx := context.Background()
	go handleAction(ctx)

	functions.WaitForShutdown(client)

}
