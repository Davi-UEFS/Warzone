package main

import (
	"fmt"
	"time"

	"github.com/schollz/progressbar/v3"
)

func carryWater() {

	fmt.Println("Carregando água para o local do incidente...")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(50 * time.Millisecond) // Simula o tempo de carregamento
	}
	fmt.Println("Incêndio prevenido!")

}

func drainOil() {

	fmt.Println("Drenando óleo. Vazamento contido!")
	bar := progressbar.Default(100)
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(100 * time.Millisecond) // Simula o tempo de drenagem
	}
}
