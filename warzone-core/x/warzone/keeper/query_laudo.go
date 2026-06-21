package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) ListLaudo(ctx context.Context, req *types.QueryAllLaudoRequest) (*types.QueryAllLaudoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	laudos, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Laudo,
		req.Pagination,
		func(_ string, value types.Laudo) (types.Laudo, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllLaudoResponse{Laudo: laudos, Pagination: pageRes}, nil
}

func (q queryServer) GetLaudo(ctx context.Context, req *types.QueryGetLaudoRequest) (*types.QueryGetLaudoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, err := q.k.Laudo.Get(ctx, req.RequisitionId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetLaudoResponse{Laudo: val}, nil
}
