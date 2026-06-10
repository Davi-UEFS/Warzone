package keeper

import (
	"context"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// O seu pacote oficial de constantes
	"github.com/Davi-UEFS/Warzone/shared"
)

func (k msgServer) RegDrone(goCtx context.Context, msg *types.MsgRegDrone) (*types.MsgRegDroneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ====================================================
	// 1. Montar a estrutura inicial do Drone
	// ====================================================
	drone := types.Drone{
		DroneId:          msg.DroneId,
		Sector:           msg.Sector,
		Battery:          msg.Battery,
		Status:           string(shared.DRONE_IDLE), // Nasce livre e pronto para operação
		CurrentMissionId: 0,                         // Nasce sem nenhuma missão vinculada
	}

	// ====================================================
	// 2. Salvar no Banco de Dados
	// ====================================================
	// Grava de forma persistente e imutável no KVStore da blockchain
	if err := k.Drone.Set(ctx, msg.DroneId, drone); err != nil {
		return nil, err
	}

	// ====================================================
	// 3. Emitir Evento para a Rede
	// ====================================================
	// O Manager (via WebSockets/HTTP) escutará isso para saber que a frota aumentou
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"drone_registrado",
			sdk.NewAttribute("drone_id", msg.DroneId),
			sdk.NewAttribute("sector", msg.Sector),
			sdk.NewAttribute("battery", msg.Battery),
		),
	)

	return &types.MsgRegDroneResponse{}, nil
}
