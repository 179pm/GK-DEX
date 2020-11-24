package bancorlite

import (
	"errors"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/coinexchain/cet-sdk/modules/bancorlite/internal/keepers"
	"github.com/coinexchain/cet-sdk/modules/bancorlite/internal/types"
	dex "github.com/coinexchain/cet-sdk/types"
)

type GenesisState struct {
	Params        types.Params                  `json:"params"`
	BancorInfoMap map[string]keepers.BancorInfo `json:"bancor_info_map"`
}

// NewGenesisState - Create a new genesis state
func NewGenesisState(params types.Params, bancorInfoMap map[string]keepers.BancorInfo) GenesisState {
	return GenesisState{
		Params:        params,
		BancorInfoMap: bancorInfoMap,
	}
}

// DefaultGenesisState - Return a default genesis state
func DefaultGenesisState() GenesisState {
	return NewGenesisState(types.DefaultParams(), make(map[string]keepers.BancorInfo))
}

// InitGenesis - Init store state from genesis data
func InitGenesis(ctx sdk.Context, keeper Keeper, data GenesisState) {
	for _, bi := range data.BancorInfoMap {
		keeper.Save(ctx, &bi)
	}
	keeper.SetParams(ctx, data.Params)
}

// ExportGenesis returns a GenesisState for a given context and keeper
func ExportGenesis(ctx sdk.Context, k Keeper) GenesisState {
	m := make(map[string]keepers.BancorInfo)
	k.Iterate(ctx, func(bi *keepers.BancorInfo) {
		m[bi.GetSymbol()] = *bi
	})
	return NewGenesisState(k.GetParams(ctx), m)
}

func (data GenesisState) Validate() error {
	for symbol, bi := range data.BancorInfoMap {
		s := strings.Split(symbol, "/")
		if len(s) != 2 {
			return errors.New("invalid symbol")
		}
		if s[0] != bi.Stock {
			return errors.New("stock mismatch")
		}
		if s[1] != bi.Money {
			return errors.New("money mismatch")
		}
		if s[0] == dex.CET {
			return errors.New("stock can not be cet")
		}
		if len(bi.Owner) == 0 {
			return errors.New("token owner is empty")
		}
		if len(bi.Stock) == 0 {
			return errors.New("stock is empty")
		}
		if len(bi.Money) == 0 {
			return errors.New("money is empty")
		}
		if bi.InitPrice.IsNegative() {
			return errors.New("init price is negative")
		}
		if !bi.MaxSupply.IsPositive() {
			return errors.New("max Supply is not positive")
		}
		if !bi.MaxPrice.IsPositive() {
			return errors.New("max Price is not positive")
		}
		if bi.StockInPool.IsNegative() {
			return errors.New("StockInPool is negative")
		}
		if bi.MoneyInPool.IsNegative() {
			return errors.New("MoneyInPool is negative")
		}
		if bi.EarliestCancelTime < 0 {
			return errors.New("EarliestCancelTime cannot be negative")
		}
		if bi.StockPrecision > 8 {
			return errors.New("invalid stock precision")
		}
		if !bi.IsConsistent() {
			return errors.New("BancorInfo is not consistent")
		}
	}
	return data.Params.ValidateGenesis()
}
