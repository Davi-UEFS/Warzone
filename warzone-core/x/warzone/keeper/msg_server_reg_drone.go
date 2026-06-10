package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
)

func (k msgServer) RegDrone(ctx context.Context, msg *types.MsgRegDrone) (*types.MsgRegDroneResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	// TODO: Handle the message

	return &types.MsgRegDroneResponse{}, nil
}
