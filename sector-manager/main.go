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
	fmt.Println("2 - Enviar comando de óleo")
	fmt.Println("3 - Sair")
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
			cmd := types.DroneCommand{
				AtomicID:  i,
				Action:    "water",
				Timestamp: time.Now(),
			}

			payload, _ := json.Marshal(cmd)

			sendCommand(client, payload)

		case "2":
			cmd := types.DroneCommand{
				AtomicID:  i,
				Action:    "oil",
				Timestamp: time.Now(),
			}

			payload, _ := json.Marshal(cmd)

			sendCommand(client, payload)
		case "3":
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

	mainMenu(client)

}
