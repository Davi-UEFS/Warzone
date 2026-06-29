package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/Davi-UEFS/Warzone/shared"
)

// AssignDrone vincula um drone disponível a uma missão pendente na blockchain.
// Esta função é crítica para a concorrência: ela garante de forma atômica que
// um drone não seja atribuído a duas missões e que uma missão não seja pega por dois drones.
func (k msgServer) AssignDrone(goCtx context.Context, msg *types.MsgAssignDrone) (*types.MsgAssignDroneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ====================================================
	// 1. Validação da Requisição (Missão)
	// ====================================================

	// Busca a missão no banco de dados da blockchain usando o ID fornecido.
	requisicao, err := k.Mission.Get(ctx, msg.MissionId)
	if err != nil {
		// Se a missão não existe no ledger, a transação é rejeitada.
		return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "requisição %v não encontrada", msg.MissionId)
	}

	// Trava de Concorrência: Verifica se a missão ainda está aguardando atendimento.
	// Se dois managers tentarem pegar a mesma missão no mesmo bloco, o primeiro altera
	// o status para IN_PROGRESS. O segundo vai cair neste erro e a transação falha.
	if requisicao.Status != shared.PENDING {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "requisição %v não está aguardando atendimento", msg.MissionId)
	}

	// ====================================================
	// 2. Validação do Status do Drone
	// ====================================================

	// Busca o drone registrado no ledger para garantir que ele é reconhecido pela rede.
	drone, err := k.Drone.Get(ctx, msg.DroneId)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "drone %s não registrado na rede", msg.DroneId)
	}

	// Trava de Concorrência: Garante que o drone está livre (IDLE).
	// Impede que um drone que já está em voo (BUSY) receba uma segunda ordem simultânea.
	if drone.Status != string(shared.DRONE_IDLE) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "drone %s não está disponível para voo", msg.DroneId)
	}

	// ====================================================
	// 3. Efetivação da Transição de Estado (Commit)
	// ====================================================

	// Atualiza a missão informando que agora ela está em progresso e vincula o ID do drone.
	requisicao.Status = shared.IN_PROGRESS
	requisicao.AssignedDroneId = msg.DroneId

	// Salva a missão modificada de forma imutável no KVStore da blockchain.
	if err := k.Mission.Set(ctx, msg.MissionId, requisicao); err != nil {
		return nil, err
	}

	// Atualiza o drone informando que ele está ocupado e vincula o ID da missão atual.
	drone.Status = string(shared.DRONE_BUSY)
	drone.CurrentMissionId = msg.MissionId

	// Salva o drone modificado no banco de dados. Como isso ocorre no mesmo bloco,
	// a alteração da missão e do drone é atômica (ou as duas funcionam, ou as duas falham).
	if err := k.Drone.Set(ctx, msg.DroneId, drone); err != nil {
		return nil, err
	}

	// ====================================================
	// 4. Emissão de Eventos (Auditabilidade)
	// ====================================================

	// Dispara um evento nativo na rede Cosmos.
	// Isso permite que o Dashboard, ou qualquer membro do consórcio, filtre o histórico
	// da blockchain para saber exatamente quando e por quem um drone foi despachado.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"drone_atribuido",
			sdk.NewAttribute("requisicao_id", strconv.Itoa(int(msg.MissionId))),
			sdk.NewAttribute("drone_id", msg.DroneId),
			sdk.NewAttribute("creator", msg.Creator),
		),
	)

	// Retorna sucesso. A transação é empacotada no bloco com as mudanças de estado.
	return &types.MsgAssignDroneResponse{}, nil
}
