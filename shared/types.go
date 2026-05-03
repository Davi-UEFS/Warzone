package shared

import (
	"sync"
	"time"
)

type DroneCommand struct {
	OccurrenceID string
	Action       string
	Timestamp    time.Time
}

type Drone struct {
	ID             string
	StationID      string
	BatteryLevel   int
	Status         string
	CurrentBroker  string
	CurrentMission string
}

func (drone *Drone) SetBusy() {
	drone.Status = "BUSY"
}

func (drone *Drone) SetIdle() {
	drone.Status = "IDLE"
}

func (drone *Drone) UpdateBroker(id string) {
	drone.CurrentBroker = id
}

type Requisition struct {
	OccurrenceID string `json:"occurrence_id"`
	DroneID      string `json:"drone_id"`
}

type Incident struct {
	ID          string     `json:"id"`
	Priority    int        `json:"priority"`
	Coord       Coordinate `json:"coord"`
	LamportTime int        `json:"lamport_time"`
}

type Coordinate struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type LamportClock struct {
	Time int
	Mu   sync.Mutex
}

func (lc *LamportClock) Tick() {
	lc.Mu.Lock()
	defer lc.Mu.Unlock()
	lc.Time++
}

func (lc *LamportClock) GetTime() int {
	lc.Mu.Lock()
	defer lc.Mu.Unlock()
	return lc.Time
}

func (lc *LamportClock) CompareAndUpdate(received int) {
	lc.Mu.Lock()
	if lc.Time < received {
		lc.Time = received
	}
	lc.Time++
	lc.Mu.Unlock()
}
