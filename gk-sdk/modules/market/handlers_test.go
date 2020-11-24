package market

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/supply"

	"github.com/coinexchain/cet-sdk/modules/asset"
	"github.com/coinexchain/cet-sdk/modules/authx"
	"github.com/coinexchain/cet-sdk/modules/bankx"
	"github.com/coinexchain/cet-sdk/modules/market/internal/keepers"
	"github.com/coinexchain/cet-sdk/modules/market/internal/types"
	"github.com/coinexchain/cet-sdk/msgqueue"
	dex "github.com/coinexchain/cet-sdk/types"
)

type testInput struct {
	ctx     sdk.Context
	mk      keepers.Keeper
	handler sdk.Handler
	akp     auth.AccountKeeper
	keys    storeKeys
	cdc     *codec.Codec // mk.cdc
}

func (t testInput) getCoinFromAddr(addr sdk.AccAddress, denom string) (cetCoin sdk.Coin) {
	coins := t.akp.GetAccount(t.ctx, addr).GetCoins()
	for _, coin := range coins {
		if coin.Denom == denom {
			cetCoin = coin
			return
		}
	}
	return
}

func (t testInput) hasCoins(addr sdk.AccAddress, coins sdk.Coins) bool {
	coinsStore := t.akp.GetAccount(t.ctx, addr).GetCoins()
	if len(coinsStore) < len(coins) {
		return false
	}

	for _, coin := range coins {
		find := false
		for _, coinC := range coinsStore {
			if coinC.Denom == coin.Denom {
				find = true
				if coinC.IsEqual(coin) {
					break
				} else {
					return false
				}
			}
		}
		if !find {
			return false
		}
	}

	return true
}

var (
	haveCetAddress      sdk.AccAddress
	notHaveCetAddress   sdk.AccAddress
	forbidAddr          sdk.AccAddress
	stock                     = "tusdt"
	money                     = "teos"
	OriginHaveCetAmount int64 = 1e13
	issueAmount         int64 = 210000000000
	Bech32MainPrefix          = "coinex"
)

type storeKeys struct {
	assetCapKey *sdk.KVStoreKey
	authCapKey  *sdk.KVStoreKey
	authxCapKey *sdk.KVStoreKey
	keyParams   *sdk.KVStoreKey
	tkeyParams  *sdk.TransientStoreKey
	marketKey   *sdk.KVStoreKey
	authxKey    *sdk.KVStoreKey
	keyStaking  *sdk.KVStoreKey
	tkeyStaking *sdk.TransientStoreKey
	keySupply   *sdk.KVStoreKey
}

func initAddress() {
	haveCetAddress, _ = simpleAddr("00001")
	notHaveCetAddress, _ = simpleAddr("00002")
	forbidAddr, _ = simpleAddr("00003")
}

func prepareAssetKeeper(t *testing.T, keys storeKeys, cdc *codec.Codec, ctx sdk.Context, addrForbid, tokenForbid bool) (types.ExpectedAssetStatusKeeper, auth.AccountKeeper, authx.AccountXKeeper) {
	asset.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	supply.RegisterCodec(cdc)

	//create auth, asset keeper
	ak := auth.NewAccountKeeper(
		cdc,
		keys.authCapKey,
		params.NewKeeper(cdc, keys.keyParams, keys.tkeyParams, params.DefaultCodespace).Subspace(auth.DefaultParamspace), auth.ProtoBaseAccount,
	)
	bk := bank.NewBaseKeeper(
		ak,
		params.NewKeeper(cdc, keys.keyParams, keys.tkeyParams, params.DefaultCodespace).Subspace(bank.DefaultParamspace),
		sdk.CodespaceRoot, map[string]bool{},
	)

	// account permissions
	maccPerms := map[string][]string{
		auth.FeeCollectorName:     nil,
		authx.ModuleName:          nil,
		distr.ModuleName:          nil,
		staking.BondedPoolName:    {supply.Burner, supply.Staking},
		staking.NotBondedPoolName: {supply.Burner, supply.Staking},
		gov.ModuleName:            {supply.Burner},
		types.ModuleName:          nil,
		asset.ModuleName:          {supply.Minter},
	}
	sk := supply.NewKeeper(cdc, keys.keySupply, ak, bk, maccPerms)
	ak.SetAccount(ctx, supply.NewEmptyModuleAccount(authx.ModuleName))
	ak.SetAccount(ctx, supply.NewEmptyModuleAccount(asset.ModuleName, supply.Minter))
	sk.SetSupply(ctx, supply.Supply{Total: sdk.Coins{}})
	axk := authx.NewKeeper(
		cdc,
		keys.authxCapKey,
		params.NewKeeper(cdc, keys.keyParams, keys.tkeyParams, params.DefaultCodespace).Subspace(authx.DefaultParamspace),
		sk,
		ak,
		bk,
		"",
	)

	ask := asset.NewBaseTokenKeeper(
		cdc,
		keys.assetCapKey,
	)
	bkx := bankx.NewKeeper(
		params.NewKeeper(cdc, keys.keyParams, keys.tkeyParams, params.DefaultCodespace).Subspace(bankx.DefaultParamspace),
		axk, bk, ak, ask,
		sk,
		msgqueue.NewProducer(nil),
	)
	tk := asset.NewBaseKeeper(
		cdc,
		keys.assetCapKey,
		params.NewKeeper(cdc, keys.keyParams, keys.tkeyParams, params.DefaultCodespace).Subspace(asset.DefaultParamspace),
		bkx,
		sk,
	)
	tk.SetParams(ctx, asset.DefaultParams())

	// create an account by auth keeper
	cetacc := ak.NewAccountWithAddress(ctx, haveCetAddress)
	coins := dex.NewCetCoins(OriginHaveCetAmount).
		Add(sdk.NewCoins(sdk.NewCoin(stock, sdk.NewInt(issueAmount))))
	cetacc.SetCoins(coins)
	ak.SetAccount(ctx, cetacc)
	usdtacc := ak.NewAccountWithAddress(ctx, forbidAddr)
	usdtacc.SetCoins(sdk.NewCoins(sdk.NewCoin(stock, sdk.NewInt(issueAmount)),
		sdk.NewCoin(dex.CET, sdk.NewInt(issueAmount))))
	ak.SetAccount(ctx, usdtacc)
	onlyIssueToken := ak.NewAccountWithAddress(ctx, notHaveCetAddress)
	onlyIssueToken.SetCoins(dex.NewCetCoins(asset.DefaultIssue3CharTokenFee))
	ak.SetAccount(ctx, onlyIssueToken)

	// issue tokens
	msgStock := asset.NewMsgIssueToken(stock, stock, sdk.NewInt(issueAmount), haveCetAddress,
		false, false, addrForbid, tokenForbid, "", "", asset.TestIdentityString)
	msgMoney := asset.NewMsgIssueToken(money, money, sdk.NewInt(issueAmount), notHaveCetAddress,
		false, false, addrForbid, tokenForbid, "", "", asset.TestIdentityString)
	msgCet := asset.NewMsgIssueToken("cet", "cet", sdk.NewInt(issueAmount), haveCetAddress,
		false, false, addrForbid, tokenForbid, "", "", asset.TestIdentityString)
	handler := asset.NewHandler(tk)
	ret := handler(ctx, msgStock)
	require.Equal(t, true, ret.IsOK(), "issue token should succeed", ret)
	ret = handler(ctx, msgMoney)
	require.Equal(t, true, ret.IsOK(), "issue token should succeed", ret)
	ret = handler(ctx, msgCet)
	require.Equal(t, true, ret.IsOK(), "issue token should succeed", ret)

	if tokenForbid {
		msgForbidToken := asset.MsgForbidToken{
			Symbol:       stock,
			OwnerAddress: haveCetAddress,
		}
		tk.ForbidToken(ctx, msgForbidToken.Symbol, msgForbidToken.OwnerAddress)
	}
	if addrForbid {
		msgForbidAddr := asset.MsgForbidAddr{
			Symbol:    stock,
			OwnerAddr: haveCetAddress,
			Addresses: []sdk.AccAddress{forbidAddr},
		}
		tk.ForbidAddress(ctx, msgForbidAddr.Symbol, msgForbidAddr.OwnerAddr, msgForbidAddr.Addresses)
	}

	return tk, ak, axk
}

