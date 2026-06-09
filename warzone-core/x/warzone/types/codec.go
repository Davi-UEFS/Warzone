package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitReport{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRequestDrone{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateMission{},
		&MsgUpdateMission{},
		&MsgDeleteMission{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateDrone{},
		&MsgUpdateDrone{},
		&MsgDeleteDrone{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
