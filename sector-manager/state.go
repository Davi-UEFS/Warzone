package main

import (
	"sync"

	"github.com/Davi-UEFS/Warzone/shared"
)

// SectorMemory centraliza o estado da RAM para evitar corridas de concorrência (Race Conditions)
type SectorMemory struct {
	Mu               sync.Mutex
	DroneMap         map[string]*shared.Drone
	PendingReqsQueue ReqHeap          // Certifique-se de que o nome bate com o seu Heap
	Graveyard        map[string]int64 // ID do Drone -> Timestamp da morte
}

// GlobalState é a nossa única fonte da verdade na memória RAM
var GlobalState = SectorMemory{
	DroneMap:  make(map[string]*shared.Drone),
	Graveyard: make(map[string]int64),
	// A PendingReqsQueue geralmente é inicializada vazia automaticamente pelo Go,
	// mas se o seu Heap exigir um Init() ou make(), coloque aqui ou no main.
}

// --- Métodos Auxiliares de Segurança ---

// IsGhost verifica se o drone morreu há menos de 30 segundos
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

// BuryDrone remove da RAM e adiciona ao cemitério de forma segura
func (s *SectorMemory) BuryDrone(droneID string, currentTime int64) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	delete(s.DroneMap, droneID)
	s.Graveyard[droneID] = currentTime
}
