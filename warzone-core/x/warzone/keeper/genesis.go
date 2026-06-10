package keeper

import (
	"context"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	for _, elem := range genState.DroneMap {
		if err := k.Drone.Set(ctx, elem.DroneId, elem); err != nil {
			return err
		}
	}
	for _, elem := range genState.MissionList {
		if err := k.Mission.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
	}

	if err := k.MissionSeq.Set(ctx, genState.MissionCount); err != nil {
		return err
	}

	return k.Params.Set(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := k.Drone.Walk(ctx, nil, func(_ string, val types.Drone) (stop bool, err error) {
		genesis.DroneMap = append(genesis.DroneMap, val)
		return false, nil
	}); err != nil {
		return nil, err
	}
	err = k.Mission.Walk(ctx, nil, func(key uint64, elem types.Mission) (bool, error) {
		genesis.MissionList = append(genesis.MissionList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	genesis.MissionCount, err = k.MissionSeq.Peek(ctx)
	if err != nil {
		return nil, err
	}

	return genesis, nil
}
