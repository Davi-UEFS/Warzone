package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/schollz/progressbar/v3"
)

func (app *DroneApp) CarryWater(command shared.DroneMission) {
	fmt.Println("Carregando água para o local do incidente...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(100 * time.Millisecond)
	}
	app.drainBattery(2)
	fmt.Println("Incêndio prevenido!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) DrainOil(command shared.DroneMission) {
	fmt.Println("Drenando óleo...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(80 * time.Millisecond)
	}
	app.drainBattery(2)
	fmt.Println("Vazamento contido!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) RetrieveGoods(command shared.DroneMission) {
	fmt.Println("Recuperando mantimentos do naufrágio...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(60 * time.Millisecond)
	}
	app.drainBattery(2)
	fmt.Println("Mantimentos recuperados!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) IdentifyObject(command shared.DroneMission) {
	fmt.Println("Identificando objeto suspeito...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(80 * time.Millisecond)
	}
	app.drainBattery(1)

	rand := rand.Intn(3)

	switch rand {
	case 0:
		fmt.Println("Objeto identificado como embarcação!")
	case 1:
		fmt.Println("Objeto identificado como cardume de peixes!")
	case 2:
		fmt.Println("Objeto identificado como submarino!")
	}

	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) PerformInspection(command shared.DroneMission) {
	fmt.Println("Inspecionando área...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(70 * time.Millisecond)
	}
	app.drainBattery(1)
	fmt.Println("Inspeção concluída!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) OptimizeRoute(command shared.DroneMission) {
	fmt.Println("Organizando navios para tirar engarrafamento...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(60 * time.Millisecond)
	}

	app.drainBattery(1)

	fmt.Println("Rota otimizada!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)

}
