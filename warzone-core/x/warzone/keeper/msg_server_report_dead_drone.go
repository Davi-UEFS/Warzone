package keeper

import (
	"context"
	"strconv"

	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ReportDeadDrone(goCtx context.Context, msg *types.MsgReportDeadDrone) (*types.MsgReportDeadDroneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	drone, err := k.Drone.Get(ctx, msg.DroneId)
	if err != nil {
		return nil, err
	}

	if drone.Status == string(shared.DRONE_BUSY) && drone.CurrentMissionId != 0 {

		req, err := k.Mission.Get(ctx, drone.CurrentMissionId)
		if err != nil {
			return nil, err
		}

		// Recupera missao
		req.Status = shared.PENDING
		req.AssignedDroneId = "" // Desvincula o drone da missão
		if err := k.Mission.Set(ctx, drone.CurrentMissionId, req); err != nil {
			return nil, err
		}
	}

	if err := k.Drone.Set(ctx, msg.DroneId, drone); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"drone_inoperante",
			sdk.NewAttribute("drone_id", msg.DroneId),
			sdk.NewAttribute("missao_abortada_id", strconv.FormatUint(drone.CurrentMissionId, 10)),
			sdk.NewAttribute("solicitante", msg.Creator),
		),
	)

	return &types.MsgReportDeadDroneResponse{}, nil
}
