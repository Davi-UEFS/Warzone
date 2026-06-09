package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CreateMission(ctx context.Context, msg *types.MsgCreateMission) (*types.MsgCreateMissionResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	nextId, err := k.MissionSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "failed to get next id")
	}

	var mission = types.Mission{
		Id:              nextId,
		Creator:         msg.Creator,
		Sector:          msg.Sector,
		Status:          msg.Status,
		Priority:        msg.Priority,
		AssignedDroneId: msg.AssignedDroneId,
	}

	if err = k.Mission.Set(
		ctx,
		nextId,
		mission,
	); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to set mission")
	}

	return &types.MsgCreateMissionResponse{
		Id: nextId,
	}, nil
}

func (k msgServer) UpdateMission(ctx context.Context, msg *types.MsgUpdateMission) (*types.MsgUpdateMissionResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	var mission = types.Mission{
		Creator:         msg.Creator,
		Id:              msg.Id,
		Sector:          msg.Sector,
		Status:          msg.Status,
		Priority:        msg.Priority,
		AssignedDroneId: msg.AssignedDroneId,
	}

	// Checks that the element exists
	val, err := k.Mission.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("key %d doesn't exist", msg.Id))
		}

		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get mission")
	}

	// Checks if the msg creator is the same as the current owner
	if msg.Creator != val.Creator {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	if err := k.Mission.Set(ctx, msg.Id, mission); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to update mission")
	}

	return &types.MsgUpdateMissionResponse{}, nil
}

func (k msgServer) DeleteMission(ctx context.Context, msg *types.MsgDeleteMission) (*types.MsgDeleteMissionResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	// Checks that the element exists
	val, err := k.Mission.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("key %d doesn't exist", msg.Id))
		}

		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get mission")
	}

	// Checks if the msg creator is the same as the current owner
	if msg.Creator != val.Creator {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	if err := k.Mission.Remove(ctx, msg.Id); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to delete mission")
	}

	return &types.MsgDeleteMissionResponse{}, nil
}
