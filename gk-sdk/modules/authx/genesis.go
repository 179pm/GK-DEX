package authx

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/coinexchain/cet-sdk/modules/authx/internal/types"
)

type GenesisState struct {
	Params    types.Params    `json:"params"`
	AccountXs types.AccountXs `json:"accountxs"`
}

func NewGenesisState(params types.Params, accountXs types.AccountXs) GenesisState {
	return GenesisState{
		Params:    params,
		AccountXs: accountXs,
	}
}

// DefaultGenesisState - Return a default genesis state
func DefaultGenesisState() GenesisState {
	return NewGenesisState(types.DefaultParams(), types.AccountXs{})
}

// InitGenesis - Init store state from genesis data
func InitGenesis(ctx sdk.Context, keeper AccountXKeeper, data GenesisState) {
	keeper.SetParams(ctx, data.Params)

	for _, accx := range data.AccountXs {
		accountX := types.NewAccountX(accx.Address, accx.MemoRequired,
			accx.LockedCoins, accx.FrozenCoins,
			accx.Referee, accx.RefereeChangeTime)
		keeper.SetAccountX(ctx, accountX)
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper
func ExportGenesis(ctx sdk.Context, keeper AccountXKeeper) GenesisState {
	var accountXs types.AccountXs

	keeper.IterateAccounts(ctx, func(accountX types.AccountX) (stop bool) {
		accountXs = append(accountXs, accountX)
		return false
	})

	return NewGenesisState(keeper.GetParams(ctx), accountXs)
}

// ValidateGenesis performs basic validation of asset genesis data returning an
// error for any failed validation criteria.
func (data GenesisState) ValidateGenesis() error {
	err := data.Params.ValidateGenesis()
	if err != nil {
		return err
	}
	addrMap := make(map[string]bool, len(data.AccountXs))
	for _, accx := range data.AccountXs {
		addrStr := accx.Address.String()
		if addrStr == "" {
			return fmt.Errorf("nil accountX found in genesis state; address: %s", addrStr)
		}

		if _, exists := addrMap[addrStr]; exists {
			return fmt.Errorf("duplicate accountX found in genesis state; address: %s", addrStr)
		}

		addrMap[addrStr] = true
	}

	return nil
}
