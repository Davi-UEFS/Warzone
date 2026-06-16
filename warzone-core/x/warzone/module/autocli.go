package warzone

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "ListDrone",
					Use:       "list-drone",
					Short:     "List all drone",
				},
				{
					RpcMethod:      "GetDrone",
					Use:            "get-drone [id]",
					Short:          "Gets a drone",
					Alias:          []string{"show-drone"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "drone_id"}},
				},
				{
					RpcMethod: "ListMission",
					Use:       "list-mission",
					Short:     "List all mission",
				},
				{
					RpcMethod:      "GetMission",
					Use:            "get-mission [id]",
					Short:          "Gets a mission by id",
					Alias:          []string{"show-mission"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "CreateDrone",
					Use:            "create-drone [drone_id] [status] [sector] [battery]",
					Short:          "Create a new drone",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "drone_id"}, {ProtoField: "status"}, {ProtoField: "sector"}, {ProtoField: "battery"}},
				},
				{
					RpcMethod:      "UpdateDrone",
					Use:            "update-drone [drone_id] [status] [sector] [battery]",
					Short:          "Update drone",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "drone_id"}, {ProtoField: "status"}, {ProtoField: "sector"}, {ProtoField: "battery"}},
				},
				{
					RpcMethod:      "DeleteDrone",
					Use:            "delete-drone [drone_id]",
					Short:          "Delete drone",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "drone_id"}},
				},
				{
					RpcMethod:      "CreateMission",
					Use:            "create-mission [sector] [status] [priority] [assigned-drone-id]",
					Short:          "Create mission",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "sector"}, {ProtoField: "status"}, {ProtoField: "priority"}, {ProtoField: "assigned_drone_id"}},
				},
				{
					RpcMethod:      "UpdateMission",
					Use:            "update-mission [id] [sector] [status] [priority] [assigned-drone-id]",
					Short:          "Update mission",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}, {ProtoField: "sector"}, {ProtoField: "status"}, {ProtoField: "priority"}, {ProtoField: "assigned_drone_id"}},
				},
				{
					RpcMethod:      "DeleteMission",
					Use:            "delete-mission [id]",
					Short:          "Delete mission",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod: "AddReq",
					Use:       "add-req [sector] [priority] [req-type] [coord]",
					Short:     "Send a addReq tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sector"},
						{ProtoField: "priority"},
						{ProtoField: "reqType"},
						{ProtoField: "coord"},
						{ProtoField: "alertId"},
					},
				},
				{
					RpcMethod:      "AssignDrone",
					Use:            "assign-drone [mission-id] [drone-id]",
					Short:          "Send a assignDrone tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "mission_id"}, {ProtoField: "drone_id"}},
				},
				{
					RpcMethod:      "ReportDeadDrone",
					Use:            "report-dead-drone [drone-id]",
					Short:          "Send a reportDeadDrone tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "drone_id"}},
				},
				{
					RpcMethod:      "RegDrone",
					Use:            "reg-drone [drone-id] [sector] [battery]",
					Short:          "Send a regDrone tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "drone_id"}, {ProtoField: "sector"}, {ProtoField: "battery"}},
				},
				{
					RpcMethod: "RmvReq",
					Use:       "rmv-req [mission-id] [drone-id] [laudo]",
					Short:     "Send a rmvReq tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "missionId"},
						{ProtoField: "drone_id"},
						{ProtoField: "laudo"},
					},
				},
				{
					RpcMethod:      "SubmitLaudo",
					Use:            "submit-laudo [requisition-id] [drone-id] [relatorio] [status]",
					Short:          "Send a submitLaudo tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "requisition_id"}, {ProtoField: "drone_id"}, {ProtoField: "relatorio"}, {ProtoField: "status"}},
				},
			},
		},
	}
}
