package keepers

import (
	"bytes"
	"errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/params"

	"github.com/coinexchain/cet-sdk/modules/asset"
	"github.com/coinexchain/cet-sdk/modules/market/internal/types"
	"github.com/coinexchain/cet-sdk/msgqueue"
	dex "github.com/coinexchain/cet-sdk/types"
)

type QueryMarketInfoAndParams interface {
	GetParams(ctx sdk.Context) types.Params
	GetMarketVolume(ctx sdk.Context, stock, money string, stockVolume, moneyVolume sdk.Dec) sdk.Dec
}

type Keeper struct {
	paramSubspace params.Subspace
	marketKey     sdk.StoreKey
	cdc           *codec.Codec
	axk           types.ExpectedAssetStatusKeeper
	bnk           types.ExpectedBankxKeeper
	ock           *OrderCleanUpDayKeeper
	gmk           GlobalMarketInfoKeeper
	msgProducer   msgqueue.MsgSender
	ak            auth.AccountKeeper
	authX         types.ExpectedAuthXKeeper
	supplyKeeper     authtypes.SupplyKeeper
}

func NewKeeper(key sdk.StoreKey, axkVal types.ExpectedAssetStatusKeeper,
	bnkVal types.ExpectedBankxKeeper, cdcVal *codec.Codec,
	msgKeeperVal msgqueue.MsgSender, paramstore params.Subspace,
	ak auth.AccountKeeper, authX types.ExpectedAuthXKeeper,
	supplyKeeper     authtypes.SupplyKeeper) Keeper {

	return Keeper{
		paramSubspace: paramstore.WithKeyTable(types.ParamKeyTable()),
		marketKey:     key,
		cdc:           cdcVal,
		axk:           axkVal,
		bnk:           bnkVal,
		ock:           NewOrderCleanUpDayKeeper(key),
		gmk:           NewGlobalMarketInfoKeeper(key, cdcVal),
		msgProducer:   msgKeeperVal,
		ak:            ak,
		authX:         authX,
		supplyKeeper: supplyKeeper,
	}
}

func (k Keeper) GetMarketsWithNewlyAddedOrder(ctx sdk.Context) []string {
	store := ctx.KVStore(k.marketKey)
	iter := store.Iterator(NewlyAddedKeyPrefix, NewlyAddedKeyEnd)
	res := make([]string, 0, 100)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		res = append(res, string(key[1:]))
	}
	return res
}

func (k Keeper) QuerySeqWithAddr(ctx sdk.Context, addr sdk.AccAddress) (uint64, sdk.Error) {
	acc := k.ak.GetAccount(ctx, addr)
	if acc != nil {
		return acc.GetSequence(), nil
	}
	return 0, sdk.ErrUnknownAddress("account does not exist")
}

func (k Keeper) GetToken(ctx sdk.Context, symbol string) asset.Token {
	return k.axk.GetToken(ctx, symbol)
}

func (k Keeper) HasCoins(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) bool {
	return k.bnk.HasCoins(ctx, addr, amt)
}

func (k Keeper) FreezeCoins(ctx sdk.Context, acc sdk.AccAddress, amt sdk.Coins) sdk.Error {
	return k.bnk.FreezeCoins(ctx, acc, amt)
}

func (k Keeper) SubtractCoins(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) sdk.Error {
	return k.bnk.SubtractCoins(ctx, addr, amt)
}

func (k Keeper) DeductInt64CetFee(ctx sdk.Context, addr sdk.AccAddress, amt int64) sdk.Error {
	return k.bnk.DeductInt64CetFee(ctx, addr, amt)
}

func (k Keeper) SendCoins(ctx sdk.Context, from sdk.AccAddress, to sdk.AccAddress, amt sdk.Coins) sdk.Error {
	return k.bnk.SendCoins(ctx, from, to, amt)
}

func (k Keeper) UnFreezeCoins(ctx sdk.Context, acc sdk.AccAddress, amt sdk.Coins) sdk.Error {
	return k.bnk.UnFreezeCoins(ctx, acc, amt)
}

func (k Keeper) IsTokenIssuer(ctx sdk.Context, denom string, addr sdk.AccAddress) bool {
	return k.axk.IsTokenIssuer(ctx, denom, addr)
}

func (k Keeper) IsTokenExists(ctx sdk.Context, symbol string) bool {
	return k.axk.IsTokenExists(ctx, symbol)
}

func (k Keeper) IsSubScribed(topic string) bool {
	return k.msgProducer.IsSubscribed(topic)
}

func (k Keeper) IsForbiddenByTokenIssuer(ctx sdk.Context, denom string, addr sdk.AccAddress) bool {
	return k.axk.IsForbiddenByTokenIssuer(ctx, denom, addr)
}

