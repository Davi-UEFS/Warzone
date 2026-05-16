package app

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/schollz/progressbar/v3"
)

// runMissionProgress desenha uma barra de progresso para a missão atual.
func (app *DroneApp) runMissionProgress(taskName string, delayMs time.Duration) {
	bar := progressbar.NewOptions(100,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[yellow]█[reset]",
			SaucerHead:    "[yellow]█[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	for i := 0; i < 100; i++ {
		if i < 33 {
			bar.Describe(fmt.Sprintf("[red]%s...[reset]", taskName))
		} else if i < 66 {
			bar.Describe(fmt.Sprintf("[yellow]%s...[reset]", taskName))
		} else {
			bar.Describe(fmt.Sprintf("[green]%s...[reset]", taskName))
		}

		bar.Add(1)
		time.Sleep(delayMs * time.Millisecond)
	}
	fmt.Println()
}

func (app *DroneApp) CarryWater(command shared.DroneMission) {
	app.runMissionProgress("Carregando água", 100)
	app.drainBattery(2)
	fmt.Println("Incêndio prevenido!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) DrainOil(command shared.DroneMission) {
	app.runMissionProgress("Drenando óleo", 80)
	app.drainBattery(2)
	fmt.Println("Vazamento contido!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) RetrieveGoods(command shared.DroneMission) {
	app.runMissionProgress("Recuperando mantimentos", 60)
	app.drainBattery(2)
	fmt.Println("Mantimentos recuperados!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) IdentifyObject(command shared.DroneMission) {
	app.runMissionProgress("Identificando objeto suspeito", 80)
	app.drainBattery(1)

	randomObj := rand.Intn(3)
	switch randomObj {
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
	app.runMissionProgress("Inspecionando área", 70)
	app.drainBattery(1)
	fmt.Println("Inspeção concluída!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}

func (app *DroneApp) OptimizeRoute(command shared.DroneMission) {
	app.runMissionProgress("Otimizando rotas navais", 60)
	app.drainBattery(1)
	fmt.Println("Rota otimizada!")
	payload, _ := app.makeResult(command)
	app.notifyDone(payload)
}