func prepareBankxKeeper(keys storeKeys, cdc *codec.Codec, ctx sdk.Context) types.ExpectedBankxKeeper {
	paramsKeeper := params.NewKeeper(cdc, keys.keyParams, keys.tkeyParams, params.DefaultCodespace)
	producer := msgqueue.NewProducer(nil)
	ak := auth.NewAccountKeeper(cdc, keys.authCapKey, paramsKeeper.Subspace(auth.StoreKey), auth.ProtoBaseAccount)

	bk := bank.NewBaseKeeper(ak, paramsKeeper.Subspace(bank.DefaultParamspace), sdk.CodespaceRoot, map[string]bool{})
	maccPerms := map[string][]string{
		auth.FeeCollectorName:     nil,
		authx.ModuleName:          nil,
		distr.ModuleName:          nil,
		staking.BondedPoolName:    {supply.Burner, supply.Staking},
		staking.NotBondedPoolName: {supply.Burner, supply.Staking},
		gov.ModuleName:            {supply.Burner},
		types.ModuleName:          nil,
		asset.ModuleName:          {supply.Minter},
	}
	sk := supply.NewKeeper(cdc, keys.keySupply, ak, bk, maccPerms)
	ak.SetAccount(ctx, supply.NewEmptyModuleAccount(authx.ModuleName))
	ak.SetAccount(ctx, supply.NewEmptyModuleAccount(asset.ModuleName, supply.Minter))

	axk := authx.NewKeeper(cdc, keys.authxKey, paramsKeeper.Subspace(authx.DefaultParamspace), sk, ak, bk, "")
	ask := asset.NewBaseTokenKeeper(cdc, keys.assetCapKey)
	bxkKeeper := bankx.NewKeeper(paramsKeeper.Subspace("bankx"), axk, bk, ak, ask, sk, producer)
	bk.SetSendEnabled(ctx, true)
	bxkKeeper.SetParams(ctx, bankx.DefaultParams())

	return bxkKeeper
}

func prepareMockInput(t *testing.T, addrForbid, tokenForbid bool) testInput {
	cdc := codec.New()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	initAddress()

	keys := storeKeys{}
	keys.marketKey = sdk.NewKVStoreKey(types.StoreKey)
	keys.assetCapKey = sdk.NewKVStoreKey(asset.StoreKey)
	keys.authCapKey = sdk.NewKVStoreKey(auth.StoreKey)
	keys.authxCapKey = sdk.NewKVStoreKey(authx.StoreKey)
	keys.keyParams = sdk.NewKVStoreKey(params.StoreKey)
	keys.tkeyParams = sdk.NewTransientStoreKey(params.TStoreKey)
	keys.authxKey = sdk.NewKVStoreKey(authx.StoreKey)
	keys.keyStaking = sdk.NewKVStoreKey(staking.StoreKey)
	keys.tkeyStaking = sdk.NewTransientStoreKey(staking.TStoreKey)
	keys.keySupply = sdk.NewKVStoreKey(supply.StoreKey)

	ms.MountStoreWithDB(keys.assetCapKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keys.authCapKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keys.keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keys.tkeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keys.marketKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keys.authxKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keys.keySupply, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "test-chain-id"}, false, log.NewNopLogger())
	ak, akp, _ := prepareAssetKeeper(t, keys, cdc, ctx, addrForbid, tokenForbid)
	bk := prepareBankxKeeper(keys, cdc, ctx)
	paramsKeeper := params.NewKeeper(cdc, keys.keyParams, keys.tkeyParams, params.DefaultCodespace)
	mk := keepers.NewKeeper(keys.marketKey, ak, bk, cdc,
		msgqueue.NewProducer(nil), paramsKeeper.Subspace(types.StoreKey), akp, &mockKeeper{})
	types.RegisterCodec(cdc)

	parameters := types.DefaultParams()
	mk.SetParams(ctx, parameters)

	return testInput{ctx: ctx, mk: mk, handler: NewHandler(mk), akp: akp, keys: keys, cdc: cdc}
}

func TestMarketInfoSetFailed(t *testing.T) {
	input := prepareMockInput(t, false, true)
	remainCoin := dex.NewCetCoin(OriginHaveCetAmount + issueAmount - asset.DefaultIssue4CharTokenFee*2 - asset.DefaultIssue5CharTokenFee)
	require.Equal(t, true, input.hasCoins(haveCetAddress, sdk.Coins{remainCoin}), "The amount is error")

	msgMarket := types.MsgCreateTradingPair{
		Stock:          stock,
		Money:          money,
		Creator:        haveCetAddress,
		PricePrecision: 8,
	}

	// failed by token not exist
	failedToken := msgMarket
	failedToken.Money = "tbtc"
	ret := input.handler(input.ctx, failedToken)
	require.Equal(t, types.CodeInvalidToken, ret.Code, "create market info should failed by token not exist")
	require.Equal(t, true, input.hasCoins(haveCetAddress, sdk.Coins{remainCoin}), "The amount is error")

	failedToken.Stock = "tiota"
	failedToken.Money = money
	ret = input.handler(input.ctx, failedToken)
	require.Equal(t, types.CodeInvalidToken, ret.Code, "create market info should failed by token not exist")
	require.Equal(t, true, input.hasCoins(haveCetAddress, sdk.Coins{remainCoin}), "The amount is error")

	// failed by not token issuer
	failedTokenIssuer := msgMarket
	addr, _ := simpleAddr("00008")
	failedTokenIssuer.Creator = addr
	ret = input.handler(input.ctx, failedTokenIssuer)
	require.Equal(t, types.CodeInvalidTokenIssuer, ret.Code, "create market info should failed by not token issuer")
	require.Equal(t, true, input.hasCoins(haveCetAddress, sdk.Coins{remainCoin}), "The amount is error")

	// failed by not have sufficient cet
	parameters := types.DefaultParams()
	parameters.CreateMarketFee = 1e12
	input.mk.SetParams(input.ctx, parameters)
	failedInsufficient := msgMarket
	failedInsufficient.Creator = notHaveCetAddress
	failedInsufficient.Money = "cet"
	failedInsufficient.Stock = money
	ret = input.handler(input.ctx, failedInsufficient)
	require.Equal(t, types.CodeInsufficientCoin, ret.Code, "create market info should failed")
	require.Equal(t, true, input.hasCoins(haveCetAddress, sdk.Coins{remainCoin}), "The amount is error")

	// failed by not have cet trade
	failedNotHaveCetTrade := msgMarket
	ret = input.handler(input.ctx, failedNotHaveCetTrade)
	require.EqualValues(t, sdk.CodeOK, ret.Code)
	remainCoin = remainCoin.Sub(dex.NewCetCoin(parameters.CreateMarketFee))
	require.Equal(t, true, input.hasCoins(haveCetAddress, sdk.Coins{remainCoin}), "The amount is error")
}

func createMarket(input testInput) sdk.Result {
	return createImpMarket(input, stock, money, 0)
}

func createImpMarket(input testInput, stock, money string, orderPrecision byte) sdk.Result {
	msgMarketInfo := types.MsgCreateTradingPair{Stock: stock, Money: money, Creator: haveCetAddress, PricePrecision: 8, OrderPrecision: orderPrecision}
	return input.handler(input.ctx, msgMarketInfo)
}

func createCetMarket(input testInput, stock string, orderPrecision byte) sdk.Result {
	return createImpMarket(input, stock, dex.CET, orderPrecision)
}

func IsEqual(old, new sdk.Coin, diff sdk.Coin) bool {
	return old.IsEqual(new.Add(diff))
}

