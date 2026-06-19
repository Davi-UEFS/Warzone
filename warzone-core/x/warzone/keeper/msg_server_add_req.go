package keeper

import (
	"context"

	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var prices = map[string]string{
	shared.FIRE:           "10stake",
	shared.OIL:            "8stake",
	shared.WRECKAGE:       "6stake",
	shared.INSPECTION:     "2stake",
	shared.UNKNOWN_OBJECT: "4stake",
	shared.BOTTLENECK:     "4stake",
}

func (k msgServer) AddReq(goCtx context.Context, msg *types.MsgAddReq) (*types.MsgAddReqResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Abre uma conexão direta com o banco de dados interno da sua blockchain
	store := k.storeService.OpenKVStore(ctx)

	// Cria uma chave única de rastreio para este alerta específico
	alertaKey := []byte("alerta_processado_" + msg.AlertId)

	// Verifica se algum outro Manager já cravou esta chave no bloco milissegundos antes
	jaProcessado, _ := store.Has(alertaKey)
	if jaProcessado {
		// Se a chave já existe, a blockchain rejeita a transação.
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "alerta duplicado: este incidente já foi despachado pela rede")
	}

	// Se é a primeira vez que a rede vê este alerta, guarda a chave no banco.
	// O valor []byte{1} é apenas um bool mais simples.
	store.Set(alertaKey, []byte{1})

	// ====================================================
	// 1. Validação Financeira (A Cobrança)
	// ====================================================

	// Transforma a string do PAYER (País Pagante) em um endereço válido
	paganteAddr, err := sdk.AccAddressFromBech32(msg.Payer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "endereço da carteira do país pagante é inválido")
	}

	// Pega o custo do serviço com base no tipo de requisição
	custoStr, exists := prices[msg.ReqType]
	if !exists {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "tipo de requisição não suportado")
	}

	// Define o custo do serviço convertendo a string (ex: "10token") para o tipo Coin nativo
	custo, err := sdk.ParseCoinsNormalized(custoStr)
	if err != nil {
		return nil, errorsmod.Wrap(err, "erro ao formatar a moeda de cobrança")
	}

	// TENTA RETIRAR OS FUNDOS DO PAÍS E MOVER PARA O MÓDULO
	// Se o país não tiver dinheiro, a função devolve erro aqui e a missão NUNCA entra na rede!
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, paganteAddr, types.ModuleName, custo)
	if err != nil {
		return nil, errorsmod.Wrap(err, "alerta rejeitado: o país não tem saldo suficiente para despachar o drone")
	}

	// QUEIMA AS MOEDAS (Retira do suprimento total da rede)
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, custo)
	if err != nil {
		return nil, errorsmod.Wrap(err, "falha crítica ao queimar os tokens do país")
	}

	// ====================================================
	// 2. Registo da Missão no Banco de Dados
	// ====================================================

	// Pega o próximo ID automático da fila
	nextId, err := k.MissionSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "falha ao gerar o ID da missão")
	}

	// Monta a estrutura da missão
	novaMissao := types.Mission{
		Id:              nextId,
		Creator:         msg.Creator, // O Manager que assinou a transação
		Sector:          msg.Sector,
		Status:          shared.PENDING,
		Priority:        msg.Priority,
		AssignedDroneId: "",
		ReqType:         msg.ReqType,
		Coord:           msg.Coord,
		CreatedAt:       ctx.BlockTime().Unix(), // Relógio imutável validado pelo consenso
	}

	// Salva de forma imutável no KVStore
	if err = k.Mission.Set(ctx, nextId, novaMissao); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "falha ao salvar a missão no banco de dados da blockchain")
	}

	// Retorna sucesso e a rede inclui a transação no bloco!
	return &types.MsgAddReqResponse{}, nil
}
