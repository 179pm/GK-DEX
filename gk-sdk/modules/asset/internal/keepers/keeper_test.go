package keepers_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/supply"

	"github.com/coinexchain/cet-sdk/modules/asset/internal/types"
	"github.com/coinexchain/cet-sdk/modules/authx"
)

func TestTokenKeeper_IssueToken(t *testing.T) {
	input := createTestInput()

	type args struct {
		ctx sdk.Context
		msg types.MsgIssueToken
	}
	tests := []struct {
		name string
		args args
		want sdk.Error
	}{
		{
			"base-case",
			args{
				input.ctx,
				types.NewMsgIssueToken("ABC Token", "abc", sdk.NewInt(2100), testAddr,
					false, false, false, false, "", "", types.TestIdentityString),
			},
			nil,
		},
		{
			"case-duplicate",
			args{
				input.ctx,
				types.NewMsgIssueToken("ABC Token", "abc", sdk.NewInt(2100), testAddr,
					false, false, false, false, "", "", types.TestIdentityString),
			},
			types.ErrDuplicateTokenSymbol("abc"),
		},
		{
			"case-invalid",
			args{
				input.ctx,
				types.NewMsgIssueToken("ABC Token", "999", sdk.NewInt(2100), testAddr,
					false, false, false, false, "", "", types.TestIdentityString),
			},
			types.ErrInvalidTokenSymbol("999"),
		},
		{
			"invalid owner address",
			args{
				input.ctx,
				types.NewMsgIssueToken("ABC Token", "add", sdk.NewInt(2100), supply.NewModuleAddress(authx.ModuleName),
					false, false, false, false, "", "", types.TestIdentityString),
			},
			types.ErrAccInBlackList(supply.NewModuleAddress(authx.ModuleName)),
		},
		{
			"suffix token symbol",
			args{
				input.ctx,
				types.NewMsgIssueToken("ABC Token", "acd.ss", sdk.NewInt(2100), testAddr,
					false, false, false, false, "", "", types.TestIdentityString),
			},
			types.ErrInvalidTokenSymbol("acd.ss"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := input.tk.IssueToken(
				tt.args.ctx,
				tt.args.msg.Name,
				tt.args.msg.Symbol,
				tt.args.msg.TotalSupply,
				tt.args.msg.Owner,
				tt.args.msg.Mintable,
				tt.args.msg.Burnable,
				tt.args.msg.AddrForbiddable,
				tt.args.msg.TokenForbiddable,
				tt.args.msg.URL,
				tt.args.msg.Description,
				tt.args.msg.Identity,
			); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TokenKeeper.IssueToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenKeeper_TokenStore(t *testing.T) {
	input := createTestInput()

	// set token
	token1, err := types.NewToken("ABC token", "abc", sdk.NewInt(2100), testAddr,
		false, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.SetToken(input.ctx, token1)
	require.NoError(t, err)

	token2, err := types.NewToken("XYZ token", "xyz", sdk.NewInt(2100), testAddr,
		false, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.SetToken(input.ctx, token2)
	require.NoError(t, err)

	// get all tokens
	tokens := input.tk.GetAllTokens(input.ctx)
	require.Equal(t, 2, len(tokens))
	require.Contains(t, []string{"abc", "xyz"}, tokens[0].GetSymbol())
	require.Contains(t, []string{"abc", "xyz"}, tokens[1].GetSymbol())

	// remove token
	input.tk.RemoveToken(input.ctx, token1)

	// get token
	res := input.tk.GetToken(input.ctx, token1.GetSymbol())
	require.Nil(t, res)

}
func TestTokenKeeper_TokenReserved(t *testing.T) {
	input := createTestInput()
	addr, _ := sdk.AccAddressFromBech32("coinex133w8vwj73s4h2uynqft9gyyy52cr6rg8dskv3h")
	expectErr := types.ErrInvalidIssueOwner()

	// issue btc token failed
	err := input.tk.IssueToken(input.ctx, "BTC token", "btc", sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.Equal(t, expectErr, err)

	// issue abc token success
	err = input.tk.IssueToken(input.ctx, "ABC token", "abc", sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)

	// issue cet token success
	err = input.tk.IssueToken(input.ctx, "CET token", "cet", sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)

	// cet owner issue btc token success
	err = input.tk.IssueToken(input.ctx, "BTC token", "btc", sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)

	// only cet owner can issue reserved token
	err = input.tk.IssueToken(input.ctx, "ETH token", "eth", sdk.NewInt(2100), addr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.Equal(t, expectErr, err)

}

func TestTokenKeeper_TransferOwnership(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	var addr1, _ = sdk.AccAddressFromBech32("coinex133w8vwj73s4h2uynqft9gyyy52cr6rg8dskv3h")

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		false, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.TransferOwnership(input.ctx, symbol, testAddr, addr1)
	require.NoError(t, err)

	// get token
	token := input.tk.GetToken(input.ctx, symbol)
	require.NotNil(t, token)
	require.Equal(t, addr1.String(), token.GetOwner().String())

	//case2: invalid token
	err = input.tk.TransferOwnership(input.ctx, "xyz", testAddr, addr1)
	require.Error(t, err)

	//case3: invalid original owner
	err = input.tk.TransferOwnership(input.ctx, symbol, testAddr, addr1)
	require.Error(t, err)

	//case4: invalid new owner
	err = input.tk.TransferOwnership(input.ctx, symbol, addr1, sdk.AccAddress{})
	require.Error(t, err)

	//case5: invalid new owner
	err = input.tk.TransferOwnership(input.ctx, symbol, addr1, supply.NewModuleAddress(authx.ModuleName))
	require.Error(t, err)
}

func TestTokenKeeper_MintToken(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	var addr, _ = sdk.AccAddressFromBech32("coinex133w8vwj73s4h2uynqft9gyyy52cr6rg8dskv3h")

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.MintToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.NoError(t, err)

	token := input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, sdk.NewInt(3100), token.GetTotalSupply())
	require.Equal(t, sdk.NewInt(1000), token.GetTotalMint())

	err = input.tk.MintToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.NoError(t, err)
	token = input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, sdk.NewInt(4100), token.GetTotalSupply())
	require.Equal(t, sdk.NewInt(2000), token.GetTotalMint())

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 2: un mintable token
	// set token mintable: false
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		false, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.MintToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 3: mint invalid token
	err = input.tk.IssueToken(input.ctx, "ABC token", "xyz", sdk.NewInt(2100), testAddr,
		true, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.MintToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 4: only token owner can mint token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), addr,
		true, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.MintToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)

}

func TestTokenKeeper_BurnToken(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	var addr, _ = sdk.AccAddressFromBech32("coinex133w8vwj73s4h2uynqft9gyyy52cr6rg8dskv3h")

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.NoError(t, err)

	token := input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, sdk.NewInt(1100), token.GetTotalSupply())
	require.Equal(t, sdk.NewInt(1000), token.GetTotalBurn())

	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.NoError(t, err)
	token = input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, sdk.NewInt(100), token.GetTotalSupply())
	require.Equal(t, sdk.NewInt(2000), token.GetTotalBurn())

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 2: un burnable token
	// set token burnable: false
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		false, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 3: burn invalid token
	err = input.tk.IssueToken(input.ctx, "ABC token", "xyz", sdk.NewInt(2100), testAddr,
		true, true, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 4: only token owner can burn token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), addr,
		true, true, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 5: token total supply limited to > 0
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(2100))
	require.Error(t, err)
	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(2200))
	require.Error(t, err)
}

func TestTokenKeeper_BurnTokenAfterModifyToken(t *testing.T) {
	input := createTestInput()
	symbol := "abc"

	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		false, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)
	token := input.tk.GetToken(input.ctx, symbol)
	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.Error(t, err)
	require.Contains(t, err.Error(), "token abc do not support burn")

	err = input.tk.ModifyTokenInfo(input.ctx, symbol, token.GetOwner(),
		token.GetURL(), token.GetDescription(), token.GetIdentity(), token.GetName(),
		token.GetTotalSupply(), token.GetMintable(), true,
		token.GetAddrForbiddable(), token.GetTokenForbiddable())
	require.NoError(t, err)

	err = input.tk.BurnToken(input.ctx, symbol, testAddr, sdk.NewInt(1000))
	require.NoError(t, err)
	token = input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, sdk.NewInt(1100), token.GetTotalSupply())
	require.Equal(t, sdk.NewInt(1000), token.GetTotalBurn())
}

