package distributionx

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"

	"github.com/coinexchain/cet-sdk/modules/bankx"
)

type Keeper struct {
	bxk bankx.Keeper
	dk  distribution.Keeper
}

func NewKeeper(bxk bankx.Keeper, dk distribution.Keeper) Keeper {
	return Keeper{
		bxk,
		dk,
	}
}

func (keeper Keeper) AddCoinsToFeePool(ctx sdk.Context, coins sdk.Coins) {
	feePool := keeper.dk.GetFeePool(ctx)
	feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoins(coins))
	keeper.dk.SetFeePool(ctx, feePool)
}

func (keeper Keeper) DonateToCommunityPool(ctx sdk.Context, fromAddr sdk.AccAddress, amt sdk.Coins) sdk.Error {
	if err := keeper.bxk.DonateCoins(ctx, fromAddr, amt); err != nil {
		return err
	}

	keeper.AddCoinsToFeePool(ctx, amt)
	return nil
}
