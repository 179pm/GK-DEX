package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	E8 = 100000000

	CET              = "gkex"
	DefaultBondDenom = CET // default bond denomination
)

func NewCetCoin(amount int64) sdk.Coin {
	return sdk.NewCoin(CET, sdk.NewInt(amount))
}
func NewCetCoinE8(amount int64) sdk.Coin {
	return sdk.NewCoin(CET, sdk.NewInt(amount*E8))
}

func NewCetCoins(amount int64) sdk.Coins {
	return sdk.NewCoins(NewCetCoin(amount))
}
func NewCetCoinsE8(amount int64) sdk.Coins {
	return sdk.NewCoins(NewCetCoin(amount * E8))
}
func NewCoins(denom string, amount int64) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(amount)))
}
func IsCET(coin sdk.Coin) bool {
	return coin.Denom == CET
}
