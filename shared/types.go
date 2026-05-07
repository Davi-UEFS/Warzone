package shared

import (
	"encoding/json"
	"io"
	"strings"
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

type CommandTemporary struct {
	OccurrenceID string `json:"occurrence_id"`
	DroneID      string `json:"drone_id"`
}

type Alert struct {
	SensorID    string     `json:"sensor_id"`
	Coordinate  Coordinate `json:"coordinate"`
	Type        string     `json:"type"`
	LamportTime int        `json:"lamport_time"`
}

type Requisition struct {
	ID           string     `json:"id"`
	Priority     int        `json:"priority"`
	Coord        Coordinate `json:"coord"`
	OriginSector string     `json:"origin_sector"`
	LamportTime  int        `json:"lamport_time"`
}

type SolvedInfo struct {
	IncidentID string `json:"incident_id"`
	LCTime     int    `json:"lc_time"`
}

type HeaderCommand struct {
	Operation string          `json:"op"`
	Payload   json.RawMessage `json:"payload"`
}

type Coordinate struct {
	Longitude int `json:"longitude"`
	Latitude  int `json:"latitude"`
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

type FilteredWriter struct {
	Output  io.Writer
	Filters []string
}

func (f *FilteredWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	for _, filter := range f.Filters {
		if strings.Contains(msg, filter) {
			return len(p), nil // descarta silenciosamente
		}
	}
	return f.Output.Write(p)
}