func (k Keeper) IsTokenForbidden(ctx sdk.Context, symbol string) bool {
	return k.axk.IsTokenForbidden(ctx, symbol)
}

func (k Keeper) GetOrderCleanTime(ctx sdk.Context) int64 {
	return k.ock.GetUnixTime(ctx)
}

func (k Keeper) SetOrderCleanTime(ctx sdk.Context, t int64) {
	k.ock.SetUnixTime(ctx, t)
}

func (k Keeper) GetMsgProducer() msgqueue.MsgSender {
	return k.msgProducer
}

func (k Keeper) GetBankxKeeper() types.ExpectedBankxKeeper {
	return k.bnk
}

func (k Keeper) GetAssetKeeper() types.ExpectedAssetStatusKeeper {
	return k.axk
}

func (k Keeper) GetMarketKey() sdk.StoreKey {
	return k.marketKey
}

func (k Keeper) GetRefereeAddr(ctx sdk.Context, accAddr sdk.AccAddress) sdk.AccAddress {
	acc := k.authX.GetRefereeAddr(ctx, accAddr)
	if len(acc) == 0 {
		return nil
	}
	return acc
}

func (k Keeper) GetRebateRatio(ctx sdk.Context) int64 {
	return k.authX.GetRebateRatio(ctx)
}

func (k Keeper) GetRebateRatioBase(ctx sdk.Context) int64 {
	return k.authX.GetRebateRatioBase(ctx)
}

// -----------------------------------------------------------------------------
// Params

// SetParams sets the asset module's parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSubspace.SetParamSet(ctx, &params)
}

// GetParams gets the asset module's parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSubspace.GetParamSet(ctx, &params)
	return
}

func (k Keeper) GetMarketFeeMin(ctx sdk.Context) int64 {
	return k.GetParams(ctx).MarketFeeMin
}

// -----------------------------------------------------------------------------
// Order

// SetOrder implements token Keeper.
func (k Keeper) SetOrder(ctx sdk.Context, order *types.Order) sdk.Error {
	return NewOrderKeeper(k.marketKey, order.TradingPair, k.cdc).Add(ctx, order)
}

func (k Keeper) GetAllOrders(ctx sdk.Context) []*types.Order {
	return NewGlobalOrderKeeper(k.marketKey, k.cdc).GetAllOrders(ctx)
}

// -----------------------------------------------
// market info

func (k Keeper) SetMarket(ctx sdk.Context, info types.MarketInfo) sdk.Error {
	return k.gmk.SetMarket(ctx, info)
}

func (k Keeper) RemoveMarket(ctx sdk.Context, symbol string) sdk.Error {
	return k.gmk.RemoveMarket(ctx, symbol)
}

func (k Keeper) GetAllMarketInfos(ctx sdk.Context) []types.MarketInfo {
	return k.gmk.GetAllMarketInfos(ctx)
}

func (k Keeper) MarketCountOfStock(ctx sdk.Context, stock string) int64 {
	return k.gmk.MarketCountOfStock(ctx, stock)
}

func (k Keeper) GetMarketInfo(ctx sdk.Context, symbol string) (types.MarketInfo, error) {
	return k.gmk.GetMarketInfo(ctx, symbol)
}

func (k Keeper) SubtractFeeAndCollectFee(ctx sdk.Context, addr sdk.AccAddress, amt int64) sdk.Error {
	return k.bnk.DeductInt64CetFee(ctx, addr, amt)
}

func (k Keeper) MarketOwner(ctx sdk.Context, info types.MarketInfo) sdk.AccAddress {
	return k.axk.GetToken(ctx, info.Stock).GetOwner()
}

func (k *Keeper) GetMarketLastExePrice(ctx sdk.Context, symbol string) (sdk.Dec, error) {
	mi, err := k.GetMarketInfo(ctx, symbol)
	if err != nil {
		return sdk.ZeroDec(), err
	}
	return mi.LastExecutedPrice, err
}

func (k Keeper) GetMarketVolume(ctx sdk.Context, stock, money string, stockVolume, moneyVolume sdk.Dec) sdk.Dec {
	volume := sdk.ZeroDec()
	if stock == dex.CET {
		volume = stockVolume
	} else if money == dex.CET {
		volume = moneyVolume
	} else if marketInfo, err := k.GetMarketInfo(ctx, dex.GetSymbol(dex.CET, money)); err == nil {
		if marketInfo.LastExecutedPrice.IsZero() {
			return volume
		}
		volume = moneyVolume.Quo(marketInfo.LastExecutedPrice)
	} else if marketInfo, err := k.GetMarketInfo(ctx, dex.GetSymbol(dex.CET, stock)); err == nil {
		if marketInfo.LastExecutedPrice.IsZero() {
			return volume
		}
		volume = stockVolume.Quo(marketInfo.LastExecutedPrice)
	} else if marketInfo, err := k.GetMarketInfo(ctx, dex.GetSymbol(money, dex.CET)); err == nil {
		volume = moneyVolume.Mul(marketInfo.LastExecutedPrice)
	} else if marketInfo, err := k.GetMarketInfo(ctx, dex.GetSymbol(stock, dex.CET)); err == nil {
		volume = stockVolume.Mul(marketInfo.LastExecutedPrice)
	}
	return volume
}

