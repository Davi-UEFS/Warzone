package main

import (
	"sync"

	"github.com/Davi-UEFS/Warzone/shared"
)

// SectorMemory são os dados em memória RAM do Setor. É a antiga sectorFSM.
type SectorMemory struct {
	Mu               sync.Mutex
	DroneMap         map[string]*shared.Drone // Drones conhecidos
	PendingReqsQueue ReqHeap                  // Fila de prioridade de requisições pendentes
	Graveyard        map[string]int64         // ID do Drone -> Timestamp da morte
	DispatchedSet    map[string]int64         // ID da Requisição -> Timestamp da execução
}

// GlobalState é a nossa única fonte da verdade na memória RAM
var GlobalState = SectorMemory{
	DroneMap:      make(map[string]*shared.Drone),
	Graveyard:     make(map[string]int64),
	DispatchedSet: make(map[string]int64),
}

// IsGhost verifica se o drone morreu há menos de 30 segundos. Usado para encontrar drones fantasmas que ainda estão vivos na blockchain,
// mas que já foram mortos pelo Watchdog.
func (s *SectorMemory) IsGhost(droneID string, currentTime int64) bool {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if deathTime, exists := s.Graveyard[droneID]; exists {
		if currentTime-deathTime < 30 {
			return true // É um fantasma recente, ignore-o!
		}
		// Se já passou de 30s, a blockchain falhou. Tiramos do cemitério para tentar matar de novo.
		delete(s.Graveyard, droneID)
	}
	return false
}

// BuryDrone remove da RAM e adiciona ao cemitério.
func (s *SectorMemory) BuryDrone(droneID string, currentTime int64) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	delete(s.DroneMap, droneID)
	s.Graveyard[droneID] = currentTime
}
