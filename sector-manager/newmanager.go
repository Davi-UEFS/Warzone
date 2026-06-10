package main

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
)

// FetchMissionsPENDING simulado para o laboratório (não precisa de blockchain rodando)
func FetchMissionsPENDING(blockchainURL string, targetSector string) ([]types.Mission, error) {
	fmt.Println("[MOCK] Simulando leitura de dados da Blockchain...")

	// Simulamos o JSON que a blockchain enviaria se estivesse ligada
	mockMissions := []types.Mission{
		{
			Id:              "1",
			Creator:         "cosmos1xyz...",
			Sector:          targetSector,
			Status:          "PENDING",
			Priority:        "2", // Alta
			AssignedDroneId: "",
			ReqType:         "THROW_WATER",
			Coord:           "12.34,-56.78",
			CreatedAt:       "1717941234",
		},
		{
			Id:              "2",
			Creator:         "cosmos1abc...",
			Sector:          targetSector,
			Status:          "PENDING",
			Priority:        "1", // Baixa
			AssignedDroneId: "",
			ReqType:         "INSPECT_AREA",
			Coord:           "12.99,-56.11",
			CreatedAt:       "1717941500",
		},
	}

	return mockMissions, nil
}

func FetchDronesIDLE(blockchainURL string, targetSector string) ([]types.Drone, error) {
	fmt.Println("[MOCK] Simulando leitura de dados da Blockchain...")

	// Simulamos o JSON que a blockchain enviaria se estivesse ligada
	mockDrones := []types.Drone{
		{
			Id:        "drone-1",
			Sector:    targetSector,
			Status:    "IDLE",
			Coord:     "12.30,-56.70",
			Battery:   85,
			CreatedAt: "1717940000",
		},
		{
			Id:        "drone-2",
			Sector:    targetSector,
			Status:    "IDLE",
			Coord:     "12.50,-56.90",
			Battery:   60,
			CreatedAt: "1717940500",
		},
	}

	return mockDrones, nil
}

func sortRequisitionsWithAging(requisitions []types.Mission) {

	if len(requisitions) == 0 {
		return
	}

	sort.Slice(requisitions, func(i, j int) bool {
		now := time.Now().Unix()

		// 1. Converter os timestamps da blockchain (string) para int64
		timeI, _ := strconv.ParseInt(requisitions[i].CreatedAt, 10, 64)
		timeJ, _ := strconv.ParseInt(requisitions[j].CreatedAt, 10, 64)

		waitTimeI := now - timeI
		PrioBoostI := waitTimeI / 10
		scoreI := PrioBoostI + waitTimeI

		waitTimeJ := now - timeJ
		PrioBoostJ := waitTimeJ / 10
		scoreJ := PrioBoostJ + waitTimeJ

		// Ordena pelo maior score final
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		// Desempate pelo ID menor
		return requisitions[i].Id < requisitions[j].Id
	})

}

func newDispatch() {

	requisitions, err := FetchMissionsPENDING("http://temp-blockchain-url", "Sector-1")
	if err != nil {
		fmt.Println("Erro ao buscar missões:", err)
		return
	}

	nextReq := requisitions[0]

	freeDrones, err := FetchDronesIDLE("http://temp-blockchain-url", "Sector-1")
	if err != nil {
		fmt.Println("Erro ao buscar drones:", err)
		return
	}

}
