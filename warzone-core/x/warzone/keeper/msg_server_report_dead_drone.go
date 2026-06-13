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

	// 1. Busca o drone
	drone, err := k.Drone.Get(ctx, msg.DroneId)
	if err != nil {
		return nil, err
	}

	// 2. Se estava em missão, resgata a missão refém
	if drone.Status == string(shared.DRONE_BUSY) && drone.CurrentMissionId != 0 {
		req, err := k.Mission.Get(ctx, drone.CurrentMissionId)
		if err != nil {
			return nil, err
		}

		// Devolve a missão para a fila global
		req.Status = shared.PENDING
		req.AssignedDroneId = ""
		if err := k.Mission.Set(ctx, drone.CurrentMissionId, req); err != nil {
			return nil, err
		}
	}

	// 3. APAGA O DRONE DE VEZ DA BLOCKCHAIN
	err = k.Drone.Remove(ctx, msg.DroneId)
	if err != nil {
		return nil, err
	}

	// 4. Emite o evento público
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
