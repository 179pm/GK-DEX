package authx_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/coinexchain/cet-sdk/modules/authx"
)

func TestValidate(t *testing.T) {
	var addr1, _ = sdk.AccAddressFromBech32("coinex1y5kdxnzn2tfwayyntf2n28q8q2s80mcul852ke")
	var addr2, err = sdk.AccAddressFromBech32("coinex133w8vwj73s4h2uynqft9gyyy52cr6rg8dskv3h")
	require.NoError(t, err)

	genState := authx.DefaultGenesisState()
	require.Nil(t, genState.ValidateGenesis())

	genState = authx.NewGenesisState(authx.NewParams(sdk.NewDec(10), 24*60*60*1000, 1000), []authx.AccountX{authx.NewAccountXWithAddress(addr1), authx.NewAccountXWithAddress(addr2)})
	require.Nil(t, genState.ValidateGenesis())

	errGenState := authx.NewGenesisState(authx.NewParams(sdk.NewDec(-1), 24*60*60*1000, 1000), []authx.AccountX{})
	require.NotNil(t, errGenState.ValidateGenesis())

	errGenState = authx.NewGenesisState(authx.NewParams(sdk.NewDec(10), 24*60*60*1000, 1000), []authx.AccountX{authx.NewAccountXWithAddress(sdk.AccAddress{})})
	require.NotNil(t, errGenState.ValidateGenesis())

	errGenState = authx.NewGenesisState(authx.NewParams(sdk.NewDec(10), 24*60*60*1000, 1000), []authx.AccountX{authx.NewAccountXWithAddress(addr1), authx.NewAccountXWithAddress(addr1)})
	require.NotNil(t, errGenState.ValidateGenesis())

	errGenState = authx.NewGenesisState(authx.NewParams(sdk.NewDec(10), -1, 1000), []authx.AccountX{})
	require.NotNil(t, errGenState.ValidateGenesis())

	errGenState = authx.NewGenesisState(authx.NewParams(sdk.NewDec(10), 24*60*60*1000, 100000), []authx.AccountX{})
	require.NotNil(t, errGenState.ValidateGenesis())

}

func TestExport(t *testing.T) {
	accx := authx.NewAccountX(sdk.AccAddress([]byte("addr")), false, nil, nil, nil, 0)

	testInput := setupTestInput()
	genState1 := authx.NewGenesisState(authx.NewParams(sdk.NewDec(50), 1000, 1000), []authx.AccountX{accx})
	authx.InitGenesis(testInput.ctx, testInput.axk, genState1)
	genState2 := authx.ExportGenesis(testInput.ctx, testInput.axk)
	require.Equal(t, genState1, genState2)
	require.True(t, genState2.Params.Equal(genState1.Params))
}