func TestMarketInfoSetSuccess(t *testing.T) {
	for i := 0; i <= 10; i++ {
		input := prepareMockInput(t, true, true)
		oldCetCoin := input.getCoinFromAddr(haveCetAddress, dex.CET)
		params := input.mk.GetParams(input.ctx)

		ret := createCetMarket(input, stock, byte(i))
		newCetCoin := input.getCoinFromAddr(haveCetAddress, dex.CET)
		require.Equal(t, true, ret.IsOK(), "create market info should succeed")
		require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, dex.NewCetCoin(params.CreateMarketFee)), "The amount is error")
		info, err := input.mk.GetMarketInfo(input.ctx, GetSymbol(stock, dex.CET))
		require.Nil(t, err)
		if i <= int(types.MaxOrderPrecision) {
			require.EqualValues(t, i, info.OrderPrecision)
		} else {
			require.EqualValues(t, 0, info.OrderPrecision)
		}

		for i := 0; i <= 9; i++ {
			ret = createCetMarket(input, stock, byte(i))
			require.Equal(t, types.CodeRepeatTradingPair, ret.Code)
			require.Equal(t, false, ret.IsOK(), "repeatedly creating market would fail")
		}
	}
}

func TestCreateOrderFailed(t *testing.T) {
	input := prepareMockInput(t, false, true)
	msgOrder := types.MsgCreateOrder{
		Sender:         haveCetAddress,
		TradingPair:    GetSymbol(stock, money),
		OrderType:      types.LimitOrder,
		PricePrecision: 8,
		Price:          100,
		Quantity:       10000000,
		Side:           types.SELL,
		TimeInForce:    types.GTE,
	}
	ret := createCetMarket(input, stock, 1)
	require.Equal(t, true, ret.IsOK(), "create market trade should success")
	ret = createMarket(input)
	require.Equal(t, true, ret.IsOK(), "create market trade should success")
	zeroCet := sdk.NewCoin("cet", sdk.NewInt(0))
	newCetCoin := input.getCoinFromAddr(haveCetAddress, dex.CET)

	failedPricePrecisionOrder := msgOrder
	failedPricePrecisionOrder.PricePrecision = 9
	ret = input.handler(input.ctx, failedPricePrecisionOrder)
	oldCetCoin := input.getCoinFromAddr(haveCetAddress, dex.CET)
	require.Equal(t, types.CodeInvalidPricePrecision, ret.Code, "create GTE order should failed by invalid price precision")
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, zeroCet), "The amount is error")

	failedInsufficientCoinOrder := msgOrder
	failedInsufficientCoinOrder.Quantity = issueAmount * 10
	ret = input.handler(input.ctx, failedInsufficientCoinOrder)
	oldCetCoin = input.getCoinFromAddr(haveCetAddress, dex.CET)
	require.Equal(t, types.CodeInsufficientCoin, ret.Code, "create GTE order should failed by insufficient coin")
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, zeroCet), "The amount is error")

	failedTokenForbidOrder := msgOrder
	ret = input.handler(input.ctx, failedTokenForbidOrder)
	oldCetCoin = input.getCoinFromAddr(haveCetAddress, dex.CET)
	require.Equal(t, types.CodeTokenForbidByIssuer, ret.Code, "create GTE order should failed by token forbidden by issuer")
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, zeroCet), "The amount is error")

	input = prepareMockInput(t, true, false)
	ret = createCetMarket(input, stock, 0)
	require.Equal(t, true, ret.IsOK(), "create market failed")
	ret = createMarket(input)
	require.Equal(t, true, ret.IsOK(), "create market failed")

	failedAddrForbidOrder := msgOrder
	failedAddrForbidOrder.Sender = forbidAddr
	newCetCoin = input.getCoinFromAddr(haveCetAddress, dex.CET)
	ret = input.handler(input.ctx, failedAddrForbidOrder)
	oldCetCoin = input.getCoinFromAddr(haveCetAddress, dex.CET)
	require.Equal(t, types.CodeAddressForbidByIssuer, ret.Code, "create GTE order should failed by token forbidden by issuer")
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, zeroCet), "The amount is error")

	failedMaxAmount := msgOrder
	failedMaxAmount.Side = SELL
	failedMaxAmount.Quantity = 1e18 * 5
	ret = input.handler(input.ctx, failedMaxAmount)
	require.Equal(t, types.CodeInvalidOrderAmount, ret.Code, "create GTE order should failed by token forbidden by issuer")
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, zeroCet), "The amount is error")

	ret = input.handler(input.ctx, msgOrder)
	require.Equal(t, true, ret.IsOK(), "create order should succeed")

	failedOrderHaveExist := msgOrder
	newCetCoin = input.getCoinFromAddr(haveCetAddress, dex.CET)
	ret = input.handler(input.ctx, failedOrderHaveExist)
	oldCetCoin = input.getCoinFromAddr(haveCetAddress, dex.CET)
	require.Equal(t, types.CodeOrderAlreadyExist, ret.Code, "create order should failed by order exist")
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, zeroCet), "The amount is error")
}

func TestCalculateAmount(t *testing.T) {
	// price quantity price-precision
	items := [][]int64{{100, 10000, 2}, {300, 2000, 3}, {500, 4500, 2}}
	results := []int64{10000, 600, 22500}
	for i, item := range items {
		ret, _ := calculateAmount(item[0], item[1], byte(item[2]))
		if ret.RoundInt64() != results[i] {
			t.Errorf("amount is error, actual : %d, expect : %d", ret.RoundInt64(), results[i])
		}
	}

	for i := 2; i <= 5; i++ {
		_, err := calculateAmount(math.MaxInt64, int64(i), 0)
		require.NotNil(t, err)
	}
}

func TestCreateOrderFiledByOrderPrecision(t *testing.T) {
	for i := 1; i <= 8; i++ {
		input := prepareMockInput(t, false, false)
		msgGteOrder := types.MsgCreateOrder{
			Sender:         haveCetAddress,
			Identify:       1,
			TradingPair:    stock + types.SymbolSeparator + "cet",
			OrderType:      types.LimitOrder,
			PricePrecision: 8,
			Price:          100,
			Quantity:       10000000,
			Side:           types.SELL,
			TimeInForce:    types.GTE,
		}

		ret := createCetMarket(input, stock, byte(i))
		require.Equal(t, true, ret.IsOK(), "create market should succeed")
		failedorderPrecision := msgGteOrder
		for j := 1; j <= 8; j++ {
			failedorderPrecision.Quantity = int64(rand.Intn(int(math.Pow10(i)) - 1))
			if failedorderPrecision.Quantity == 0 {
				failedorderPrecision.Quantity = 1
			}
			failedorderPrecision.TradingPair = stock + types.SymbolSeparator + dex.CET
			ret = input.handler(input.ctx, failedorderPrecision)
			require.Equal(t, false, ret.IsOK(), "create GTE order should failed")
			require.Equal(t, types.CodeInvalidOrderAmount, ret.Code, "invalid order amount, must be a multiple of granularity ")
		}
	}
}

