package keeper

import (
	"context"
	"strconv" // Necessário para converter o uint64 no evento

	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// RmvReq é chamado quando um drone conclui uma missão. Ele atualiza o status do drone para "IDLE" e a missão para "DONE".
func (k msgServer) RmvReq(goCtx context.Context, msg *types.MsgRmvReq) (*types.MsgRmvReqResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ====================================================
	// 1. Atualizar o Drone para "Livre"
	// ====================================================
	drone, err := k.Drone.Get(ctx, msg.DroneId)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "drone %s não encontrado no banco de dados", msg.DroneId)
	}

	drone.Status = string(shared.DRONE_IDLE)
	drone.CurrentMissionId = 0 // Limpa a missão atual do drone
	if err := k.Drone.Set(ctx, msg.DroneId, drone); err != nil {
		return nil, err
	}

	// ====================================================
	// 2. Atualizar a Requisição para "DONE"
	// ====================================================

	requisicao, err := k.Mission.Get(ctx, msg.MissionId)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "requisição %d não encontrada", msg.MissionId)
	}
	requisicao.Status = shared.DONE

	if err := k.Mission.Set(ctx, msg.MissionId, requisicao); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"requisicao_concluida",
			sdk.NewAttribute("drone_id", msg.DroneId),
			sdk.NewAttribute("requisicao_id", strconv.FormatUint(msg.MissionId, 10)),
			sdk.NewAttribute("laudo", msg.Laudo),
			sdk.NewAttribute("creator", msg.Creator),
		),
	)

	return &types.MsgRmvReqResponse{}, nil
}
