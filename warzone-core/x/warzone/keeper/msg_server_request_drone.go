package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) RequestDrone(goCtx context.Context, msg *types.MsgRequestDrone) (*types.MsgRequestDroneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Extrai o contexto e valida o endereço de quem está pagando
	remetenteAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(err, "endereço da carteira inválido")
	}

	// 2. Define o custo do serviço em Tokens (10 ormuztokens)
	custo, err := sdk.ParseCoinsNormalized("10ormuztoken")
	if err != nil {
		return nil, errorsmod.Wrap(err, "erro ao formatar a moeda")
	}

	// 3. Tenta transferir o saldo da Companhia para o Módulo (Cofre)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, remetenteAddr, types.ModuleName, custo)
	if err != nil {
		return nil, errorsmod.Wrap(err, "saldo insuficiente para despachar o drone")
	}

	// 4. Queima os tokens (retira de circulação conforme a regra de guerra)
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, custo)
	if err != nil {
		return nil, errorsmod.Wrap(err, "erro ao tentar queimar os tokens")
	}

	// ====================================================
	// SALVANDO A MISSÃO (Padrão exato do Ignite)
	// ====================================================

	// 5. Pega o próximo ID da fila usando o sequenciador gerado
	nextId, err := k.MissionSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "falha ao gerar o ID da missão")
	}

	// 6. Monta a missão com a prioridade real do MQTT e o ID automático
	novaMissao := types.Mission{
		Id:              nextId,
		Creator:         msg.Creator,
		Sector:          msg.Sector,
		Status:          "PENDING",
		Priority:        msg.Priority,
		AssignedDroneId: "",
	}

	// 7. Salva de forma imutável no KVStore
	if err = k.Mission.Set(ctx, nextId, novaMissao); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "falha ao salvar a missão no banco")
	}

	return &types.MsgRequestDroneResponse{}, nil
}
