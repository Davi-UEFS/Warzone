package keeper_test

import (
	"context"
	"strconv"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/keeper"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
)

func createNMission(keeper keeper.Keeper, ctx context.Context, n int) []types.Mission {
	items := make([]types.Mission, n)
	for i := range items {
		iu := uint64(i)
		items[i].Id = iu
		items[i].Sector = strconv.Itoa(i)
		items[i].Status = strconv.Itoa(i)
		items[i].Priority = int64(i)
		items[i].AssignedDroneId = strconv.Itoa(i)
		_ = keeper.Mission.Set(ctx, iu, items[i])
		_ = keeper.MissionSeq.Set(ctx, iu)
	}
	return items
}

func TestMissionQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNMission(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetMissionRequest
		response *types.QueryGetMissionResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetMissionRequest{Id: msgs[0].Id},
			response: &types.QueryGetMissionResponse{Mission: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetMissionRequest{Id: msgs[1].Id},
			response: &types.QueryGetMissionResponse{Mission: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetMissionRequest{Id: uint64(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetMission(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestMissionQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNMission(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllMissionRequest {
		return &types.QueryAllMissionRequest{
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
			resp, err := qs.ListMission(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Mission), step)
			require.Subset(t, msgs, resp.Mission)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListMission(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Mission), step)
			require.Subset(t, msgs, resp.Mission)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListMission(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Mission)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListMission(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