func TestTokenKeeper_ForbidToken(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	var addr, _ = sdk.AccAddressFromBech32("coinex133w8vwj73s4h2uynqft9gyyy52cr6rg8dskv3h")

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.ForbidToken(input.ctx, symbol, testAddr)
	require.NoError(t, err)
	require.True(t, input.tk.IsTokenForbidden(input.ctx, symbol))

	// remove token
	token := input.tk.GetToken(input.ctx, symbol)
	input.tk.RemoveToken(input.ctx, token)

	//case 2: un forbiddable token
	// set token forbiddable: false
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		false, false, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.ForbidToken(input.ctx, symbol, testAddr)
	require.Error(t, err)
	require.False(t, input.tk.IsTokenForbidden(input.ctx, symbol))

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 3: duplicate forbid token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.ForbidToken(input.ctx, symbol, testAddr)
	require.NoError(t, err)

	err = input.tk.ForbidToken(input.ctx, symbol, testAddr)
	require.Error(t, err)
	require.True(t, input.tk.IsTokenForbidden(input.ctx, symbol))

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 4: only token owner can forbid token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), addr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.ForbidToken(input.ctx, symbol, testAddr)
	require.Error(t, err)
	require.False(t, input.tk.IsTokenForbidden(input.ctx, symbol))

	// remove token
	input.tk.RemoveToken(input.ctx, token)

}

