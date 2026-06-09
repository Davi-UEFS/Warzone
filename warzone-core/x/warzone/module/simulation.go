package warzone

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/Davi-UEFS/warzone-core/testutil/sample"
	warzonesimulation "github.com/Davi-UEFS/warzone-core/x/warzone/simulation"
	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	warzoneGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		DroneMap: []types.Drone{{Creator: sample.AccAddress(),
			DroneId: "0",
		}, {Creator: sample.AccAddress(),
			DroneId: "1",
		}}, MissionList: []types.Mission{{Id: 0, Creator: sample.AccAddress()}, {Id: 1, Creator: sample.AccAddress()}}, MissionCount: 2,
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&warzoneGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)
	const (
		opWeightMsgCreateDrone          = "op_weight_msg_warzone"
		defaultWeightMsgCreateDrone int = 100
	)

	var weightMsgCreateDrone int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateDrone, &weightMsgCreateDrone, nil,
		func(_ *rand.Rand) {
			weightMsgCreateDrone = defaultWeightMsgCreateDrone
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateDrone,
		warzonesimulation.SimulateMsgCreateDrone(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgUpdateDrone          = "op_weight_msg_warzone"
		defaultWeightMsgUpdateDrone int = 100
	)

	var weightMsgUpdateDrone int
	simState.AppParams.GetOrGenerate(opWeightMsgUpdateDrone, &weightMsgUpdateDrone, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateDrone = defaultWeightMsgUpdateDrone
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUpdateDrone,
		warzonesimulation.SimulateMsgUpdateDrone(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgDeleteDrone          = "op_weight_msg_warzone"
		defaultWeightMsgDeleteDrone int = 100
	)

	var weightMsgDeleteDrone int
	simState.AppParams.GetOrGenerate(opWeightMsgDeleteDrone, &weightMsgDeleteDrone, nil,
		func(_ *rand.Rand) {
			weightMsgDeleteDrone = defaultWeightMsgDeleteDrone
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDeleteDrone,
		warzonesimulation.SimulateMsgDeleteDrone(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgCreateMission          = "op_weight_msg_warzone"
		defaultWeightMsgCreateMission int = 100
	)

	var weightMsgCreateMission int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateMission, &weightMsgCreateMission, nil,
		func(_ *rand.Rand) {
			weightMsgCreateMission = defaultWeightMsgCreateMission
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateMission,
		warzonesimulation.SimulateMsgCreateMission(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgUpdateMission          = "op_weight_msg_warzone"
		defaultWeightMsgUpdateMission int = 100
	)

	var weightMsgUpdateMission int
	simState.AppParams.GetOrGenerate(opWeightMsgUpdateMission, &weightMsgUpdateMission, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateMission = defaultWeightMsgUpdateMission
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUpdateMission,
		warzonesimulation.SimulateMsgUpdateMission(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgDeleteMission          = "op_weight_msg_warzone"
		defaultWeightMsgDeleteMission int = 100
	)

	var weightMsgDeleteMission int
	simState.AppParams.GetOrGenerate(opWeightMsgDeleteMission, &weightMsgDeleteMission, nil,
		func(_ *rand.Rand) {
			weightMsgDeleteMission = defaultWeightMsgDeleteMission
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDeleteMission,
		warzonesimulation.SimulateMsgDeleteMission(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgRequestDrone          = "op_weight_msg_warzone"
		defaultWeightMsgRequestDrone int = 100
	)

	var weightMsgRequestDrone int
	simState.AppParams.GetOrGenerate(opWeightMsgRequestDrone, &weightMsgRequestDrone, nil,
		func(_ *rand.Rand) {
			weightMsgRequestDrone = defaultWeightMsgRequestDrone
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRequestDrone,
		warzonesimulation.SimulateMsgRequestDrone(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgSubmitReport          = "op_weight_msg_warzone"
		defaultWeightMsgSubmitReport int = 100
	)

	var weightMsgSubmitReport int
	simState.AppParams.GetOrGenerate(opWeightMsgSubmitReport, &weightMsgSubmitReport, nil,
		func(_ *rand.Rand) {
			weightMsgSubmitReport = defaultWeightMsgSubmitReport
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubmitReport,
		warzonesimulation.SimulateMsgSubmitReport(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}
