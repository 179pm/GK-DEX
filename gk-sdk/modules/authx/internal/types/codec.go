package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

var ModuleCdc = codec.New()

func init() {
	RegisterCodec(ModuleCdc)
}

// RegisterCodec registers concrete types on the codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(AccountX{}, "authx/AccountX", nil)
	cdc.RegisterConcrete(MsgSetReferee{}, "authx/MsgSetReferee", nil)
}