func TestTokenKeeper_UnForbidToken(t *testing.T) {
	input := createTestInput()
	symbol := "abc"

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.ForbidToken(input.ctx, symbol, testAddr)
	require.NoError(t, err)

	token := input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, true, token.GetIsForbidden())

	err = input.tk.UnForbidToken(input.ctx, symbol, testAddr)
	require.NoError(t, err)

	token = input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, false, token.GetIsForbidden())

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 2: unforbid token before forbid token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)
	err = input.tk.UnForbidToken(input.ctx, symbol, testAddr)
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)
}

func TestTokenKeeper_AddTokenWhitelist(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	whitelist := mockAddrList()

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)
	token := input.tk.GetToken(input.ctx, symbol)

	err = input.tk.AddTokenWhitelist(input.ctx, symbol, testAddr, whitelist)
	require.NoError(t, err)
	addresses := input.tk.GetWhitelist(input.ctx, symbol)
	for _, addr := range addresses {
		require.Contains(t, whitelist, addr)
	}
	require.Equal(t, len(whitelist), len(addresses))
	for _, addr := range whitelist {
		require.False(t, input.tk.IsForbiddenByTokenIssuer(input.ctx, symbol, addr))
	}

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 2: un forbiddable token
	// set token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.AddTokenWhitelist(input.ctx, symbol, testAddr, whitelist)
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)
}

func TestTokenKeeper_RemoveTokenWhitelist(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	whitelist := mockAddrList()

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, true, "", "", types.TestIdentityString)
	require.NoError(t, err)
	token := input.tk.GetToken(input.ctx, symbol)

	err = input.tk.AddTokenWhitelist(input.ctx, symbol, testAddr, whitelist)
	require.NoError(t, err)
	addresses := input.tk.GetWhitelist(input.ctx, symbol)
	for _, addr := range addresses {
		require.Contains(t, whitelist, addr)
	}
	require.Equal(t, len(whitelist), len(addresses))

	err = input.tk.RemoveTokenWhitelist(input.ctx, symbol, testAddr, []sdk.AccAddress{whitelist[0]})
	require.NoError(t, err)
	addresses = input.tk.GetWhitelist(input.ctx, symbol)
	require.Equal(t, len(whitelist)-1, len(addresses))
	require.NotContains(t, addresses, whitelist[0])

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 2: un-forbiddable token
	// set token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.RemoveTokenWhitelist(input.ctx, symbol, testAddr, whitelist)
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)
}

