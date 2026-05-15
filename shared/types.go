package shared

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
)

type DroneStatus string

const (
	DRONE_IDLE   DroneStatus = "IDLE"
	DRONE_BUSY   DroneStatus = "BUSY"
	DRONE_ERROR  DroneStatus = "ERROR"
	DRONE_RETURN DroneStatus = "RETURNING"
)

const NONE = "NONE"

const (
	FIRE           = "THROW_WATER"
	OIL            = "DRAIN_OIL"
	WRECKAGE       = "RESCUE_GOODS"
	INSPECTION     = "INSPECT_AREA"
	UNKNOWN_OBJECT = "DETECT_UNKNOWN_OBJECT"
	BOTTLENECK     = "OPTIMIZE_TRAFFIC"
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
	BatteryLevel   int
	Status         DroneStatus
	CurrentSector  string
	CurrentBroker  string
	CurrentMission string
	LastSeen       int64
}

type DroneHeartbeat struct {
	ID           string `json:"id"`
	BatteryLevel int    `json:"battery_level"`
}

// Warzone/shared/types.go

// SetPhysicalLocation define onde o drone está fisicamente agora.
func (d *Drone) SetPhysicalLocation(sectorID string, brokerAddr string) {
	d.CurrentSector = sectorID
	d.CurrentBroker = brokerAddr
}

// SetBusy marca o drone como ocupado com uma missão específica.
func (d *Drone) SetBusy(missionID string) {
	d.Status = DRONE_BUSY
	d.CurrentMission = missionID
}

// SetIdle liberta o drone para novas tarefas.
func (d *Drone) SetIdle() {
	d.Status = DRONE_IDLE
	d.CurrentMission = NONE
}

// HasJurisdiction verifica se este manager local deve falar com este drone.
func (d *Drone) HasJurisdiction(localSectorID string) bool {
	return d.CurrentSector == localSectorID
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
	Type         string     `json:"type"`
	Coord        Coordinate `json:"coord"`
	OriginSector string     `json:"origin_sector"`
	LamportTime  int        `json:"lamport_time"`
	CreatedAt    int64      `json:"created_at"` // Unix timestamp for aging
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
