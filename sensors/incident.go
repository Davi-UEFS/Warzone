package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/Davi-UEFS/Warzone/shared/functions"
)

func getEnviromentVariables() (string, string) {
	return os.Getenv("BROKER_IP"), os.Getenv("CLIENT_ID")
}

// Retorna incidente com chance de 10%
func generateIncident() bool {
	chance := rand.Float64()
	if chance <= 0.1 {
		return true
	}

	return false

}

func makeTopic() string {
	return fmt.Sprintf("sensors/%s", os.Getenv("CLIENT_ID"))
}

func main() {

	BROKER_IP, CLIENT_ID := getEnviromentVariables()
	TOPIC := makeTopic()

	client := functions.MakeClient(BROKER_IP, CLIENT_ID)

	var trigger bool
	trigger = false

	token := client.Publish(TOPIC, 1, false, fmt.Sprintf("DISPAROU: %v\n", trigger))
	token.Wait()

	for {
		if !trigger {
			trigger = generateIncident()
		} else {
			token := client.Publish(TOPIC, 1, false, fmt.Sprintf("DISPAROU: %v\n", trigger))
			token.Wait()
			trigger = false
			fmt.Println("Evento disparou. Comecando de novo")
		}

		time.Sleep(time.Second)

	}

}