func TestTokenKeeper_ForbidAddress(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	mock := mockAddrList()

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, true, true, "", "", types.TestIdentityString)
	require.NoError(t, err)
	token := input.tk.GetToken(input.ctx, symbol)

	err = input.tk.ForbidAddress(input.ctx, symbol, testAddr, mock)
	require.NoError(t, err)
	forbidden := input.tk.GetForbiddenAddresses(input.ctx, symbol)
	for _, addr := range forbidden {
		require.Contains(t, mock, addr)
	}
	require.Equal(t, len(mock), len(forbidden))
	for _, addr := range mock {
		require.True(t, input.tk.IsForbiddenByTokenIssuer(input.ctx, symbol, addr))
	}
	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 2: addr un-forbiddable token
	// set token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.ForbidAddress(input.ctx, symbol, testAddr, mock)
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)
}

func TestTokenKeeper_UnForbidAddress(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	mock := mockAddrList()

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, true, true, "", "", types.TestIdentityString)
	require.NoError(t, err)
	token := input.tk.GetToken(input.ctx, symbol)

	err = input.tk.ForbidAddress(input.ctx, symbol, testAddr, mock)
	require.NoError(t, err)
	forbidden := input.tk.GetForbiddenAddresses(input.ctx, symbol)
	for _, addr := range forbidden {
		require.Contains(t, mock, addr)
	}
	require.Equal(t, len(mock), len(forbidden))

	err = input.tk.UnForbidAddress(input.ctx, symbol, testAddr, []sdk.AccAddress{mock[0]})
	require.NoError(t, err)
	forbidden = input.tk.GetForbiddenAddresses(input.ctx, symbol)
	require.Equal(t, len(mock)-1, len(forbidden))
	require.NotContains(t, forbidden, mock[0])
	require.False(t, input.tk.IsForbiddenByTokenIssuer(input.ctx, symbol, mock[0]))

	// remove token
	input.tk.RemoveToken(input.ctx, token)

	//case 2: addr un-forbiddable token
	// set token
	err = input.tk.IssueToken(input.ctx, "ABC token", symbol, sdk.NewInt(2100), testAddr,
		true, true, false, false, "", "", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.UnForbidAddress(input.ctx, symbol, testAddr, mock)
	require.Error(t, err)

	// remove token
	input.tk.RemoveToken(input.ctx, token)
}

func TestTokenKeeper_ModifyTokenInfo(t *testing.T) {
	input := createTestInput()
	symbol := "abc"
	var addr, _ = sdk.AccAddressFromBech32("coinex133w8vwj73s4h2uynqft9gyyy52cr6rg8dskv3h")
	url := "www.abc.com"
	description := "token abc is a example token"
	identity := types.TestIdentityString
	supply := sdk.NewInt(2100)
	name := "ABC token"
	mintable := true
	burnable := false
	addrForbiddable := false
	tokenForbiddable := false

	//case 1: base-case ok
	// set token
	err := input.tk.IssueToken(input.ctx, name, symbol, supply, testAddr,
		true, false, false, false, "www.abc.org", "abc example description", types.TestIdentityString)
	require.NoError(t, err)

	err = input.tk.ModifyTokenInfo(input.ctx, symbol, testAddr, url, description, identity,
		name, supply, mintable, burnable, addrForbiddable, tokenForbiddable)
	require.NoError(t, err)
	token := input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, url, token.GetURL())
	require.Equal(t, description, token.GetDescription())

	//case 2: only token owner can modify token info
	err = input.tk.ModifyTokenInfo(input.ctx, symbol, addr, "www.abc.org", "token abc is a example token", identity,
		name, supply, mintable, burnable, addrForbiddable, tokenForbiddable)
	require.Error(t, err)
	token = input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, url, token.GetURL())
	require.Equal(t, description, token.GetDescription())

	//case 3: invalid url
	err = input.tk.ModifyTokenInfo(input.ctx, symbol, testAddr, string(make([]byte, types.MaxTokenURLLength+1)), description, identity,
		name, supply, mintable, burnable, addrForbiddable, tokenForbiddable)
	require.Error(t, err)
	token = input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, url, token.GetURL())
	require.Equal(t, description, token.GetDescription())

	//case 4: invalid description
	err = input.tk.ModifyTokenInfo(input.ctx, symbol, testAddr, url, string(make([]byte, types.MaxTokenDescriptionLength+1)), identity,
		name, supply, mintable, burnable, addrForbiddable, tokenForbiddable)
	require.Error(t, err)
	token = input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, url, token.GetURL())
	require.Equal(t, description, token.GetDescription())

	//case 4: invalid identity
	err = input.tk.ModifyTokenInfo(input.ctx, symbol, testAddr, url, description, string(make([]byte, types.MaxTokenIdentityLength+1)),
		name, supply, mintable, burnable, addrForbiddable, tokenForbiddable)
	require.Error(t, err)
	token = input.tk.GetToken(input.ctx, symbol)
	require.Equal(t, url, token.GetURL())
	require.Equal(t, description, token.GetDescription())
}

