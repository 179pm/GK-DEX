package keepers

import (
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/coinexchain/cet-sdk/modules/alias/internal/types"
)

const (
	QueryAliasInfo  = "alias-info"
	QueryParameters = "parameters"
)

// creates a querier for asset REST endpoints
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QueryAliasInfo:
			return queryAliasInfo(ctx, req, keeper)
		case QueryParameters:
			return queryParameters(ctx, keeper)
		default:
			return nil, sdk.ErrUnknownRequest("query symbol : " + path[0])
		}
	}
}

const (
	GetAddressFromAlias = 1
	ListAliasOfAccount  = 2
)

type QueryAliasInfoParam struct {
	Owner   sdk.AccAddress `json:"owner"`
	Alias   string         `json:"alias"`
	QueryOp int32          `json:"query_op"`
}

func queryAliasInfo(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var param QueryAliasInfoParam
	if err := types.ModuleCdc.UnmarshalJSON(req.Data, &param); err != nil {
		return nil, sdk.NewError(types.CodeSpaceAlias, types.CodeUnMarshalFailed, "failed to parse param")
	}

	res := []string{}
	if param.QueryOp == GetAddressFromAlias {
		addr, _ := keeper.GetAddressFromAlias(ctx, param.Alias)
		acc := sdk.AccAddress(addr)
		if len(acc) != 0 {
			res = []string{acc.String()}
		}
	} else if param.QueryOp == ListAliasOfAccount {
		res = keeper.GetAliasListOfAccount(ctx, param.Owner)
	} else {
		return nil, sdk.NewError(types.CodeSpaceAlias, types.CodeUnknownOperation, "Unknown Operation")
	}

	bz, err := codec.MarshalJSONIndent(types.ModuleCdc, res)
	if err != nil {
		return nil, sdk.NewError(types.CodeSpaceAlias, types.CodeMarshalFailed, "could not marshal result to JSON")
	}
	return bz, nil
}

func queryParameters(ctx sdk.Context, k Keeper) ([]byte, sdk.Error) {
	params := k.GetParams(ctx)

	res, err := codec.MarshalJSONIndent(types.ModuleCdc, params)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return res, nil
}