func TestCreateOrderSuccess(t *testing.T) {
	input := prepareMockInput(t, false, false)
	msgGteOrder := types.MsgCreateOrder{
		Sender:         haveCetAddress,
		Identify:       1,
		TradingPair:    GetSymbol(stock, "cet"),
		OrderType:      types.LimitOrder,
		PricePrecision: 8,
		Price:          100,
		Quantity:       10000000,
		Side:           types.SELL,
		TimeInForce:    types.GTE,
	}
	ret := createCetMarket(input, stock, 10)
	require.Equal(t, true, ret.IsOK(), "create market should succeed")

	seq, err := input.mk.QuerySeqWithAddr(input.ctx, msgGteOrder.Sender)
	require.Equal(t, nil, err)
	oldCoin := input.getCoinFromAddr(haveCetAddress, stock)
	ret = input.handler(input.ctx, msgGteOrder)
	newCoin := input.getCoinFromAddr(haveCetAddress, stock)
	frozenMoney := sdk.NewCoin(stock, sdk.NewInt(msgGteOrder.Quantity))
	require.Equal(t, true, ret.IsOK(), "create GTE order should succeed")
	require.Equal(t, true, IsEqual(oldCoin, newCoin, frozenMoney), "The amount is error")

	glk := keepers.NewGlobalOrderKeeper(input.keys.marketKey, input.cdc)
	orderID := types.AssemblyOrderID(msgGteOrder.Sender.String(), seq, msgGteOrder.Identify)
	order := glk.QueryOrder(input.ctx, orderID)
	require.Equal(t, true, isSameOrderAndMsg(order, msgGteOrder), "order should equal msg")

	msgIOCOrder := types.MsgCreateOrder{
		Sender:         haveCetAddress,
		Identify:       2,
		TradingPair:    GetSymbol(stock, "cet"),
		OrderType:      types.LimitOrder,
		PricePrecision: 8,
		Price:          300,
		Quantity:       68293762,
		Side:           types.BUY,
		TimeInForce:    types.IOC,
	}

	seq, err = input.mk.QuerySeqWithAddr(input.ctx, msgGteOrder.Sender)
	require.Equal(t, nil, err)
	oldCoin = input.getCoinFromAddr(haveCetAddress, dex.CET)
	ret = input.handler(input.ctx, msgIOCOrder)
	newCoin = input.getCoinFromAddr(haveCetAddress, dex.CET)
	frozen, _ := calculateAmount(msgIOCOrder.Price, msgIOCOrder.Quantity, msgIOCOrder.PricePrecision)
	frozenMoney = sdk.NewCoin(dex.CET, frozen.RoundInt())
	frozenFee, err := calOrderCommission(input.ctx, input.mk, msgIOCOrder)
	require.Nil(t, err)
	totalFrozen := frozenMoney.Add(sdk.Coin{Denom: dex.CET, Amount: sdk.NewInt(frozenFee)})
	require.Equal(t, true, ret.IsOK(), "create Ioc order should succeed ; ", ret.Log)
	require.Equal(t, true, IsEqual(oldCoin, newCoin, totalFrozen), "The amount is error")

	orderID = types.AssemblyOrderID(msgIOCOrder.Sender.String(), seq, msgIOCOrder.Identify)
	order = glk.QueryOrder(input.ctx, orderID)
	require.Equal(t, true, isSameOrderAndMsg(order, msgIOCOrder), "order should equal msg")
}

func isSameOrderAndMsg(order *types.Order, msg types.MsgCreateOrder) bool {
	p := sdk.NewDec(msg.Price).Quo(sdk.NewDec(int64(math.Pow10(int(msg.PricePrecision)))))
	samePrice := order.Price.Equal(p)
	return bytes.Equal(order.Sender, msg.Sender) && order.TradingPair ==
		msg.TradingPair && order.OrderType == msg.OrderType && samePrice &&
		order.Quantity == msg.Quantity && order.Side == msg.Side &&
		order.TimeInForce == msg.TimeInForce
}

func TestCancelOrderFailed(t *testing.T) {
	input := prepareMockInput(t, false, false)
	createCetMarket(input, stock, 0)
	cancelOrder := types.MsgCancelOrder{
		Sender: haveCetAddress,
	}

	failedInvalidOrderID := cancelOrder
	failedInvalidOrderID.OrderID = types.AssemblyOrderID(haveCetAddress.String(), 1, 2)
	ret := input.handler(input.ctx, failedInvalidOrderID)
	require.Equal(t, types.CodeOrderNotFound, ret.Code, "cancel order should failed by not exist ")

	// create order
	msgIOCOrder := types.MsgCreateOrder{
		Sender:         haveCetAddress,
		Identify:       1,
		TradingPair:    GetSymbol(stock, "cet"),
		OrderType:      types.LimitOrder,
		PricePrecision: 8,
		Price:          300,
		Quantity:       68293762,
		Side:           types.BUY,
		TimeInForce:    types.IOC,
	}
	ret = input.handler(input.ctx, msgIOCOrder)
	require.Equal(t, true, ret.IsOK(), "create Ioc order should succeed ; ", ret.Log)

	seq, err := input.mk.QuerySeqWithAddr(input.ctx, msgIOCOrder.Sender)
	require.Equal(t, nil, err)
	failedNotOrderSender := cancelOrder
	failedNotOrderSender.OrderID = types.AssemblyOrderID(msgIOCOrder.Sender.String(), seq, msgIOCOrder.Identify)
	failedNotOrderSender.Sender = notHaveCetAddress
	ret = input.handler(input.ctx, failedNotOrderSender)
	require.Equal(t, types.CodeNotMatchSender, ret.Code, "cancel order should failed by not match order sender")
}

func TestCancelOrderSuccess(t *testing.T) {
	input := prepareMockInput(t, false, false)
	createCetMarket(input, stock, 0)

	// create order
	msgIOCOrder := types.MsgCreateOrder{
		Sender:         haveCetAddress,
		Identify:       2,
		TradingPair:    GetSymbol(stock, "cet"),
		OrderType:      types.LimitOrder,
		PricePrecision: 8,
		Price:          300,
		Quantity:       68293762,
		Side:           types.BUY,
		TimeInForce:    types.IOC,
	}
	seq, err := input.mk.QuerySeqWithAddr(input.ctx, msgIOCOrder.Sender)
	require.Equal(t, nil, err)
	ret := input.handler(input.ctx, msgIOCOrder)
	require.Equal(t, true, ret.IsOK(), "create Ioc order should succeed ; ", ret.Log)

	cancelOrder := types.MsgCancelOrder{
		Sender: haveCetAddress,
	}
	cancelOrder.OrderID = types.AssemblyOrderID(msgIOCOrder.Sender.String(), seq, msgIOCOrder.Identify)
	ret = input.handler(input.ctx, cancelOrder)
	require.Equal(t, true, ret.IsOK(), "cancel order should succeed ; ", ret.Log)

	remainCoin := sdk.NewCoin(money, sdk.NewInt(issueAmount))
	require.Equal(t, true, input.hasCoins(notHaveCetAddress, sdk.Coins{remainCoin}), "The amount is error ")
}

func TestCancelMarketFailed(t *testing.T) {
	input := prepareMockInput(t, false, false)
	createCetMarket(input, stock, 0)

	now := time.Now()
	msgCancelMarket := types.MsgCancelTradingPair{
		Sender:        haveCetAddress,
		TradingPair:   GetSymbol(stock, "cet"),
		EffectiveTime: now.UnixNano() + int64(types.DefaultMarketMinExpiredTime),
	}

	header := abci.Header{Time: now, Height: 10}
	input.ctx = input.ctx.WithBlockHeader(header)
	failedTime := msgCancelMarket
	failedTime.EffectiveTime = 10
	ret := input.handler(input.ctx, failedTime)
	require.Equal(t, types.CodeInvalidCancelTime, ret.Code, "cancel order should failed by invalid cancel time")

	failedSymbol := msgCancelMarket
	failedSymbol.TradingPair = GetSymbol(stock, "not exist")
	ret = input.handler(input.ctx, failedSymbol)
	require.Equal(t, types.CodeInvalidMarket, ret.Code, "cancel order should failed by invalid symbol")

	failedSender := msgCancelMarket
	failedSender.Sender = notHaveCetAddress
	ret = input.handler(input.ctx, failedSender)
	require.Equal(t, types.CodeNotMatchSender, ret.Code, "cancel order should failed by not match sender")

	failedByNotForbidden := msgCancelMarket
	ret = input.handler(input.ctx, failedByNotForbidden)
	require.EqualValues(t, sdk.CodeOK, ret.Code)
}

