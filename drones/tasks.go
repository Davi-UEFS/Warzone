package main

import (
	"fmt"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/schollz/progressbar/v3"
)

func (app *DroneApp) CarryWater(command shared.DroneMission) {
	fmt.Println("Carregando água para o local do incidente...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println("Incêndio prevenido!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) DrainOil(command shared.DroneMission) {
	fmt.Println("Drenando óleo...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("Vazamento contido!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}
