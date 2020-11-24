package authx_test

import (
	"os"
	"testing"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/supply"

	"github.com/coinexchain/cet-sdk/modules/asset"
	"github.com/coinexchain/cet-sdk/modules/authx/internal/keepers"
	"github.com/coinexchain/cet-sdk/testapp"
	dex "github.com/coinexchain/cet-sdk/types"
)

func TestMain(m *testing.M) {
	dex.InitSdkConfig()
	os.Exit(m.Run())
}

type testInput struct {
	ctx sdk.Context
	axk keepers.AccountXKeeper
	ak  auth.AccountKeeper
	sk  supply.Keeper
	cdc *codec.Codec
	tk  asset.Keeper
}

func setupTestInput() testInput {
	testApp := testapp.NewTestApp()
	ctx := sdk.NewContext(testApp.Cms, abci.Header{ChainID: "test-chain-id", Time: time.Unix(1560334620, 0)}, false, log.NewNopLogger())
	initSupply := dex.NewCetCoinsE8(10000)
	testApp.SupplyKeeper.SetSupply(ctx, supply.NewSupply(initSupply))

	return testInput{ctx: ctx, axk: testApp.AccountXKeeper, ak: testApp.AccountKeeper,
		sk: testApp.SupplyKeeper, cdc: testApp.Cdc, tk: testApp.AssetKeeper}
}
