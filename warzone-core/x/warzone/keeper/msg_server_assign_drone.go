package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/Davi-UEFS/Warzone/shared"
)

func (k msgServer) AssignDrone(goCtx context.Context, msg *types.MsgAssignDrone) (*types.MsgAssignDroneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Validar e Atualizar a Requisição
	requisicao, err := k.Mission.Get(ctx, msg.MissionId)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "requisição %v não encontrada", msg.MissionId)
	}

	if requisicao.Status != shared.PENDING {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "requisição %v não está aguardando atendimento", msg.MissionId)
	}

	// 2. Validar e Atualizar o Drone
	drone, err := k.Drone.Get(ctx, msg.DroneId)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "drone %s não registrado na rede", msg.DroneId)
	}

	if drone.Status != string(shared.DRONE_IDLE) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "drone %s não está disponível para voo", msg.DroneId)
	}

	// 3. Efetivar a Transição de Estado
	requisicao.Status = shared.IN_PROGRESS
	requisicao.AssignedDroneId = msg.DroneId
	if err := k.Mission.Set(ctx, msg.MissionId, requisicao); err != nil {
		return nil, err
	}

	drone.Status = string(shared.DRONE_BUSY)
	drone.CurrentMissionId = msg.MissionId
	if err := k.Drone.Set(ctx, msg.DroneId, drone); err != nil {
		return nil, err
	}

	// 4. Emitir Evento para o Manager
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"drone_atribuido",
			sdk.NewAttribute("requisicao_id", strconv.Itoa(int(msg.MissionId))),
			sdk.NewAttribute("drone_id", msg.DroneId),
			sdk.NewAttribute("creator", msg.Creator),
		),
	)

	return &types.MsgAssignDroneResponse{}, nil
}
