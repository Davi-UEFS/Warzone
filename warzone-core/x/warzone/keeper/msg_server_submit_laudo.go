package keeper

import (
	"context"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) SubmitLaudo(goCtx context.Context, msg *types.MsgSubmitLaudo) (*types.MsgSubmitLaudoResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ====================================================
	// 1. Emitir Evento para a Rede
	// ====================================================
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

	// ====================================================
	// 2. Montar a estrutura do Laudo
	// ====================================================
	novoLaudo := types.Laudo{
		RequisitionId: msg.RequisitionId,
		DroneId:       msg.DroneId,
		Relatorio:     msg.Relatorio,
		Status:        msg.Status,
		Creator:       msg.Creator,
	}

	// ====================================================
	// 3. Salvar no Banco de Dados (KVStore / Collections)
	// ====================================================
	// Grava de forma persistente. A chave é o RequisitionId.
	if err := k.Laudo.Set(ctx, msg.RequisitionId, novoLaudo); err != nil {
		return nil, err
	}

	return &types.MsgSubmitLaudoResponse{}, nil
}
