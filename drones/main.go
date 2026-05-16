package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	droneapp "github.com/Davi-UEFS/Warzone/drones/app"
)

func main() {
	idFlag := flag.String("id", "drone-01", "ID do drone")
	brokersFlag := flag.String("brokers", "tcp://localhost:1883,tcp://localhost:1884", "Lista de brokers separados por vírgula")
	flag.Parse()

	droneID := *idFlag
	brokers := strings.Split(*brokersFlag, ",")

	app := droneapp.NewDroneApp(droneID, brokers)
	go app.Run()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("\nEncerrando atividades...")
}
