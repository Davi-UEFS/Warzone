package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/Davi-UEFS/warzone-core/x/warzone/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) ListDrone(ctx context.Context, req *types.QueryAllDroneRequest) (*types.QueryAllDroneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	drones, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Drone,
		req.Pagination,
		func(_ string, value types.Drone) (types.Drone, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllDroneResponse{Drone: drones, Pagination: pageRes}, nil
}

func (q queryServer) GetDrone(ctx context.Context, req *types.QueryGetDroneRequest) (*types.QueryGetDroneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, err := q.k.Drone.Get(ctx, req.DroneId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetDroneResponse{Drone: val}, nil
}