func TestTokenKeeper_ModifyTokenInfo_OK(t *testing.T) {
	tokenTmpl, err := types.NewToken("ABC token", "abc", sdk.NewInt(2100), testAddr,
		false, true, false, false, "www.abc.org", "abc example description", types.TestIdentityString)
	require.NoError(t, err)

	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { return token.SetURL("new.url") }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { return token.SetDescription("newDesc") }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { return token.SetIdentity("newID") }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { return token.SetName("newName") }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { return token.SetTotalSupply(token.GetTotalSupply().MulRaw(2)) }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { return token.SetTotalSupply(token.GetTotalSupply().SubRaw(100)) }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { token.SetMintable(!token.GetMintable()); return nil }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { token.SetBurnable(!token.GetBurnable()); return nil }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { token.SetAddrForbiddable(!token.GetAddrForbiddable()); return nil }, "")
	testModifyTokenInfo(t, tokenTmpl, false, func(token types.Token) error { token.SetTokenForbiddable(!token.GetTokenForbiddable()); return nil }, "")
}

func TestTokenKeeper_ModifyTokenInfo_AfterDistribution(t *testing.T) {
	tokenTmpl, err := types.NewToken("ABC token", "abc", sdk.NewInt(2100), testAddr,
		true, false, true, true, "www.abc.org", "abc example description", types.TestIdentityString)
	require.NoError(t, err)

	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { return token.SetURL("new.url") }, "")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { return token.SetDescription("newDesc") }, "")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { return token.SetIdentity("newID") }, "")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { return token.SetName("newName") }, "token Name sealed")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { return token.SetTotalSupply(token.GetTotalSupply().MulRaw(2)) }, "token TotalSupply sealed")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { token.SetMintable(!token.GetMintable()); return nil }, "")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { token.SetBurnable(!token.GetBurnable()); return nil }, "")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { token.SetAddrForbiddable(!token.GetAddrForbiddable()); return nil }, "")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { token.SetTokenForbiddable(!token.GetTokenForbiddable()); return nil }, "")
}

func TestTokenKeeper_ModifyTokenInfo_AfterDistribution2(t *testing.T) {
	tokenTmpl, err := types.NewToken("ABC token", "abc", sdk.NewInt(2100), testAddr,
		false, true, false, false, "www.abc.org", "abc example description", types.TestIdentityString)
	require.NoError(t, err)

	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { token.SetMintable(!token.GetMintable()); return nil }, "token Mintable sealed")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { token.SetBurnable(!token.GetBurnable()); return nil }, "token Burnable sealed")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { token.SetAddrForbiddable(!token.GetAddrForbiddable()); return nil }, "token AddrForbiddable sealed")
	testModifyTokenInfo(t, tokenTmpl, true, func(token types.Token) error { token.SetTokenForbiddable(!token.GetTokenForbiddable()); return nil }, "token TokenForbiddable sealed")
}

