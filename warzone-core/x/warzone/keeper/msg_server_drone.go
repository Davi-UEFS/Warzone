package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CreateDrone(ctx context.Context, msg *types.MsgCreateDrone) (*types.MsgCreateDroneResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	// Check if the value already exists
	ok, err := k.Drone.Has(ctx, msg.DroneId)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, err.Error())
	} else if ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "index already set")
	}

	var drone = types.Drone{
		Creator: msg.Creator,
		DroneId: msg.DroneId,
		Status:  msg.Status,
		Sector:  msg.Sector,
		Battery: msg.Battery,
	}

	if err := k.Drone.Set(ctx, drone.DroneId, drone); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, err.Error())
	}

	return &types.MsgCreateDroneResponse{}, nil
}

func (k msgServer) UpdateDrone(ctx context.Context, msg *types.MsgUpdateDrone) (*types.MsgUpdateDroneResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid signer address: %s", err))
	}

	// Check if the value exists
	val, err := k.Drone.Get(ctx, msg.DroneId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
		}

		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, err.Error())
	}

	// Checks if the msg creator is the same as the current owner
	if msg.Creator != val.Creator {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	var drone = types.Drone{
		Creator: msg.Creator,
		DroneId: msg.DroneId,
		Status:  msg.Status,
		Sector:  msg.Sector,
		Battery: msg.Battery,
	}

	if err := k.Drone.Set(ctx, drone.DroneId, drone); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to update drone")
	}

	return &types.MsgUpdateDroneResponse{}, nil
}

func (k msgServer) DeleteDrone(ctx context.Context, msg *types.MsgDeleteDrone) (*types.MsgDeleteDroneResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid signer address: %s", err))
	}

	// Check if the value exists
	val, err := k.Drone.Get(ctx, msg.DroneId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
		}

		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, err.Error())
	}

	// Checks if the msg creator is the same as the current owner
	if msg.Creator != val.Creator {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	if err := k.Drone.Remove(ctx, msg.DroneId); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to remove drone")
	}

	return &types.MsgDeleteDroneResponse{}, nil
}
