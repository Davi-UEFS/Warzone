package keeper

import (
	"context"

	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) SubmitReport(goCtx context.Context, msg *types.MsgSubmitReport) (*types.MsgSubmitReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Nova API: Tenta buscar o Drone no banco
	drone, err := k.Drone.Get(ctx, msg.DroneId)
	if err != nil {
		// Se o drone não existe, apenas ignora
		return &types.MsgSubmitReportResponse{}, nil
	}

	drone.Status = "Livre"

	// Nova API: Salva o novo estado no banco
	err = k.Drone.Set(ctx, msg.DroneId, drone)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"laudo_recebido",
			sdk.NewAttribute("drone_id", msg.DroneId),
			sdk.NewAttribute("laudo", msg.Laudo),
			sdk.NewAttribute("creator", msg.Creator),
		),
	)

	return &types.MsgSubmitReportResponse{}, nil
}
