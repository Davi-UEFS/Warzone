package shared

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
)

const (
	DRONE_IDLE   = "IDLE"
	DRONE_BUSY   = "BUSY"
	DRONE_ERROR  = "ERROR"
	DRONE_RETURN = "RETURNING"
)

type DroneMission struct {
	RequisitionID string     `json:"id"`           // ID da Requisição (OccurrenceID)
	AssignedDrone string     `json:"drone_id"`     // ID do drone designado para a missão
	Type          string     `json:"type"`         // Tipo do incidente (Fogo, Óleo, etc.)
	Coordinate    Coordinate `json:"coordinate"`   // Onde o drone deve ir
	LamportTime   int        `json:"lamport_time"` // Tempo lógico da missão
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
	drone.Status = DRONE_BUSY
}

func (drone *Drone) SetIdle() {
	drone.Status = DRONE_IDLE
}

func (drone *Drone) UpdateBroker(id string) {
	drone.CurrentBroker = id
}

func (drone *Drone) AssignMission(missionID string) {
	drone.CurrentMission = missionID
}

func (drone *Drone) ClearMission() {
	drone.CurrentMission = ""
}

type DoneInfo struct {
	RequisitionID string `json:"occurrence_id"`
	DroneID       string `json:"drone_id"`
	LCTime        int    `json:"lc_time"`
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
	RequisitionID string `json:"incident_id"`
	LCTime        int    `json:"lc_time"`
}

type HeaderCommand struct {
	Operation   string          `json:"op"`
	Payload     json.RawMessage `json:"payload"`
	LamportTime int             `json:"lamport_time"`
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