func TestTokenKeeper_ModifyMintableToFalse(t *testing.T) {
	tokenTmpl, err := types.NewToken("ABC token", "abc", sdk.NewInt(2100), testAddr,
		true, true, true, true, "www.abc.org", "abc example description", types.TestIdentityString)
	require.NoError(t, err)
	err = tokenTmpl.SetTotalMint(sdk.NewInt(10))
	require.NoError(t, err)
	modifyFn := func(token types.Token) error { token.SetMintable(false); return nil }
	newToken := testModifyTokenInfo(t, tokenTmpl, true, modifyFn, "")
	require.True(t, newToken.GetTotalMint().IsZero()) // total_mint -> 0
}
func TestTokenKeeper_ModifyBurnableToFalse(t *testing.T) {
	tokenTmpl, err := types.NewToken("ABC token", "abc", sdk.NewInt(2100), testAddr,
		true, true, true, true, "www.abc.org", "abc example description", types.TestIdentityString)
	require.NoError(t, err)
	err = tokenTmpl.SetTotalBurn(sdk.NewInt(10))
	require.NoError(t, err)
	modifyFn := func(token types.Token) error { token.SetBurnable(false); return nil }
	newToken := testModifyTokenInfo(t, tokenTmpl, false, modifyFn, "")
	require.True(t, newToken.GetTotalBurn().IsZero()) // total_burn -> 0
}
func TestTokenKeeper_ModifyTokenForbiddableToFalse(t *testing.T) {
	tokenTmpl, err := types.NewToken("ABC token", "abc", sdk.NewInt(2100), testAddr,
		true, true, true, true, "www.abc.org", "abc example description", types.TestIdentityString)
	require.NoError(t, err)
	tokenTmpl.SetIsForbidden(true)
	modifyFn := func(token types.Token) error { token.SetTokenForbiddable(false); return nil }
	newToken := testModifyTokenInfo(t, tokenTmpl, true, modifyFn, "")
	require.False(t, newToken.GetIsForbidden()) // forbidden -> false
}
func TestTokenKeeper_ModifyAddrForbiddableToFalse(t *testing.T) {
	tokenTmpl, err := types.NewToken("ABC token", "abc", sdk.NewInt(2100), testAddr,
		true, true, true, true, "www.abc.org", "abc example description", types.TestIdentityString)
	require.NoError(t, err)

	input := createTestInput()
	err = input.tk.IssueToken(input.ctx, tokenTmpl.GetName(), tokenTmpl.GetSymbol(),
		tokenTmpl.GetTotalSupply(), tokenTmpl.GetOwner(),
		tokenTmpl.GetMintable(), tokenTmpl.GetBurnable(),
		tokenTmpl.GetAddrForbiddable(), tokenTmpl.GetTokenForbiddable(),
		tokenTmpl.GetURL(), tokenTmpl.GetDescription(), tokenTmpl.GetIdentity())
	require.NoError(t, err)

	var _, _, badGuy1 = keyPubAddr()
	var _, _, badGuy2 = keyPubAddr()
	badGuys := []sdk.AccAddress{badGuy1, badGuy2}
	err = input.tk.ForbidAddress(input.ctx, tokenTmpl.GetSymbol(), tokenTmpl.GetOwner(), badGuys)
	require.NoError(t, err)
	blackList := input.tk.GetForbiddenAddresses(input.ctx, tokenTmpl.GetSymbol())
	require.Equal(t, 2, len(blackList))
	require.True(t, input.tk.GetToken(input.ctx, tokenTmpl.GetSymbol()).GetAddrForbiddable())

	err = input.tk.ModifyTokenInfo(input.ctx, tokenTmpl.GetSymbol(), tokenTmpl.GetOwner(),
		tokenTmpl.GetURL(), tokenTmpl.GetDescription(), tokenTmpl.GetIdentity(),
		tokenTmpl.GetName(), tokenTmpl.GetTotalSupply(),
		tokenTmpl.GetMintable(), tokenTmpl.GetBurnable(),
		false, tokenTmpl.GetTokenForbiddable())
	require.NoError(t, err)
	blackList = input.tk.GetForbiddenAddresses(input.ctx, tokenTmpl.GetSymbol())
	require.Equal(t, 0, len(blackList)) // blacklist -> empty
	require.False(t, input.tk.GetToken(input.ctx, tokenTmpl.GetSymbol()).GetAddrForbiddable())
}

