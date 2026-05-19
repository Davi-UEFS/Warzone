package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	raft "github.com/hashicorp/raft"
)

type DashboardState struct {
	Pending     []shared.Requisition `json:"pending"`
	InProgress  []shared.Requisition `json:"in_progress"`
	Logs        []string             `json:"logs"`
	Drones      []DashboardDrone     `json:"drones"`
	Sensors     []string             `json:"sensors"`
	GeneratedAt int64                `json:"generated_at"`
	Sector      string               `json:"sector"`
	Leader      bool                 `json:"leader"`
	RaftState   string               `json:"raft_state"`
}

type DashboardDrone struct {
	ID       string             `json:"id"`
	Status   shared.DroneStatus `json:"status"`
	Mission  string             `json:"mission"`
	Battery  int                `json:"battery"`
	LastSeen int64              `json:"last_seen"`
	Broker   string             `json:"broker"`
	Sector   string             `json:"sector"`
}

//go:embed GUI/dashboard.html
var dashboardHTML []byte

func dashboardIndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.URL.Path != "/" && r.URL.Path != "/dashboard" {
		http.NotFound(w, r)
		return
	}

	// 3. REMOVA O http.ServeFile E USE O HTML EMBUTIDO!
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(dashboardHTML)
}

func startDashboardServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", dashboardStateHandler)
	mux.HandleFunc("/", dashboardIndexHandler)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("\033[1;94m[LOCAL]:\033[0m Dashboard disponível em http://localhost:%d\n", port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("\033[1;94m[LOCAL]:\033[0m Erro no servidor do dashboard: %v\n", err)
	}
}

func dashboardStateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	state := buildDashboardState()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(state)
}

func buildDashboardState() DashboardState {
	state := DashboardState{
		GeneratedAt: time.Now().Unix(),
	}

	if raftNode != nil {
		state.RaftState = fmt.Sprintf("%v", raftNode.State())
		state.Leader = raftNode.State() == raft.Leader
	}

	// --- NOVA LÓGICA DE SENSORES ---
	sensorsList := make([]string, 0)

	// Varre o mapa global em tempo real
	ConnectedSensors.Range(func(key, value interface{}) bool {
		sensorsList = append(sensorsList, key.(string))
		return true // Continua o loop
	})

	sort.Strings(sensorsList) // Mantém ordem alfabética no painel
	state.Sensors = sensorsList
	// -------------------------------

	if sectorFSM == nil {
		return state
	}

	sectorFSM.Mu.Lock()

	pending := sectorFSM.PendingReqsQueue.ToSlice()
	inProgress := make([]shared.Requisition, 0, len(sectorFSM.InProgressReqs))

	for _, req := range sectorFSM.InProgressReqs {
		inProgress = append(inProgress, req)
	}

	// (A velha lógica do activeSensorsMap foi APAGADA daqui, o código fica mais limpo)

	drones := make([]DashboardDrone, 0, len(sectorFSM.DroneMap))
	for _, drone := range sectorFSM.DroneMap {
		mission := drone.CurrentMission
		drones = append(drones, DashboardDrone{
			ID:       drone.ID,
			Status:   drone.Status,
			Mission:  mission,
			Battery:  drone.BatteryLevel,
			LastSeen: drone.LastSeen,
			Broker:   drone.CurrentBroker,
			Sector:   drone.CurrentSector,
		})
	}

	sector := sectorFSM.Sector
	sectorFSM.Mu.Unlock()

	sort.Slice(pending, func(i, j int) bool {
		if pending[i].Priority != pending[j].Priority {
			return pending[i].Priority > pending[j].Priority
		}
		return pending[i].LamportTime < pending[j].LamportTime
	})

	sort.Slice(inProgress, func(i, j int) bool {
		if inProgress[i].Priority != inProgress[j].Priority {
			return inProgress[i].Priority > inProgress[j].Priority
		}
		return inProgress[i].LamportTime < inProgress[j].LamportTime
	})

	sort.Slice(drones, func(i, j int) bool {
		return drones[i].ID < drones[j].ID
	})

	state.Pending = pending
	state.InProgress = inProgress
	state.Drones = drones
	state.Sector = sector
	logCpy := make([]string, len(sectorFSM.ActionLogs))
	copy(logCpy, sectorFSM.ActionLogs)
	state.Logs = logCpy

	return state
}
