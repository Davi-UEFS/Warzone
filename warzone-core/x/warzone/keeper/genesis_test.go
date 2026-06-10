package keeper_test

import (
	"testing"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:   types.DefaultParams(),
		DroneMap: []types.Drone{{DroneId: "0"}, {DroneId: "1"}}, MissionList: []types.Mission{{Id: 0}, {Id: 1}},
		MissionCount: 2,
	}
	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
	require.EqualExportedValues(t, genesisState.DroneMap, got.DroneMap)
	require.EqualExportedValues(t, genesisState.MissionList, got.MissionList)
	require.Equal(t, genesisState.MissionCount, got.MissionCount)

}
