package types_test

import (
	"testing"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc:     "valid genesis state",
			genState: &types.GenesisState{DroneMap: []types.Drone{{DroneId: "0"}, {DroneId: "1"}}, MissionList: []types.Mission{{Id: 0}, {Id: 1}}, MissionCount: 2}, valid: true,
		}, {
			desc: "duplicated drone",
			genState: &types.GenesisState{
				DroneMap: []types.Drone{
					{
						DroneId: "0",
					},
					{
						DroneId: "0",
					},
				},
				MissionList: []types.Mission{{Id: 0}, {Id: 1}}, MissionCount: 2,
			}, valid: false,
		}, {
			desc: "duplicated mission",
			genState: &types.GenesisState{
				MissionList: []types.Mission{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		}, {
			desc: "invalid mission count",
			genState: &types.GenesisState{
				MissionList: []types.Mission{
					{
						Id: 1,
					},
				},
				MissionCount: 0,
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
