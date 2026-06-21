package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		DroneMap: []Drone{}, MissionList: []Mission{}, LaudoMap: []Laudo{}}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	droneIndexMap := make(map[string]struct{})

	for _, elem := range gs.DroneMap {
		index := fmt.Sprint(elem.DroneId)
		if _, ok := droneIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for drone")
		}
		droneIndexMap[index] = struct{}{}
	}
	missionIdMap := make(map[uint64]bool)
	missionCount := gs.GetMissionCount()
	for _, elem := range gs.MissionList {
		if _, ok := missionIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for mission")
		}
		if elem.Id >= missionCount {
			return fmt.Errorf("mission id should be lower or equal than the last id")
		}
		missionIdMap[elem.Id] = true
	}
	laudoIndexMap := make(map[string]struct{})

	for _, elem := range gs.LaudoMap {
		index := fmt.Sprint(elem.RequisitionId)
		if _, ok := laudoIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for laudo")
		}
		laudoIndexMap[index] = struct{}{}
	}

	return gs.Params.Validate()
}