func TestCancelMarketSuccess(t *testing.T) {
	input := prepareMockInput(t, false, true)
	createCetMarket(input, stock, 0)

	msgCancelMarket := types.MsgCancelTradingPair{
		Sender:        haveCetAddress,
		TradingPair:   GetSymbol(stock, "cet"),
		EffectiveTime: int64(types.DefaultMarketMinExpiredTime + 10),
	}

	ret := input.handler(input.ctx, msgCancelMarket)
	require.Equal(t, true, ret.IsOK(), "cancel market should success")

	msgCancelMarket = types.MsgCancelTradingPair{
		Sender:        haveCetAddress,
		TradingPair:   GetSymbol(stock, "cet"),
		EffectiveTime: int64(types.DefaultMarketMinExpiredTime + 10),
	}

	ret = input.handler(input.ctx, msgCancelMarket)
	require.Equal(t, false, ret.IsOK(), "repeatedly cancel market will fail")
	require.EqualValues(t, types.CodeDelistRequestExist, ret.Code)

	dlk := keepers.NewDelistKeeper(input.keys.marketKey)
	delSymbol := dlk.GetDelistSymbolsBeforeTime(input.ctx, int64(types.DefaultMarketMinExpiredTime+10+1))[0]
	require.EqualValues(t, delSymbol, GetSymbol(stock, dex.CET))
}

func TestCancelMarketAgainstCetFail(t *testing.T) {
	input := prepareMockInput(t, false, true)
	createCetMarket(input, stock, 0)
	createImpMarket(input, stock, money, 10)

	msgCancelMarket := types.MsgCancelTradingPair{
		Sender:        haveCetAddress,
		TradingPair:   GetSymbol(stock, "cet"),
		EffectiveTime: int64(types.DefaultMarketMinExpiredTime + 10),
	}

	ret := input.handler(input.ctx, msgCancelMarket)
	require.EqualValues(t, ret.Code, sdk.CodeOK)
}

func TestCancelMarketFailWhenCetDelist(t *testing.T) {
	input := prepareMockInput(t, false, true)
	createCetMarket(input, stock, 0)

	msgCancelMarket := types.MsgCancelTradingPair{
		Sender:        haveCetAddress,
		TradingPair:   GetSymbol(stock, "cet"),
		EffectiveTime: int64(types.DefaultMarketMinExpiredTime + 10),
	}

	ret := input.handler(input.ctx, msgCancelMarket)
	require.Equal(t, true, ret.IsOK(), "cancel market should success")

	msg := MsgCreateTradingPair{
		Creator:        haveCetAddress,
		Stock:          stock,
		Money:          money,
		PricePrecision: 8,
		OrderPrecision: 8,
	}
	ret = input.handler(input.ctx, msg)
	require.EqualValues(t, sdk.CodeOK, ret.Code)
}

func TestChargeOrderFee(t *testing.T) {
	input := prepareMockInput(t, false, false)
	ret := createCetMarket(input, stock, 0)
	require.Equal(t, true, ret.IsOK(), "create market should success")

	msgOrder := types.MsgCreateOrder{
		Sender:         haveCetAddress,
		Identify:       1,
		TradingPair:    GetSymbol(stock, dex.CET),
		OrderType:      types.LimitOrder,
		PricePrecision: 8,
		Price:          300,
		Quantity:       100000000000,
		Side:           types.BUY,
		TimeInForce:    types.IOC,
	}

	// charge fix trade fee, because the stock/cet LastExecutedPrice is zero.
	oldCetCoin := input.getCoinFromAddr(msgOrder.Sender, dex.CET)
	ret = input.handler(input.ctx, msgOrder)
	newCetCoin := input.getCoinFromAddr(msgOrder.Sender, dex.CET)
	frozen, _ := calculateAmount(msgOrder.Price, msgOrder.Quantity, msgOrder.PricePrecision)
	frozeCoin := dex.NewCetCoin(frozen.RoundInt64())
	frozeFee, err := calOrderCommission(input.ctx, input.mk, msgOrder)
	require.Nil(t, err)
	totalFreeze := frozeCoin.Add(sdk.Coin{Denom: dex.CET, Amount: sdk.NewInt(frozeFee)})
	require.Equal(t, true, ret.IsOK(), "create Ioc order should succeed ; ", ret.Log)
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, totalFreeze), "The amount is error ")

	// If stock is cet symbol, Charge a percentage of the transaction fee,
	ret = createImpMarket(input, dex.CET, stock, 0)
	require.Equal(t, true, ret.IsOK(), "create market should success")
	stockIsCetOrder := msgOrder
	stockIsCetOrder.Identify = 2
	stockIsCetOrder.TradingPair = GetSymbol(dex.CET, stock)
	oldCetCoin = input.getCoinFromAddr(msgOrder.Sender, dex.CET)
	ret = input.handler(input.ctx, stockIsCetOrder)
	require.EqualValues(t, sdk.CodeOK, ret.Code)
	newCetCoin = input.getCoinFromAddr(msgOrder.Sender, dex.CET)
	frozeFee, err = calOrderCommission(input.ctx, input.mk, stockIsCetOrder)
	require.Nil(t, err)
	require.Equal(t, true, ret.IsOK(), "create Ioc order should succeed ; ", ret.Log)
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, dex.NewCetCoin(frozeFee)), "The amount is error ")

	marketInfo, fail := input.mk.GetMarketInfo(input.ctx, msgOrder.TradingPair)
	require.Equal(t, nil, fail, "get %s market failed", msgOrder.TradingPair)
	marketInfo.LastExecutedPrice = sdk.NewDec(12)
	err = input.mk.SetMarket(input.ctx, marketInfo)
	require.Equal(t, nil, err, "set %s market failed", msgOrder.TradingPair)

	// Freeze fee at market execution prices
	msgOrder.Identify = 3
	oldCetCoin = input.getCoinFromAddr(msgOrder.Sender, dex.CET)
	ret = input.handler(input.ctx, msgOrder)
	newCetCoin = input.getCoinFromAddr(msgOrder.Sender, dex.CET)
	frozeFee, err = calOrderCommission(input.ctx, input.mk, msgOrder)
	require.Nil(t, err)
	totalFreeze = dex.NewCetCoin(frozeFee).Add(frozeCoin)
	require.Equal(t, true, ret.IsOK(), "create Ioc order should succeed ; ", ret.Log)
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, totalFreeze), "The amount is error ")
}

func TestModifyPricePrecisionFaild(t *testing.T) {
	input := prepareMockInput(t, false, false)
	createCetMarket(input, stock, 0)

	msg := types.MsgModifyPricePrecision{
		Sender:         haveCetAddress,
		TradingPair:    GetSymbol(stock, dex.CET),
		PricePrecision: 12,
	}

	msgFailedBySender := msg
	msgFailedBySender.Sender = notHaveCetAddress
	ret := input.handler(input.ctx, msgFailedBySender)
	require.Equal(t, types.CodeNotMatchSender, ret.Code, "the tx should failed by dis match sender")
}

func TestModifyPricePrecisionSuccess(t *testing.T) {
	input := prepareMockInput(t, false, false)
	createCetMarket(input, stock, 0)

	msg := types.MsgModifyPricePrecision{
		Sender:         haveCetAddress,
		TradingPair:    GetSymbol(stock, dex.CET),
		PricePrecision: 12,
	}

	oldCetCoin := input.getCoinFromAddr(haveCetAddress, dex.CET)
	ret := input.handler(input.ctx, msg)
	newCetCoin := input.getCoinFromAddr(haveCetAddress, dex.CET)
	require.Equal(t, true, ret.IsOK(), "the tx should success")
	require.Equal(t, true, IsEqual(oldCetCoin, newCetCoin, sdk.NewCoin(dex.CET, sdk.NewInt(0))), "the amount is error")
}

func TestGetGranularityOfOrder(t *testing.T) {
	var expectValue = []float64{math.Pow10(0), math.Pow10(1), math.Pow10(2),
		math.Pow10(3), math.Pow10(4), math.Pow10(5), math.Pow10(6),
		math.Pow10(7), math.Pow10(8), math.Pow10(0)}
	for i := 0; i <= 9; i++ {
		ret := types.GetGranularityOfOrder(byte(i))
		require.EqualValues(t, ret, expectValue[i])
	}
}

