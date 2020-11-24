package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"

	"github.com/coinexchain/cet-sdk/modules/bankx/internal/keeper"
	"github.com/coinexchain/cet-sdk/modules/bankx/internal/types"
	"github.com/coinexchain/cosmos-utils/client/restutil"
)

func queryParamsHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := fmt.Sprintf("custom/%s/%s", types.StoreKey, keeper.QueryParameters)
		restutil.RestQuery(nil, cliCtx, w, r, route, nil, nil)
	}
}

// query accountREST Handler
func QueryBalancesRequestHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := fmt.Sprintf("custom/%s/%s", types.StoreKey, keeper.QueryBalances)
		vars := mux.Vars(r)
		acc, err := sdk.AccAddressFromBech32(vars["address"])
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		params := keeper.NewQueryAddrBalances(acc)

		restutil.RestQuery(cdc, cliCtx, w, r, route, &params, nil)
	}
}

func QueryAllBalancesRequestHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := fmt.Sprintf("custom/%s/%s", types.StoreKey, keeper.QueryAllBalances)
		

		restutil.RestQuery(cdc, cliCtx, w, r, route, nil, nil)
	}
}
