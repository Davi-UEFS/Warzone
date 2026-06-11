package main

import (
	"fmt"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
)

// FetchMissionsPENDING simulado para o laboratório (não precisa de blockchain rodando)
func FetchMissionsPENDING(blockchainURL string, targetSector string) ([]types.Mission, error) {
	fmt.Println("[MOCK] Simulando leitura de dados da Blockchain...")

	// Simulamos o JSON que a blockchain enviaria se estivesse ligada
	mockMissions := []types.Mission{
		{
			Id:              1,
			Creator:         "cosmos1xyz...",
			Sector:          targetSector,
			Status:          "PENDING",
			Priority:        2, // Alta
			AssignedDroneId: "",
			ReqType:         "THROW_WATER",
			Coord:           "12.34,-56.78",
			CreatedAt:       1717941234,
		},
		{
			Id:              2,
			Creator:         "cosmos1abc...",
			Sector:          targetSector,
			Status:          "PENDING",
			Priority:        1, // Baixa
			AssignedDroneId: "",
			ReqType:         "INSPECT_AREA",
			Coord:           "12.99,-56.11",
			CreatedAt:       1717941500,
		},
	}

	return mockMissions, nil
}

func FetchDronesIDLE(blockchainURL string, targetSector string) ([]types.Drone, error) {
	fmt.Println("[MOCK] Simulando leitura de dados da Blockchain...")

	// Simulamos o JSON que a blockchain enviaria se estivesse ligada
	mockDrones := []types.Drone{
		{
			DroneId: "drone-1",
			Sector:  targetSector,
			Status:  "IDLE",
			Battery: "85", //TODO: ajustar tipo de dado

		},
	}

	return mockDrones, nil
}

func sortRequisitionsWithAging(requisitions []types.Mission) {

	if len(requisitions) == 0 {
		return
	}

	//TODO: implementar ordenação por prioridade e tempo de espera (aging) para evitar starvation

	/*
		sort.Slice(requisitions, func(i, j int) bool {
			now := time.Now().Unix()

			// 1. Converter os timestamps da blockchain (string) para int64
			timeI, _ := requisitions[i].CreatedAt
			timeJ, _ :=

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
	*/
}

func newDispatch() {

	requisitions, err := FetchMissionsPENDING("http://temp-blockchain-url", "Sector-1")
	if err != nil {
		fmt.Println("Erro ao buscar missões:", err)
		return
	}

	// TODO: FAZER LOGICA DE DESPACHO
	print(requisitions)

}
