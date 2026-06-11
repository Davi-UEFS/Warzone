package keeper

import (
	"context"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) SubmitLaudo(goCtx context.Context, msg *types.MsgSubmitLaudo) (*types.MsgSubmitLaudoResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Emissão do evento de "Laudo Submetido".
	// Isso cria um log indexado na blockchain que pode ser consultado via REST API (porta 1317)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"laudo_registrado",
			sdk.NewAttribute("requisition_id", msg.RequisitionId),
			sdk.NewAttribute("drone_id", msg.DroneId),
			sdk.NewAttribute("relatorio", msg.Relatorio),
			sdk.NewAttribute("status", msg.Status),
			sdk.NewAttribute("creator", msg.Creator),
		),
	)

	return &types.MsgSubmitLaudoResponse{}, nil
}
