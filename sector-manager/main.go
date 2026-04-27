package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Davi-UEFS/Warzone/shared/functions"
	"github.com/Davi-UEFS/Warzone/shared/types"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func showMenu() {
	fmt.Println("\n=== Sector Manager ===")
	fmt.Println("1 - Enviar comando de água")
	fmt.Println("2 - Enviar comando para drone")
	fmt.Println("3 - Marcar ocorrência como resolvida")
	fmt.Println("4 - Sair")
	fmt.Print("Escolha: ")
}

func mainMenu(client mqtt.Client) {
	scanner := bufio.NewScanner(os.Stdin)
	var i int = 1

	for {
		showMenu()
		scanner.Scan()
		option := scanner.Text()

		switch option {
		case "1":
			fmt.Println("EM BREVE...")

		case "2":
			cmd := types.DroneCommand{
				OccurrenceID: fmt.Sprintf("cmd-%d", i),
				Action:       "oil",
				Timestamp:    time.Now(),
			}
			payload, _ := json.Marshal(cmd)
			sendCommand(client, payload)

		case "3":
			fmt.Print("ID do sensor: ")
			scanner.Scan()
			sensorID := scanner.Text()

			payload, _ := json.Marshal("DONE")

			topic := fmt.Sprintf("sensors/%s/solved", sensorID)
			token := client.Publish(topic, 2, false, payload)
			token.Wait()
			if token.Error() != nil {
				fmt.Println("Erro ao publicar:", token.Error())
			} else {
				fmt.Printf("→ Ocorrência marcada como resolvida no sensor %s\n", sensorID)
			}

		case "4":
			fmt.Println("Saindo...")
			client.Disconnect(250)
			return

		default:
			fmt.Println("Opção inválida")
		}

		i++
	}
}

func main() {
	client := functions.MakeClient(
		os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID"))

	client.Subscribe("drones/+/missions/done", 1, onMissionDoneHandler)
	client.Subscribe("sensors/+/incidents", 1, onIncidentHandler)
	mainMenu(client)
}
