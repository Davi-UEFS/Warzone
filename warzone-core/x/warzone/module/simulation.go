package warzone

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/Davi-UEFS/Warzone/warzone-core/testutil/sample"
	warzonesimulation "github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/simulation"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
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
		opWeightMsgAddReq          = "op_weight_msg_warzone"
		defaultWeightMsgAddReq int = 100
	)

	var weightMsgAddReq int
	simState.AppParams.GetOrGenerate(opWeightMsgAddReq, &weightMsgAddReq, nil,
		func(_ *rand.Rand) {
			weightMsgAddReq = defaultWeightMsgAddReq
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddReq,
		warzonesimulation.SimulateMsgAddReq(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgAssignDrone          = "op_weight_msg_warzone"
		defaultWeightMsgAssignDrone int = 100
	)

	var weightMsgAssignDrone int
	simState.AppParams.GetOrGenerate(opWeightMsgAssignDrone, &weightMsgAssignDrone, nil,
		func(_ *rand.Rand) {
			weightMsgAssignDrone = defaultWeightMsgAssignDrone
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAssignDrone,
		warzonesimulation.SimulateMsgAssignDrone(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgReportDeadDrone          = "op_weight_msg_warzone"
		defaultWeightMsgReportDeadDrone int = 100
	)

	var weightMsgReportDeadDrone int
	simState.AppParams.GetOrGenerate(opWeightMsgReportDeadDrone, &weightMsgReportDeadDrone, nil,
		func(_ *rand.Rand) {
			weightMsgReportDeadDrone = defaultWeightMsgReportDeadDrone
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgReportDeadDrone,
		warzonesimulation.SimulateMsgReportDeadDrone(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgRegDrone          = "op_weight_msg_warzone"
		defaultWeightMsgRegDrone int = 100
	)

	var weightMsgRegDrone int
	simState.AppParams.GetOrGenerate(opWeightMsgRegDrone, &weightMsgRegDrone, nil,
		func(_ *rand.Rand) {
			weightMsgRegDrone = defaultWeightMsgRegDrone
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRegDrone,
		warzonesimulation.SimulateMsgRegDrone(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgRmvReq          = "op_weight_msg_warzone"
		defaultWeightMsgRmvReq int = 100
	)

	var weightMsgRmvReq int
	simState.AppParams.GetOrGenerate(opWeightMsgRmvReq, &weightMsgRmvReq, nil,
		func(_ *rand.Rand) {
			weightMsgRmvReq = defaultWeightMsgRmvReq
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRmvReq,
		warzonesimulation.SimulateMsgRmvReq(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}
