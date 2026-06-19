// Package shared concentra os tipos usados por mais de um módulo do projeto.
package shared

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// DroneStatus representa o estado atual de um drone.
type DroneStatus string

// Estados possíveis para o drone. DRONE_ERROR e DRONE_RETURN ainda não são usados, mas estão previstos para futuras implementações.
const (
	DRONE_IDLE   DroneStatus = "IDLE"
	DRONE_BUSY   DroneStatus = "BUSY"
	DRONE_ERROR  DroneStatus = "ERROR"
	DRONE_RETURN DroneStatus = "RETURNING"
)

// Status de missão para as requisições.
const NONE = "NONE"
const (
	PENDING     = "PENDING"
	IN_PROGRESS = "IN_PROGRESS"
	DONE        = "DONE"
)

// Tipos de missão usados pelo sistema.
const (
	FIRE           = "THROW_WATER"
	OIL            = "DRAIN_OIL"
	WRECKAGE       = "RESCUE_GOODS"
	INSPECTION     = "INSPECT_AREA"
	UNKNOWN_OBJECT = "DETECT_UNKNOWN_OBJECT"
	BOTTLENECK     = "OPTIMIZE_TRAFFIC"
)

// No seu pacote shared
type TransferRequest struct {
	FromAlias string `json:"from"`
	ToAddress string `json:"to"`
	Amount    string `json:"amount"` // ex: "1000stake"
}

// DroneMission guarda os dados de uma missão atribuída a um drone.
type DroneMission struct {
	RequisitionID string     `json:"id"`           // ID da Requisição (OccurrenceID)
	AssignedDrone string     `json:"drone_id"`     // ID do drone designado para a missão
	Type          string     `json:"type"`         // Tipo do incidente (Fogo, Óleo, etc.)
	Coordinate    Coordinate `json:"coordinate"`   // Onde o drone deve ir
	LamportTime   int        `json:"lamport_time"` // Tempo lógico da missão
}

// Drone representa um drone registrado no sistema.
type Drone struct {
	ID             string
	BatteryLevel   int
	Status         DroneStatus
	CurrentSector  string
	CurrentBroker  string
	CurrentMission string
	LastSeen       int64
	Verified       bool
}

// DroneHeartbeat é a mensagem curta enviada pelo drone para indicar vida.
type DroneHeartbeat struct {
	ID           string `json:"id"`
	BatteryLevel int    `json:"battery_level"`
}

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

// DoneInfo representa a confirmação de conclusão de uma missão.
type DoneInfo struct {
	RequisitionID string `json:"occurrence_id"`
	DroneID       string `json:"drone_id"`
	LCTime        int    `json:"lc_time"`
}

// RegErrorMessage é usada para informar falhas no registro de drones.
type RegErrorMessage struct {
	DroneID string `json:"drone_id"`
	Error   string `json:"error"`
}

// Alert representa um alerta gerado por um sensor.
type Alert struct {
	ID          string     `json:"id"`
	SensorID    string     `json:"sensor_id"`
	Coordinate  Coordinate `json:"coordinate"`
	Type        string     `json:"type"`
	LamportTime int        `json:"lamport_time"`
	Country     string     `json:"country"`
}

// Requisition representa uma requisição gerada a partir de um alerta.
type Requisition struct {
	ID           string     `json:"id"`
	Priority     int        `json:"priority"`
	Type         string     `json:"type"`
	Coord        Coordinate `json:"coord"`
	OriginSector string     `json:"origin_sector"`
	LamportTime  int        `json:"lamport_time"`
	CreatedAt    int64      `json:"created_at"`
}

// HeaderCommand padroniza mensagens que carregam operação, payload e tempo lógico.
type HeaderCommand struct {
	Operation   string          `json:"op"`
	Payload     json.RawMessage `json:"payload"`
	LamportTime int             `json:"lamport_time"`
}

// Coordinate guarda uma posição geográfica simplificada.
type Coordinate struct {
	Longitude int `json:"longitude"`
	Latitude  int `json:"latitude"`
}

// LamportClock implementa um relógio lógico simples com proteção por mutex.
type LamportClock struct {
	Time  int
	Mu    sync.Mutex
	Debug bool
}

// Tick avança o relógio lógico em 1.
func (lc *LamportClock) Tick() {
	lc.Mu.Lock()
	defer lc.Mu.Unlock()
	lc.Time++
}

// GetTime retorna o valor atual do relógio lógico.
func (lc *LamportClock) GetTime() int {
	lc.Mu.Lock()
	defer lc.Mu.Unlock()
	return lc.Time
}

// CompareAndUpdate compara o tempo lógico recebido com o local e atualiza se estiver atrasado.
//
// Params:
//   - received: o tempo lógico recebido de outra entidade.
func (lc *LamportClock) CompareAndUpdate(received int) {
	lc.Mu.Lock()
	oldTime := lc.Time
	if lc.Time < received {
		lc.Time = received
	}
	lc.Time++

	if lc.Debug && (oldTime != lc.Time-1 || received > oldTime) {
		fmt.Printf("\n\033[1;36m[DEBUG-LAMPORT]\033[0m Sincronizacao: Local(%d) | Recebido(%d) -> Novo(%d)\n", oldTime, received, lc.Time)
	}

	lc.Mu.Unlock()
}

// FilteredWriter descarta mensagens que contenham textos filtrados.
type FilteredWriter struct {
	Output  io.Writer
	Filters []string
}

// Write implementa a interface io.Writer, filtrando mensagens que contenham os textos especificados em Filters.
func (f *FilteredWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	for _, filter := range f.Filters {
		if strings.Contains(msg, filter) {
			return len(p), nil // descarta a mensagem sem escrever. "Engana" o escritor retornando que escreveu tudo, mesmo não escrevendo nada.
		}
	}
	return f.Output.Write(p)
}
