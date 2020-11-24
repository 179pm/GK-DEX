package cli

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coinexchain/cet-sdk/modules/asset/internal/types"
	dex "github.com/coinexchain/cet-sdk/types"
)

func TestMain(m *testing.M) {
	dex.InitSdkConfig()
	os.Exit(m.Run())
}
func TestAddGenesisToken(t *testing.T) {
	token := &types.BaseToken{}
	_ = token.SetName("aaa")
	_ = token.SetSymbol("aaa")

	genesis := types.GenesisState{
		Tokens: []types.Token{token},
	}
	err := addGenesisToken(&genesis, token)
	assert.Error(t, err)

	token = &types.BaseToken{}
	_ = token.SetName("bbb")
	_ = token.SetSymbol("bbb")
	_ = addGenesisToken(&genesis, token)
	require.Equal(t, token.GetSymbol(), genesis.Tokens[1].GetSymbol())
}

func TestParseTokenInfo(t *testing.T) {
	defer os.RemoveAll("./keys")
	_, err := parseTokenInfo()
	assert.Error(t, err)

	viper.Set(flagOwner, "owner")
	_, err = parseTokenInfo()
	assert.Error(t, err)

	viper.Set(flagOwner, "coinex1paehyhx9sxdfwc3rjf85vwn6kjnmzjemtedpnl")
	viper.Set(flagName, "1")
	_, err = parseTokenInfo()
	assert.Error(t, err)

	viper.Set(flagName, "aaa")
	viper.Set(flagSymbol, "1")
	_, err = parseTokenInfo()
	assert.Error(t, err)

	viper.Set(flagSymbol, "aaa")
	viper.Set(flagTotalSupply, "100")
	viper.Set(flagTotalBurn, "100")
	viper.Set(flagTotalMint, "100")
	viper.Set(flagTokenIdentity, "552A83BA62F9B1F8")
	token, _ := parseTokenInfo()
	require.Equal(t, "aaa", token.GetSymbol())
}