func testModifyTokenInfo(t *testing.T,
	tokenTmpl types.Token, distributed bool,
	modifyFn func(token types.Token) error,
	errMsg string) types.Token {

	input, token := issueTokenForTest(t, tokenTmpl, distributed)
	oldSupply := token.GetTotalSupply()

	require.NoError(t, modifyFn(token))
	newSupply := token.GetTotalSupply()

	err := input.tk.ModifyTokenInfo(input.ctx, token.GetSymbol(), token.GetOwner(),
		token.GetURL(), token.GetDescription(), token.GetIdentity(),
		token.GetName(), token.GetTotalSupply(),
		token.GetMintable(), token.GetBurnable(),
		token.GetAddrForbiddable(), token.GetTokenForbiddable())
	if errMsg == "" {
		require.NoError(t, err)

		newToken := input.tk.GetToken(input.ctx, token.GetSymbol())
		checkTokensEqual(t, token, newToken)

		if !oldSupply.Equal(newSupply) {
			require.Equal(t, newSupply,
				input.bkx.GetTotalCoins(input.ctx, token.GetOwner()).AmountOf(token.GetSymbol()))
		}

		return newToken
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), errMsg)
	return nil
}

func issueTokenForTest(t *testing.T, tokenTmpl types.Token, distributed bool) (testInput, types.Token) {
	input := createTestInput()
	err := input.tk.IssueToken(input.ctx, tokenTmpl.GetName(), tokenTmpl.GetSymbol(),
		tokenTmpl.GetTotalSupply(), tokenTmpl.GetOwner(),
		tokenTmpl.GetMintable(), tokenTmpl.GetBurnable(),
		tokenTmpl.GetAddrForbiddable(), tokenTmpl.GetTokenForbiddable(),
		tokenTmpl.GetURL(), tokenTmpl.GetDescription(), tokenTmpl.GetIdentity())
	require.NoError(t, err)

	totalSupply := tokenTmpl.GetTotalSupply()
	if distributed {
		totalSupply = totalSupply.SubRaw(100) // simulate transfer
	}

	if tokenTmpl.GetTotalBurn().GT(sdk.ZeroInt()) {
		err = input.tk.BurnToken(input.ctx, tokenTmpl.GetSymbol(), tokenTmpl.GetOwner(), tokenTmpl.GetTotalBurn())
		require.NoError(t, err)
		totalSupply = totalSupply.Sub(tokenTmpl.GetTotalBurn())
	}

	err = input.tk.SendCoinsFromAssetModuleToAccount(input.ctx, testAddr, types.NewTokenCoins(tokenTmpl.GetSymbol(), totalSupply))
	require.NoError(t, err)

	if tokenTmpl.GetTotalMint().GT(sdk.ZeroInt()) {
		err = input.tk.MintToken(input.ctx, tokenTmpl.GetSymbol(), tokenTmpl.GetOwner(), tokenTmpl.GetTotalMint())
		require.NoError(t, err)
	}
	if tokenTmpl.GetIsForbidden() {
		err = input.tk.ForbidToken(input.ctx, tokenTmpl.GetSymbol(), tokenTmpl.GetOwner())
		require.NoError(t, err)
	}

	token := input.tk.GetToken(input.ctx, tokenTmpl.GetSymbol())
	return input, token
}

func checkTokensEqual(t *testing.T, token, newToken types.Token) {
	require.Equal(t, token.GetURL(), newToken.GetURL())
	require.Equal(t, token.GetDescription(), newToken.GetDescription())
	require.Equal(t, token.GetIdentity(), newToken.GetIdentity())
	require.Equal(t, token.GetName(), newToken.GetName())
	require.Equal(t, token.GetTotalSupply(), newToken.GetTotalSupply())
	require.Equal(t, token.GetMintable(), newToken.GetMintable())
	require.Equal(t, token.GetBurnable(), newToken.GetBurnable())
	require.Equal(t, token.GetAddrForbiddable(), newToken.GetAddrForbiddable())
	require.Equal(t, token.GetTokenForbiddable(), newToken.GetTokenForbiddable())
}
