package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) AddReq(goCtx context.Context, msg *types.MsgAddReq) (*types.MsgAddReqResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Validação Financeira
	// Transforma a string do remetente em um endereço criptográfico válido
	remetenteAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(err, "endereço da carteira inválido")
	}

	// Define o custo do serviço em Tokens (10 ormuztokens)
	custo, err := sdk.ParseCoinsNormalized("10token")
	if err != nil {
		return nil, errorsmod.Wrap(err, "erro ao formatar a moeda")
	}

	// Tenta transferir o saldo da carteira da Companhia para o Módulo
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, remetenteAddr, types.ModuleName, custo)
	if err != nil {
		return nil, errorsmod.Wrap(err, "saldo insuficiente para pedir um drone")
	}

	// Queima os tokens (retira de circulação do ecossistema)
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, custo)
	if err != nil {
		return nil, errorsmod.Wrap(err, "erro ao queimar tokens")
	}

	// ====================================================
	// 2. Salvar no Banco de Dados
	// ====================================================

	// Pega o próximo ID automático da fila
	nextId, err := k.MissionSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "falha ao gerar o ID da missão")
	}

	// Monta a estrutura da missão
	novaMissao := types.Mission{
		Id:              nextId,
		Creator:         msg.Creator,
		Sector:          msg.Sector,
		Status:          shared.PENDING,
		Priority:        msg.Priority,
		AssignedDroneId: "",
		ReqType:         msg.ReqType,            // Vem do Protobuf
		Coord:           msg.Coord,              // Vem do Protobuf
		CreatedAt:       ctx.BlockTime().Unix(), // Relógio imutável do bloco
	}

	// Salva de forma imutável no KVStore
	if err = k.Mission.Set(ctx, nextId, novaMissao); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "falha ao salvar a missão no banco")
	}

	// Retorna sucesso para a transação
	return &types.MsgAddReqResponse{}, nil
}
