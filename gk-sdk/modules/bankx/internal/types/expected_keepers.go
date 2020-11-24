package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/coinexchain/cet-sdk/modules/authx"
)

type ExpectedAccountXKeeper interface {
	GetOrCreateAccountX(ctx sdk.Context, addr sdk.AccAddress) authx.AccountX
	GetAccountX(ctx sdk.Context, addr sdk.AccAddress) (ax authx.AccountX, ok bool)
	SetAccountX(ctx sdk.Context, ax authx.AccountX)
	IterateAccounts(ctx sdk.Context, process func(authx.AccountX) (stop bool))
	InsertUnlockedCoinsQueue(ctx sdk.Context, unlockedTime int64, address sdk.AccAddress)
	RemoveFromUnlockedCoinsQueue(ctx sdk.Context, unlockedTime int64, address sdk.AccAddress)
}

type ExpectedAssetStatusKeeper interface {
	IsForbiddenByTokenIssuer(ctx sdk.Context, symbol string, addr sdk.AccAddress) bool
	IsTokenExists(ctx sdk.Context, symbol string) bool
	UpdateTokenSendLock(ctx sdk.Context, symbol string, amount sdk.Int, lock bool) sdk.Error
}

// SupplyKeeper defines the expected supply keeper
type SupplyKeeper interface {
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) sdk.Error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) sdk.Error
}
