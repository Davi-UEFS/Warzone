package main

import (
	"context"
	"os"

	"github.com/Davi-UEFS/Warzone/shared/functions"
)

func main() {
	client := functions.MakeClient(
		os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID"))

	client.Subscribe(MISSION_TOPIC, 2, commandHandler)

	ctx := context.Background()
	go handleAction(ctx)

	functions.WaitForShutdown(client)

}
