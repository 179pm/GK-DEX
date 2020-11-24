package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/genaccounts"

	"github.com/coinexchain/cet-sdk/modules/asset"
	dex "github.com/coinexchain/cet-sdk/types"
)

func newBaseGenesisAccount(address string, amt int64) genaccounts.GenesisAccount {
	return genaccounts.NewGenesisAccount(&auth.BaseAccount{
		Address: accAddressFromBech32(address),
		Coins:   dex.NewCetCoins(amt),
	})
}

func newVestingGenesisAccount(address string, amt int64, endTime int64) genaccounts.GenesisAccount {
	acc, err := genaccounts.NewGenesisAccountI(&auth.DelayedVestingAccount{
		BaseVestingAccount: &auth.BaseVestingAccount{
			BaseAccount: &auth.BaseAccount{
				Address: accAddressFromBech32(address),
				Coins:   dex.NewCetCoins(amt),
			},
			OriginalVesting: dex.NewCetCoins(amt),
			EndTime:         endTime,
		},
	})
	if err != nil {
		panic(err)
	}
	return acc
}

func accAddressFromBech32(address string) sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return addr
}

func createCetToken(ownerAddr string) asset.Token {
	token := &asset.BaseToken{
		Name:             "Diamond Chain Fee Token",
		Symbol:           dex.CET,
		TotalSupply:      sdk.NewInt(100000000000000000),
		Owner:            accAddressFromBech32(ownerAddr),
		SendLock:         sdk.ZeroInt(),
		Mintable:         true,
		Burnable:         false,
		AddrForbiddable:  false,
		TokenForbiddable: false,
		TotalBurn:        sdk.ZeroInt(),
		TotalMint:        sdk.NewInt(50000000000000000),
		IsForbidden:      false,
		URL:              "https://www.diamondnetwork.org",
		Description:      "Decentralized public chain ecosystem, Born for financial liberalization",
	}
	if err := token.Validate(); err != nil {
		panic(err)
	}

	return token
}