func (k *Keeper) IsMarketExist(ctx sdk.Context, symbol string) bool {
	_, err := k.GetMarketInfo(ctx, symbol)
	return err == nil
}

// -----------------------------------------------------------------------------

type GlobalMarketInfoKeeper interface {
	SetMarket(ctx sdk.Context, info types.MarketInfo) sdk.Error
	RemoveMarket(ctx sdk.Context, symbol string) sdk.Error
	GetAllMarketInfos(ctx sdk.Context) []types.MarketInfo
	MarketCountOfStock(ctx sdk.Context, stock string) int64
	GetMarketInfo(ctx sdk.Context, symbol string) (types.MarketInfo, error)
}

type PersistentMarketInfoKeeper struct {
	marketKey sdk.StoreKey
	cdc       *codec.Codec
}

func NewGlobalMarketInfoKeeper(key sdk.StoreKey, cdcVal *codec.Codec) GlobalMarketInfoKeeper {
	return PersistentMarketInfoKeeper{
		marketKey: key,
		cdc:       cdcVal,
	}
}

// SetMarket implements token Keeper.
func (k PersistentMarketInfoKeeper) SetMarket(ctx sdk.Context, info types.MarketInfo) sdk.Error {
	store := ctx.KVStore(k.marketKey)
	bz, err := k.cdc.MarshalBinaryBare(info)
	if err != nil {
		return sdk.ErrInternal(err.Error())
	}
	store.Set(marketStoreKey(MarketIdentifierPrefix, info.GetSymbol()), bz)
	return nil
}

func (k PersistentMarketInfoKeeper) RemoveMarket(ctx sdk.Context, symbol string) sdk.Error {
	store := ctx.KVStore(k.marketKey)
	key := marketStoreKey(MarketIdentifierPrefix, symbol)
	value := store.Get(key)
	if value != nil {
		store.Delete(key)
	}
	return nil
}

func (k PersistentMarketInfoKeeper) GetAllMarketInfos(ctx sdk.Context) []types.MarketInfo {
	var infos []types.MarketInfo
	appendMarket := func(order types.MarketInfo) (stop bool) {
		infos = append(infos, order)
		return false
	}
	k.iterateMarket(ctx, appendMarket)
	return infos
}

func (k PersistentMarketInfoKeeper) MarketCountOfStock(ctx sdk.Context, stock string) (count int64) {
	store := ctx.KVStore(k.marketKey)
	key := marketStoreKey(MarketIdentifierPrefix, stock, types.SymbolSeparator)
	iter := sdk.KVStorePrefixIterator(store, key)
	defer iter.Close()
	for {
		if !iter.Valid() {
			return
		}
		count++
		iter.Next()
	}
}

func (k PersistentMarketInfoKeeper) iterateMarket(ctx sdk.Context, process func(info types.MarketInfo) bool) {
	store := ctx.KVStore(k.marketKey)
	iter := sdk.KVStorePrefixIterator(store, MarketIdentifierPrefix)
	defer iter.Close()
	for {
		if !iter.Valid() {
			return
		}
		val := iter.Value()
		if process(k.decodeMarket(val)) {
			return
		}
		iter.Next()
	}
}

func (k PersistentMarketInfoKeeper) GetMarketInfo(ctx sdk.Context, symbol string) (info types.MarketInfo, err error) {
	store := ctx.KVStore(k.marketKey)
	value := store.Get(marketStoreKey(MarketIdentifierPrefix, symbol))
	if len(value) == 0 {
		err = errors.New("No such market exist: " + symbol)
		return
	}
	err = k.cdc.UnmarshalBinaryBare(value, &info)
	return
}

func (k PersistentMarketInfoKeeper) decodeMarket(bz []byte) (info types.MarketInfo) {
	if err := k.cdc.UnmarshalBinaryBare(bz, &info); err != nil {
		panic(err)
	}
	return
}

func marketStoreKey(prefix []byte, params ...string) []byte {
	buf := bytes.NewBuffer(prefix)
	for _, param := range params {
		if _, err := buf.Write([]byte(param)); err != nil {
			panic(err)
		}
	}
	return buf.Bytes()
}

func (k Keeper) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) sdk.Error {
	return k.supplyKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt)
}