func TestCalFeatureFeeForExistBlocks(t *testing.T) {
	msg := types.MsgCreateOrder{
		ExistBlocks: 8000,
	}
	params := types.Params{
		GTEOrderLifetime:           10000,
		GTEOrderFeatureFeeByBlocks: 1,
	}
	fee := calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(0), fee)

	msg.ExistBlocks = 10000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(0), fee)

	msg.ExistBlocks = 10001
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(1), fee)

	msg.ExistBlocks = 18000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(8000), fee)

	msg.ExistBlocks = 20000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(10000), fee)

	msg.ExistBlocks = 20001
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(10001), fee)

	msg.ExistBlocks = 28000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(18000), fee)

	msg.ExistBlocks = 30000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(20000), fee)

	msg.ExistBlocks = 30001
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(20001), fee)
	//
	params = types.Params{
		GTEOrderLifetime:           10000,
		GTEOrderFeatureFeeByBlocks: 10,
	}
	msg.ExistBlocks = 8000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(0), fee)

	msg.ExistBlocks = 10000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(0), fee)

	msg.ExistBlocks = 10001
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(10), fee)

	msg.ExistBlocks = 18000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(80000), fee)

	msg.ExistBlocks = 20000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(100000), fee)

	msg.ExistBlocks = 20001
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(100010), fee)

	msg.ExistBlocks = 28000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(180000), fee)

	msg.ExistBlocks = 30000
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(200000), fee)

	msg.ExistBlocks = 30001
	fee = calFeatureFeeForExistBlocks(msg, params)
	require.Equal(t, int64(200010), fee)
}

func TestCalOrderCommission(t *testing.T) {
	input := prepareMockInput(t, false, false)
	param := types.Params{MarketFeeRate: 10, MarketFeeMin: 21}
	input.mk.SetParams(input.ctx, param)

	// Stock is CET, commission = quantity * rate
	orderInfo := MsgCreateOrder{
		Price:          1,
		Quantity:       10000,
		PricePrecision: 4,
		TradingPair:    GetSymbol(dex.CET, money),
	}

	// cet/money; commission < MarketFeeMin
	cal, err := calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, param.MarketFeeMin, cal)

	// cet/money; commission > MarketFeeMin
	orderInfo.Quantity = 30000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, 30, cal)

	// Money is CET, commission = Quantity * Price * rate
	orderInfo.TradingPair = GetSymbol(stock, dex.CET)
	orderInfo.Quantity = 10000

	// commission < MarketFeeMin
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, param.MarketFeeMin, cal)

	// commission < MarketFeeMin
	orderInfo.Price = 3
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, param.MarketFeeMin, cal)

	// commission > MarketFeeMin
	orderInfo.Price = 3000
	orderInfo.Quantity = 100000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, 30, cal)

	// create necessary market
	mkInfo := MarketInfo{
		Stock:             stock,
		Money:             money,
		LastExecutedPrice: sdk.NewDec(0),
		PricePrecision:    1,
		OrderPrecision:    3,
	}

	// cet/money
	mkInfo.Stock = dex.CET
	mkInfo.LastExecutedPrice = sdk.NewDec(2)
	err = input.mk.SetMarket(input.ctx, mkInfo)
	require.Nil(t, err)

	// cet/stock
	mkInfo.Money = stock
	mkInfo.LastExecutedPrice = sdk.NewDec(3)
	err = input.mk.SetMarket(input.ctx, mkInfo)
	require.Nil(t, err)

	// money/cet
	mkInfo.Stock = money
	mkInfo.Money = dex.CET
	mkInfo.LastExecutedPrice = sdk.NewDec(4)
	err = input.mk.SetMarket(input.ctx, mkInfo)
	require.Nil(t, err)

	// stock/cet
	mkInfo.Stock = stock
	mkInfo.LastExecutedPrice = sdk.NewDec(5)
	err = input.mk.SetMarket(input.ctx, mkInfo)
	require.Nil(t, err)

	// When CET/money exist, and cet/money lastPrice = 2; commission = quantity * price / lastPrice
	orderInfo.TradingPair = GetSymbol(stock, money)
	orderInfo.Quantity = 10000
	orderInfo.Price = 2

	// commission < MarketFeeMin; 10000*2/2/price_precision*0.1% < 21
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, param.MarketFeeMin, cal)

	// commission > MarketFeeMin; 40000 * 2 / 2 /4 * 0.1% > 21
	orderInfo.Quantity = 400000000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, 40, cal)

	// del cet/money market
	err = input.mk.RemoveMarket(input.ctx, GetSymbol(dex.CET, money))
	require.Nil(t, err)

	// When CET/stock exist, and cet/stock lastPrice = 3; commission = quantity / lastPrice * 0.1%
	// commission < MarketFeeMin; 30000 / 3 * 0.1% < 21
	orderInfo.Quantity = 30000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, param.MarketFeeMin, cal)

	// commission > MarketFeeMin; 90000 / 3 * 0.1% < 21
	orderInfo.Quantity = 90000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, 30, cal)

	// del cet/stock market
	err = input.mk.RemoveMarket(input.ctx, GetSymbol(dex.CET, stock))
	require.Nil(t, err)

	// When money/cet exist, and money/cet lastPrice = 4; commission = quantity * price / price_precision * lastPrice * 0.1%
	// commission < MarketFeeMin; 2000 * 2 / 4 * 4 * 0.1% < 21
	orderInfo.Quantity = 20000000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, 21, cal)

	// commission < MarketFeeMin; 10000 * 2 / 4 * 4 * 0.1% > 21
	orderInfo.Quantity = 100000000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, 80, cal)

	// del monet/cet market
	err = input.mk.RemoveMarket(input.ctx, GetSymbol(money, dex.CET))
	require.Nil(t, err)

	// When stock/cet exist, and stock/cet lastPrice = 5; commission = quantity * lastPrice
	// commission < MarketFeeMin; 2000 * 5 * 0.1% < 21
	orderInfo.Quantity = 2000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, param.MarketFeeMin, cal)

	// commission > MarketFeeMin; 1000 * 5 * 0.1% > 21
	orderInfo.Quantity = 10000
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, 50, cal)

	// del stock/cet market
	err = input.mk.RemoveMarket(input.ctx, GetSymbol(stock, dex.CET))
	require.Nil(t, err)

	// commission must equal MarketFeeMin;
	orderInfo.Quantity = 100000000
	orderInfo.Price = 49
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, param.MarketFeeMin, cal)

	orderInfo.Quantity = 1
	orderInfo.Price = 1
	cal, err = calOrderCommission(input.ctx, input.mk, orderInfo)
	require.Nil(t, err)
	require.EqualValues(t, param.MarketFeeMin, cal)
}

