package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/keeper"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
)

func createNLaudo(keeper keeper.Keeper, ctx context.Context, n int) []types.Laudo {
	items := make([]types.Laudo, n)
	for i := range items {
		items[i].RequisitionId = strconv.Itoa(i)
		items[i].DroneId = strconv.Itoa(i)
		items[i].Relatorio = strconv.Itoa(i)
		items[i].Status = strconv.Itoa(i)
		items[i].Creator = strconv.Itoa(i)
		_ = keeper.Laudo.Set(ctx, items[i].RequisitionId, items[i])
	}
	return items
}

func TestLaudoQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNLaudo(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetLaudoRequest
		response *types.QueryGetLaudoResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetLaudoRequest{
				RequisitionId: msgs[0].RequisitionId,
			},
			response: &types.QueryGetLaudoResponse{Laudo: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetLaudoRequest{
				RequisitionId: msgs[1].RequisitionId,
			},
			response: &types.QueryGetLaudoResponse{Laudo: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetLaudoRequest{
				RequisitionId: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetLaudo(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestLaudoQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNLaudo(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllLaudoRequest {
		return &types.QueryAllLaudoRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListLaudo(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Laudo), step)
			require.Subset(t, msgs, resp.Laudo)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListLaudo(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Laudo), step)
			require.Subset(t, msgs, resp.Laudo)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListLaudo(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Laudo)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListLaudo(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
