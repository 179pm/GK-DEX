package rest

import (
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"

	"github.com/coinexchain/cet-sdk/modules/alias/internal/types"
	"github.com/coinexchain/cosmos-utils/client/restutil"
)

type AliasUpdateReq struct {
	BaseReq   rest.BaseReq `json:"base_req"`
	Alias     string       `json:"alias"`
	IsAdd     bool         `json:"is_add"`
	AsDefault bool         `json:"as_default"`
}

var _ restutil.RestReq = (*AliasUpdateReq)(nil)

func (req *AliasUpdateReq) New() restutil.RestReq {
	return new(AliasUpdateReq)
}
func (req *AliasUpdateReq) GetBaseReq() *rest.BaseReq {
	return &req.BaseReq
}

func (req *AliasUpdateReq) GetMsg(r *http.Request, sender sdk.AccAddress) (sdk.Msg, error) {
	return &types.MsgAliasUpdate{
		Owner:     sender,
		Alias:     req.Alias,
		IsAdd:     req.IsAdd,
		AsDefault: req.AsDefault,
	}, nil
}

func aliasUpdateHandlerFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return restutil.NewRestHandler(cdc, cliCtx, new(AliasUpdateReq))
}