func TestCheckMsgCreateOrder(t *testing.T) {
	input := prepareMockInput(t, true, true)
	require.True(t, input.mk.IsTokenForbidden(input.ctx, stock))
	require.True(t, input.mk.IsForbiddenByTokenIssuer(input.ctx, stock, forbidAddr))
	remain := OriginHaveCetAmount + issueAmount - asset.DefaultIssue4CharTokenFee*2 - asset.DefaultIssue5CharTokenFee
	remainCoin := dex.NewCetCoin(remain)
	require.Equal(t, true, input.hasCoins(haveCetAddress, sdk.Coins{remainCoin}), "The amount is error")

	// Insufficient coin
	msg := MsgCreateOrder{
		Sender:         haveCetAddress,
		Identify:       255,
		TradingPair:    GetSymbol(stock, dex.CET),
		OrderType:      LimitOrder,
		Side:           BUY,
		Price:          10,
		PricePrecision: 8,
		Quantity:       100,
		TimeInForce:    GTE,
		ExistBlocks:    10000,
	}
	err := checkMsgCreateOrder(input.ctx, input.mk, msg, remain+1, 1, dex.CET, 1)
	require.EqualValues(t, err.Code(), types.CodeInsufficientCoin)

	err = checkMsgCreateOrder(input.ctx, input.mk, msg, issueAmount, OriginHaveCetAmount, dex.CET, 1)
	require.EqualValues(t, err.Code(), types.CodeInsufficientCoin)

	// Invalid market
	err = checkMsgCreateOrder(input.ctx, input.mk, msg, issueAmount, issueAmount, dex.CET, math.MaxUint64)
	require.EqualValues(t, err.Code(), types.CodeInvalidMarket)

	mkInfo := MarketInfo{
		Stock:             stock,
		Money:             dex.CET,
		PricePrecision:    6,
		OrderPrecision:    1,
		LastExecutedPrice: sdk.NewDec(0),
	}
	ret := input.mk.SetMarket(input.ctx, mkInfo)
	require.Nil(t, ret)

	// Invalid price precision
	err = checkMsgCreateOrder(input.ctx, input.mk, msg, issueAmount, issueAmount, dex.CET, math.MaxUint64)
	require.EqualValues(t, err.Code(), types.CodeInvalidPricePrecision)

	// Forbidden token
	msg.PricePrecision = 6
	err = checkMsgCreateOrder(input.ctx, input.mk, msg, issueAmount, issueAmount, dex.CET, math.MaxUint64)
	require.EqualValues(t, err.Code(), types.CodeTokenForbidByIssuer)

	mkInfo.Stock = money
	mkInfo.Money = dex.CET
	ret = input.mk.SetMarket(input.ctx, mkInfo)
	require.Nil(t, ret)

	// Invalid order quantity
	msg.Sender = haveCetAddress
	msg.Quantity = 2
	msg.TradingPair = GetSymbol(money, dex.CET)
	err = checkMsgCreateOrder(input.ctx, input.mk, msg, 1, 6, dex.CET, math.MaxUint64)
	require.EqualValues(t, types.CodeInvalidOrderAmount, err.Code())

	// Pass
	msg.Sender = forbidAddr
	msg.Quantity = 10
	msg.TradingPair = GetSymbol(money, dex.CET)
	err = checkMsgCreateOrder(input.ctx, input.mk, msg, 1, 60, dex.CET, math.MaxUint64)
	require.Nil(t, err)
}

func TestCheckMsgCreateTradingPair(t *testing.T) {
	input := prepareMockInput(t, false, false)

	msg := MsgCreateTradingPair{
		Creator:        forbidAddr,
		Stock:          stock,
		Money:          dex.CET,
		PricePrecision: 8,
		OrderPrecision: 8,
	}

	// Not exist token
	msg.Money = "test"
	err := checkMsgCreateTradingPair(input.ctx, msg, input.mk)
	require.NotNil(t, err)
	require.EqualValues(t, types.CodeInvalidToken, err.Code())

	msg.Money = dex.CET
	msg.Stock = "test"
	err = checkMsgCreateTradingPair(input.ctx, msg, input.mk)
	require.NotNil(t, err)
	require.EqualValues(t, types.CodeInvalidToken, err.Code())

	// Invalid token issuer
	msg.Stock = stock
	err = checkMsgCreateTradingPair(input.ctx, msg, input.mk)
	require.NotNil(t, err)
	require.EqualValues(t, types.CodeInvalidTokenIssuer, err.Code())

	// Stock/Cet trading pair not exist
	msg.Money = money
	msg.Creator = haveCetAddress
	err = checkMsgCreateTradingPair(input.ctx, msg, input.mk)
	require.Nil(t, err)

	// Insufficient coin
	input.mk.SetParams(input.ctx, types.Params{
		CreateMarketFee: OriginHaveCetAmount,
	})
	msg.Creator = haveCetAddress
	msg.Money = dex.CET
	msg.Stock = stock
	input.mk.SetParams(input.ctx, types.Params{
		CreateMarketFee: 1e18,
	})
	err = checkMsgCreateTradingPair(input.ctx, msg, input.mk)
	require.NotNil(t, err)
	require.EqualValues(t, types.CodeInsufficientCoin, err.Code())

	// Success
	input.mk.SetParams(input.ctx, types.Params{
		CreateMarketFee: 100000,
	})
	err = checkMsgCreateTradingPair(input.ctx, msg, input.mk)
	require.Nil(t, err)

	err = input.mk.SetMarket(input.ctx, MarketInfo{
		Stock:             stock,
		Money:             dex.CET,
		PricePrecision:    8,
		OrderPrecision:    0,
		LastExecutedPrice: sdk.NewDec(0),
	})

	// Invalid Repeat market
	err = checkMsgCreateTradingPair(input.ctx, msg, input.mk)
	require.NotNil(t, err)
	require.EqualValues(t, types.CodeRepeatTradingPair, err.Code())
}

func TestGetDenomAndOrderAmount(t *testing.T) {
	msg := MsgCreateOrder{
		Sender:         haveCetAddress,
		Identify:       255,
		TradingPair:    GetSymbol(stock, dex.CET),
		OrderType:      LimitOrder,
		Side:           BUY,
		Price:          11,
		PricePrecision: 8,
		Quantity:       1e8,
		TimeInForce:    GTE,
		ExistBlocks:    10000,
	}

	// 1e8 * 11 / 10^8
	denom, amount, err := getDenomAndOrderAmount(msg)
	require.Nil(t, err)
	require.EqualValues(t, dex.CET, denom)
	require.EqualValues(t, 11, amount)

	// 10 * 11 / 10^8 ≈ 10^-6
	msg.Quantity = 10
	denom, amount, err = getDenomAndOrderAmount(msg)
	require.Nil(t, err)
	require.EqualValues(t, dex.CET, denom)
	require.EqualValues(t, 1, amount)

	msg.Quantity = types.MaxOrderAmount + 1
	msg.PricePrecision = 0
	msg.Price = 1
	_, _, err = getDenomAndOrderAmount(msg)
	require.NotNil(t, err)
	require.EqualValues(t, types.CodeInvalidOrderAmount, err.Code())

	msg.Side = SELL
	msg.Quantity = 100
	denom, amount, err = getDenomAndOrderAmount(msg)
	require.Nil(t, err)
	require.EqualValues(t, stock, denom)
	require.EqualValues(t, msg.Quantity, amount)

	msg.Quantity = types.MaxOrderAmount + 1
	msg.Side = SELL
	_, _, err = getDenomAndOrderAmount(msg)
	require.NotNil(t, err)
	require.EqualValues(t, types.CodeInvalidOrderAmount, err.Code())
}

func TestCheckMsgCancelOrder(t *testing.T) {
	input := prepareMockInput(t, false, false)

	orderID := types.AssemblyOrderID(haveCetAddress.String(), 1, 1)

	msg := MsgCancelOrder{
		OrderID: orderID,
		Sender:  haveCetAddress,
	}
	failed := checkMsgCancelOrder(input.ctx, msg, input.mk)
	require.NotNil(t, failed)
	require.EqualValues(t, types.CodeOrderNotFound, failed.Code())

	// Create order
	msgGteOrder := types.MsgCreateOrder{
		Sender:         haveCetAddress,
		Identify:       1,
		TradingPair:    GetSymbol(stock, "cet"),
		OrderType:      types.LimitOrder,
		PricePrecision: 8,
		Price:          100,
		Quantity:       10000000,
		Side:           types.SELL,
		TimeInForce:    types.GTE,
	}

	seq, err := input.mk.QuerySeqWithAddr(input.ctx, msgGteOrder.Sender)
	require.Nil(t, err)
	ret := createCetMarket(input, stock, 10)
	require.Equal(t, true, ret.IsOK(), "create market should succeed")
	ret = input.handler(input.ctx, msgGteOrder)
	require.Equal(t, true, ret.IsOK(), "create market should succeed")

	// Invalid order sender
	orderID = types.AssemblyOrderID(haveCetAddress.String(), seq, msgGteOrder.Identify)
	msg.OrderID = orderID
	msg.Sender = forbidAddr
	failed = checkMsgCancelOrder(input.ctx, msg, input.mk)
	require.NotNil(t, failed)
	require.EqualValues(t, types.CodeNotMatchSender, failed.Code())
}

