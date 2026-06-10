package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
)

func (k msgServer) ReportDeadDrone(ctx context.Context, msg *types.MsgReportDeadDrone) (*types.MsgReportDeadDroneResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	// TODO: Handle the message

	return &types.MsgReportDeadDroneResponse{}, nil
}
