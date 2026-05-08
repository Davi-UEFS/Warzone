package main

import (
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/schollz/progressbar/v3"
)

func carryWater(command shared.DroneMission) {

	fmt.Println("Carregando água para o local do incidente...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(50 * time.Millisecond) // Simula o tempo de carregamento
	}
	fmt.Println("Incêndio prevenido!")
	payload, _ := makeResult(command)
	notifyDone(payload)

}

func drainOil(command shared.DroneMission) {

	fmt.Println("Drenando óleo...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(100 * time.Millisecond) // Simula o tempo de drenagem
	}
	fmt.Println("Vazamento contido!")
	payload, _ := makeResult(command)
	notifyDone(payload)
}