func TestCheckMsgCancelTradingPair(t *testing.T) {
	timeNow := time.Now()
	input := prepareMockInput(t, false, false)
	input.ctx = input.ctx.WithBlockTime(timeNow)
	param := input.mk.GetParams(input.ctx)

	msg := MsgCancelTradingPair{
		Sender:        haveCetAddress,
		TradingPair:   GetSymbol(stock, dex.CET),
		EffectiveTime: timeNow.UnixNano(),
	}

	// Invalid cancel time
	err := checkMsgCancelTradingPair(input.mk, msg, input.ctx)
	require.EqualValues(t, types.CodeInvalidCancelTime, err.Code())

	msg.EffectiveTime = timeNow.UnixNano() + param.MarketMinExpiredTime - 1
	err = checkMsgCancelTradingPair(input.mk, msg, input.ctx)
	require.EqualValues(t, types.CodeInvalidCancelTime, err.Code())

	// Invalid market
	msg.EffectiveTime = timeNow.UnixNano() + param.MarketMinExpiredTime
	err = checkMsgCancelTradingPair(input.mk, msg, input.ctx)
	require.EqualValues(t, types.CodeInvalidMarket, err.Code())

	ret := createCetMarket(input, stock, 10)
	require.EqualValues(t, sdk.CodeOK, ret.Code)

	// Invalid sender
	msg.Sender = forbidAddr
	err = checkMsgCancelTradingPair(input.mk, msg, input.ctx)
	require.EqualValues(t, types.CodeNotMatchSender, err.Code())

	// Token not forbidden when money = cet
	msg.Sender = haveCetAddress
	err = checkMsgCancelTradingPair(input.mk, msg, input.ctx)
	require.Nil(t, nil)

	// Token not forbidden when money != cet
	err = input.mk.SetMarket(input.ctx, MarketInfo{
		Stock: stock,
		Money: money,
	})
	require.Nil(t, err)

	msg.TradingPair = GetSymbol(stock, money)
	err = checkMsgCancelTradingPair(input.mk, msg, input.ctx)
	require.Nil(t, err)

	// -----------------------

	input = prepareMockInput(t, true, true)
	input.ctx = input.ctx.WithBlockTime(timeNow)

	err = input.mk.SetMarket(input.ctx, MarketInfo{
		Stock: stock,
		Money: dex.CET,
	})
	require.Nil(t, err)

	// Token forbidden when money = cet
	msg.TradingPair = GetSymbol(stock, dex.CET)
	err = checkMsgCancelTradingPair(input.mk, msg, input.ctx)
	require.Nil(t, err)
}

func TestCheckMsgModifyPricePrecision(t *testing.T) {
	input := prepareMockInput(t, false, false)
	msg := MsgModifyPricePrecision{
		Sender:         haveCetAddress,
		TradingPair:    GetSymbol(stock, dex.CET),
		PricePrecision: 8,
	}

	// Invalid market
	err := checkMsgModifyPricePrecision(input.ctx, msg, input.mk)
	require.EqualValues(t, types.CodeInvalidMarket, err.Code())

	// Invalid price precision
	ret := createCetMarket(input, stock, 7)
	require.EqualValues(t, sdk.CodeOK, ret.Code)

	// Invalid tx sender
	msg.PricePrecision = 9
	msg.Sender = forbidAddr
	err = checkMsgModifyPricePrecision(input.ctx, msg, input.mk)
	require.EqualValues(t, types.CodeNotMatchSender, err.Code())

	msg.PricePrecision = 3
	msg.Sender = forbidAddr
	err = checkMsgModifyPricePrecision(input.ctx, msg, input.mk)
	require.EqualValues(t, types.CodeNotMatchSender, err.Code())
}

func TestPackageCancelOrderMsgWithDelReason(t *testing.T) {
	var (
		keeper = &mockKeeper{}
		param  = types.DefaultParams()
		ctx    = sdk.NewContext(nil, abci.Header{ChainID: "test-chain-id"},
			false, log.NewNopLogger())
	)
	param.GTEOrderLifetime = 200
	ctx = ctx.WithBlockHeight(299)
	or := &types.Order{
		Sender:           haveCetAddress,
		Sequence:         1,
		Identify:         2,
		TradingPair:      GetSymbol("abc", "cet"),
		OrderType:        types.LimitOrder,
		Price:            sdk.NewDec(100),
		Quantity:         10000000,
		Side:             types.SELL,
		TimeInForce:      types.GTE,
		Height:           100,
		ExistBlocks:      200,
		FrozenFeatureFee: 4100,
		FrozenCommission: 3900,
		FrozenFee:        5600,

		LeftStock: 10000000,
		Freeze:    30000000,
		DealStock: 0,
		DealMoney: 0,
	}

	msg := packageCancelOrderMsgWithDelReason(ctx, or, types.CancelOrderByManual, &param, keeper)
	require.EqualValues(t, 0, msg.UsedFeatureFee)
	require.EqualValues(t, param.FeeForZeroDeal, msg.UsedCommission)
	require.EqualValues(t, param.FeeForZeroDeal*keeper.GetRebateRatio(ctx)/keeper.GetRebateRatioBase(ctx), msg.RebateAmount)

}

// Only for online debug

func TestCalOrderCommissionAndFee(t *testing.T) {
	msg := MsgCreateOrder{
		Identify:       0,
		OrderType:      LimitOrder,
		Price:          4,
		PricePrecision: 3,
		Quantity:       50000000000,
		Side:           SELL,
		TimeInForce:    GTE,
		ExistBlocks:    2880000,
		TradingPair:    "blt/cet",
	}
	ctx := sdk.Context{}
	param := DefaultParams()
	mk := MockQueryMarketInfoAndParams{}
	featureFee := calFeatureFeeForExistBlocks(msg, param)
	fmt.Println(featureFee)

	commission, err := calOrderCommission(ctx, &mk, msg)
	require.Nil(t, err)
	fmt.Println(commission)

	fmt.Println(featureFee + commission)
}

type MockQueryMarketInfoAndParams struct {
	keepers.Keeper
}

func (m *MockQueryMarketInfoAndParams) GetParams(ctx sdk.Context) types.Params {
	return DefaultParams()
}

func (m *MockQueryMarketInfoAndParams) GetMarketVolume(ctx sdk.Context, stock, money string, stockVolume, moneyVolume sdk.Dec) sdk.Dec {
	if stock == dex.CET || money == dex.CET {
		return m.Keeper.GetMarketVolume(ctx, stock, money, stockVolume, moneyVolume)
	}

	volume := sdk.ZeroDec()
	if marketInfo, err := m.GetMarketInfo(ctx, dex.GetSymbol(dex.CET, money)); err == nil {
		if marketInfo.LastExecutedPrice.IsZero() {
			return volume
		}
		volume = moneyVolume.Quo(marketInfo.LastExecutedPrice)
	} else if marketInfo, err := m.GetMarketInfo(ctx, dex.GetSymbol(dex.CET, stock)); err == nil {
		if marketInfo.LastExecutedPrice.IsZero() {
			return volume
		}
		volume = stockVolume.Quo(marketInfo.LastExecutedPrice)
	} else if marketInfo, err := m.GetMarketInfo(ctx, dex.GetSymbol(money, dex.CET)); err == nil {
		volume = moneyVolume.Mul(marketInfo.LastExecutedPrice)
	} else if marketInfo, err := m.GetMarketInfo(ctx, dex.GetSymbol(stock, dex.CET)); err == nil {
		volume = stockVolume.Mul(marketInfo.LastExecutedPrice)
	}
	return volume
}

func (m *MockQueryMarketInfoAndParams) GetMarketInfo(ctx sdk.Context, tradingPair string) (types.MarketInfo, error) {
	return types.MarketInfo{}, nil
}
