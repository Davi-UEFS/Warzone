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

	// Converte a string da transação para o formato uint64 exigido pelo KVStore
	missionIdUint, err := strconv.ParseUint(msg.MissionId, 10, 64)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "ID da missão inválido: %s", msg.MissionId)
	}

	// 1. Validar e Atualizar a Requisição
	requisicao, err := k.Mission.Get(ctx, missionIdUint)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "requisição %d não encontrada", missionIdUint)
	}

	if requisicao.Status != shared.PENDING {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "requisição %d não está aguardando atendimento", missionIdUint)
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
	if err := k.Mission.Set(ctx, missionIdUint, requisicao); err != nil {
		return nil, err
	}

	drone.Status = string(shared.DRONE_BUSY)
	if err := k.Drone.Set(ctx, msg.DroneId, drone); err != nil {
		return nil, err
	}

	// 4. Emitir Evento para o Manager
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"drone_atribuido",
			sdk.NewAttribute("requisicao_id", msg.MissionId),
			sdk.NewAttribute("drone_id", msg.DroneId),
			sdk.NewAttribute("creator", msg.Creator),
		),
	)

	return &types.MsgAssignDroneResponse{}, nil
}
