package market

import (
	"crypto/sha256"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/coinexchain/cet-sdk/modules/market/internal/keepers"
	"github.com/coinexchain/cet-sdk/modules/market/internal/types"
	"github.com/coinexchain/cet-sdk/modules/market/match"
	"github.com/coinexchain/cet-sdk/msgqueue"

	dex "github.com/coinexchain/cet-sdk/types"
)

// Some handlers which are useful when orders are matched and traded.
type InfoForDeal struct {
	bxKeeper      types.ExpectedBankxKeeper
	msgSender     msgqueue.MsgSender
	dataHash      []byte
	changedOrders map[string]*types.Order
	lastPrice     sdk.Dec
	context       sdk.Context
	keeper keepers.Keeper
}

// returns true when a buyer's frozen money is not enough to buy LeftStock.
func notEnoughMoney(order *types.Order) bool {
	return order.Side == types.BUY &&
		order.Freeze < order.Price.Mul(sdk.NewDec(order.LeftStock)).RoundInt64()
}

// Wrapper an order with InfoForDeal, which contains useful handlers
type WrappedOrder struct {
	order       *types.Order
	infoForDeal *InfoForDeal
}

// WrappedOrder implements OrderForTrade interface
func (wo *WrappedOrder) GetPrice() sdk.Dec {
	return wo.order.Price
}

func (wo *WrappedOrder) GetAmount() int64 {
	if notEnoughMoney(wo.order) {
		// add this clause only for safe, should not reach here in production
		return 0
	}
	return wo.order.LeftStock
}

func (wo *WrappedOrder) GetHeight() int64 {
	return wo.order.Height
}

func (wo *WrappedOrder) GetSide() int {
	return int(wo.order.Side)
}

func (wo *WrappedOrder) GetOwner() match.Account {
	return wo.order.Sender
}

func (wo *WrappedOrder) String() string {
	return wo.order.OrderID()
}

func (wo *WrappedOrder) GetHash() []byte {
	res := sha256.Sum256(append([]byte(wo.order.OrderID()), wo.infoForDeal.dataHash...))
	return res[:]
}

func (wo *WrappedOrder) Deal(otherSide match.OrderForTrade, amount int64, price sdk.Dec) {
	other := otherSide.(*WrappedOrder)
	buyer, seller := wo.order, other.order
	if buyer.Side == types.SELL {
		buyer, seller = other.order, wo.order
	}
	stock, money := SplitSymbol(buyer.TradingPair)
	// buyer and seller will exchange stockCoins and moneyCoins
	stockCoins := sdk.Coins{sdk.NewCoin(stock, sdk.NewInt(amount))}
	moneyAmount := price.MulInt(sdk.NewInt(amount)).TruncateInt()
	moneyCoins := sdk.Coins{sdk.NewCoin(money, moneyAmount)}

	var moneyAmountInt64 int64
	if moneyAmount.GT(sdk.NewInt(types.MaxOrderAmount)) {
		// should not reach this clause in production
		return
	}
	moneyAmountInt64 = moneyAmount.Int64()
	buyer.LeftStock -= amount
	seller.LeftStock -= amount
	buyer.Freeze -= moneyAmountInt64
	seller.Freeze -= amount
	buyer.DealStock += amount
	seller.DealStock += amount
	buyer.DealMoney += moneyAmountInt64
	seller.DealMoney += moneyAmountInt64
	ctx := wo.infoForDeal.context
	// exchange the coins
	wo.infoForDeal.bxKeeper.UnFreezeCoins(ctx, seller.Sender, stockCoins)
	
	stockfee := sdk.NewInt(0)
	if buyer.FeeRate != (sdk.Dec{}) {
		stockfee = buyer.FeeRate.MulInt(sdk.NewInt(amount)).TruncateInt()
		stockFeeCoins := sdk.Coins{sdk.NewCoin(stock, stockfee)}
		if stockfee.Int64() > 0 {
			 //send fee (stockCoins * FeeRate) to fee collector
			if err := wo.infoForDeal.keeper.SendCoinsFromAccountToModule(ctx, seller.Sender, auth.FeeCollectorName, stockFeeCoins); err != nil {
				ctx.Logger().Error("%s", err.Error())
			}
		}
	}
	
	actualStockCoins := sdk.Coins{sdk.NewCoin(stock, sdk.NewInt(amount).Sub(stockfee))}
	//buyer receive (stockCoins - stockfee)
	wo.infoForDeal.bxKeeper.SendCoins(ctx, seller.Sender, buyer.Sender, actualStockCoins)
	

	wo.infoForDeal.bxKeeper.UnFreezeCoins(ctx, buyer.Sender, moneyCoins)

	moneyFee := sdk.NewInt(0)
	if seller.FeeRate != (sdk.Dec{}) {
		moneyFee = seller.FeeRate.MulInt(moneyAmount).TruncateInt()
		moneyFeeCoins := sdk.Coins{sdk.NewCoin(money, moneyFee)}
		if moneyFee.Int64() > 0 {
			 //send fee (stockCoins * FeeRate) to fee collector
			 if err := wo.infoForDeal.keeper.SendCoinsFromAccountToModule(ctx, buyer.Sender, auth.FeeCollectorName, moneyFeeCoins); err != nil {
				ctx.Logger().Error("%s", err.Error())
			}
		}
	}
	
	actualMoneyCoins := sdk.Coins{sdk.NewCoin(money, moneyAmount.Sub(moneyFee))}
	//seller receive (moneyCoins - moneyFee)
	wo.infoForDeal.bxKeeper.SendCoins(ctx, buyer.Sender, seller.Sender, actualMoneyCoins)

	// record the changed orders for further processing
	wo.infoForDeal.changedOrders[buyer.OrderID()] = buyer
	wo.infoForDeal.changedOrders[seller.OrderID()] = seller

	// record the last executed price, which will be stored in MarketInfo
	wo.infoForDeal.lastPrice = price

	if wo.infoForDeal.msgSender.IsSubscribed(types.Topic) {
		SendFillMsg(ctx, seller, buyer, amount, moneyAmountInt64, price, ctx.BlockHeight(),stockfee.Int64(), moneyFee.Int64())
	}
}

func SendFillMsg(ctx sdk.Context, seller *Order, buyer *Order, stockAmount, moneyAmount int64, price sdk.Dec, currentHeight int64, stockFee, moneyFee int64) {
	sellInfo := types.FillOrderInfo{
		OrderID:     seller.OrderID(),
		Height:      currentHeight,
		TradingPair: seller.TradingPair,
		Side:        seller.Side,
		FillPrice:   price,
		LeftStock:   seller.LeftStock,
		Freeze:      seller.Freeze,
		DealStock:   seller.DealStock,
		DealMoney:   seller.DealMoney,
		CurrStock:   stockAmount,
		CurrMoney:   moneyAmount,
		Price:       seller.Price,
		CurrStockFee: stockFee,
		CurrMoneyFee: moneyFee,
	}
	msgqueue.FillMsgs(ctx, types.FillOrderInfoKey, sellInfo)

	buyInfo := types.FillOrderInfo{
		OrderID:     buyer.OrderID(),
		Height:      currentHeight,
		TradingPair: buyer.TradingPair,
		Side:        buyer.Side,
		FillPrice:   price,
		LeftStock:   buyer.LeftStock,
		Freeze:      buyer.Freeze,
		DealStock:   buyer.DealStock,
		DealMoney:   buyer.DealMoney,
		CurrStock:   stockAmount,
		CurrMoney:   moneyAmount,
		Price:       buyer.Price,
		CurrStockFee: stockFee,
		CurrMoneyFee: moneyFee,
	}
	msgqueue.FillMsgs(ctx, types.FillOrderInfoKey, buyInfo)
}

// unfreeze the frozen token in the order and remove it from the market
func removeOrder(ctx sdk.Context, orderKeeper keepers.OrderKeeper, bxKeeper types.ExpectedBankxKeeper,
	keeper types.Keeper, order *types.Order, marketParam *types.Params) {
	if order.Freeze != 0 || order.FrozenFeatureFee != 0 || order.FrozenCommission != 0 {
		unfreezeCoinsForOrder(ctx, bxKeeper, order, keeper, marketParam)
	}
	if err := orderKeeper.Remove(ctx, order); err != nil {
		ctx.Logger().Error("%s", err.Error())
	}
}

// unfreeze an ask order's stock or a bid order's money
func unfreezeCoinsForOrder(ctx sdk.Context, bxKeeper types.ExpectedBankxKeeper, order *types.Order,
	keeper types.Keeper, marketParam *types.Params) {
	unfreezeCoinsInOrder(ctx, order, bxKeeper)
	chargeOrderCommission(ctx, order, marketParam.FeeForZeroDeal, bxKeeper, keeper)
	chargeOrderFeatureFee(ctx, order, marketParam.GTEOrderLifetime, bxKeeper, keeper)
}

func unfreezeCoinsInOrder(ctx sdk.Context, order *types.Order, bxKeeper types.ExpectedBankxKeeper) {
	frozenToken := order.GetOrderUsedDenom()
	if err := bxKeeper.UnFreezeCoins(ctx, order.Sender, dex.NewCoins(frozenToken, order.Freeze)); err != nil {
		ctx.Logger().Error("%s", err.Error())
	}
}

func chargeOrderCommission(ctx sdk.Context, order *types.Order, feeForZeroDeal int64,
	bxKeeper types.ExpectedBankxKeeper, keeper types.Keeper) {
	if order.FrozenCommission != 0 {
		if err := bxKeeper.UnFreezeCoins(ctx, order.Sender, dex.NewCetCoins(order.FrozenCommission)); err != nil {
			ctx.Logger().Error("%s", err.Error())
		}
		actualFee := order.CalActualOrderCommissionInt64(feeForZeroDeal)
		chargeFee(ctx, actualFee, order.Sender, keeper)
	}
}

func chargeOrderFeatureFee(ctx sdk.Context, order *types.Order, freeTimeBlocks int64,
	bxKeeper types.ExpectedBankxKeeper, keeper types.Keeper) {
	if order.TimeInForce == GTE && order.FrozenFeatureFee != 0 {
		if err := bxKeeper.UnFreezeCoins(ctx, order.Sender, dex.NewCetCoins(order.FrozenFeatureFee)); err != nil {
			ctx.Logger().Error("%s", err.Error())
		}
		actualFee := order.CalActualOrderFeatureFeeInt64(ctx, freeTimeBlocks)
		chargeFee(ctx, actualFee, order.Sender, keeper)
	}
}

func chargeFee(ctx sdk.Context, fee int64, userAddr sdk.AccAddress, keeper types.Keeper) {
	var (
		rebateAmount int64
		refereeAddr  sdk.AccAddress
	)
	refereeAddr = keeper.GetRefereeAddr(ctx, userAddr)
	if refereeAddr != nil {
		rebateAmount = calRebateAmount(ctx, fee, keeper)
		fee -= rebateAmount
	}
	if rebateAmount > 0 {
		if err := keeper.SendCoins(ctx, userAddr, refereeAddr, dex.NewCetCoins(rebateAmount)); err != nil {
			ctx.Logger().Error("%s", err.Error())
		}
	}
	if err := keeper.SubtractFeeAndCollectFee(ctx, userAddr, fee); err != nil {
		//should not reach this clause in production
		ctx.Logger().Debug("unfreezeCoinsForOrder: %s", err.Error())
	}
}

func calRebateAmount(ctx sdk.Context, fee int64, keeper types.ExpectedAuthXKeeper) int64 {
	ratio := keeper.GetRebateRatio(ctx)
	ratioBase := keeper.GetRebateRatioBase(ctx)
	rebateAmount := sdk.NewInt(fee).MulRaw(ratio).QuoRaw(ratioBase).Int64()
	return rebateAmount
}

// Iterate the candidate orders for matching, and remove the orders whose sender is forbidden by the money owner or the stock owner.
func filterCandidates(ctx sdk.Context, asKeeper types.ExpectedAssetStatusKeeper, ordersIn []*types.Order, stock, money string) []*types.Order {
	ordersOut := make([]*types.Order, 0, len(ordersIn))
	for _, order := range ordersIn {
		if !(asKeeper.IsForbiddenByTokenIssuer(ctx, stock, order.Sender) ||
			asKeeper.IsForbiddenByTokenIssuer(ctx, money, order.Sender)) {
			ordersOut = append(ordersOut, order)
		}
	}
	return ordersOut
}

func runMatch(ctx sdk.Context, midPrice sdk.Dec, ratio int64, symbol string, keeper keepers.Keeper, dataHash []byte, currHeight int64) (map[string]*types.Order, sdk.Dec) {
	orderKeeper := keepers.NewOrderKeeper(keeper.GetMarketKey(), symbol, types.ModuleCdc)
	asKeeper := keeper.GetAssetKeeper()
	bxKeeper := keeper.GetBankxKeeper()
	lowPrice := midPrice.Mul(sdk.NewDec(100 - ratio)).Quo(sdk.NewDec(100))
	highPrice := midPrice.Mul(sdk.NewDec(100 + ratio)).Quo(sdk.NewDec(100))

	infoForDeal := &InfoForDeal{
		bxKeeper:      bxKeeper,
		dataHash:      dataHash,
		changedOrders: make(map[string]*types.Order),
		context:       ctx,
		lastPrice:     sdk.NewDec(0),
		msgSender:     keeper.GetMsgProducer(),
		keeper: keeper,
	}

	// from the order book, we fetch the candidate orders for matching and filter them
	stock, money := SplitSymbol(orderKeeper.GetSymbol())
	orderCandidates := orderKeeper.GetMatchingCandidates(ctx)
	orderCandidates = filterCandidates(ctx, asKeeper, orderCandidates, stock, money)

	// fill bidList and askList with wrapped orders
	bidList := make([]match.OrderForTrade, 0, len(orderCandidates))
	askList := make([]match.OrderForTrade, 0, len(orderCandidates))
	for _, orderCandidate := range orderCandidates {
		wrappedOrder := &WrappedOrder{
			order:       orderCandidate,
			infoForDeal: infoForDeal,
		}
		if wrappedOrder.order.Side == types.BID {
			bidList = append(bidList, wrappedOrder)
		} else {
			askList = append(askList, wrappedOrder)
		}
	}
	// call the match engine
	match.Match(highPrice, midPrice, lowPrice, bidList, askList)

	// both dealt orders and IOC order need further processing
	ordersForUpdate := infoForDeal.changedOrders
	for _, order := range orderKeeper.GetOrdersAtHeight(ctx, currHeight) {
		if order.TimeInForce == types.IOC {
			// if an IOC order is not included, we include it
			if _, ok := ordersForUpdate[order.OrderID()]; !ok {
				ordersForUpdate[order.OrderID()] = order
			}
		}
	}

	return ordersForUpdate, infoForDeal.lastPrice
}

func removeExpiredOrder(ctx sdk.Context, keeper keepers.Keeper, marketInfoList []types.MarketInfo, marketParams *types.Params) {
	currHeight := ctx.BlockHeight()
	bankxKeeper := keeper.GetBankxKeeper()
	for _, mi := range marketInfoList {
		orderKeeper := keepers.NewOrderKeeper(keeper.GetMarketKey(), mi.GetSymbol(), types.ModuleCdc)
		oldOrders := orderKeeper.GetOlderThan(ctx, currHeight)

		for _, order := range oldOrders {
			if order.Height+order.ExistBlocks > currHeight {
				continue
			}
			removeOrder(ctx, orderKeeper, bankxKeeper, keeper, order, marketParams)
			if keeper.IsSubScribed(types.Topic) {
				cancelOrderInfo := packageCancelOrderMsgWithDelReason(ctx, order,
					types.CancelOrderByGteTimeOut, marketParams, keeper)
				msgqueue.FillMsgs(ctx, types.CancelOrderInfoKey, cancelOrderInfo)
			}
		}
	}
}

func removeExpiredMarket(ctx sdk.Context, keeper keepers.Keeper, marketParams *types.Params) {
	currHeight := ctx.BlockHeight()
	currTime := ctx.BlockHeader().Time.UnixNano()

	// process the delist requests
	bankxKeeper := keeper.GetBankxKeeper()
	delistKeeper := keepers.NewDelistKeeper(keeper.GetMarketKey())
	delistSymbols := delistKeeper.GetDelistSymbolsBeforeTime(ctx, currTime)
	for _, symbol := range delistSymbols {
		orderKeeper := keepers.NewOrderKeeper(keeper.GetMarketKey(), symbol, types.ModuleCdc)
		oldOrders := orderKeeper.GetOlderThan(ctx, currHeight+1)
		for _, ord := range oldOrders {
			removeOrder(ctx, orderKeeper, bankxKeeper, keeper, ord, marketParams)
			if keeper.IsSubScribed(types.Topic) {
				cancelOrderInfo := packageCancelOrderMsgWithDelReason(ctx, ord,
					types.CancelOrderByGteTimeOut, marketParams, keeper)
				msgqueue.FillMsgs(ctx, types.CancelOrderInfoKey, cancelOrderInfo)
			}
		}
		keeper.RemoveMarket(ctx, symbol)
	}
	delistKeeper.RemoveDelistRequestsBeforeTime(ctx, currTime)
}

func EndBlocker(ctx sdk.Context, keeper keepers.Keeper) /*sdk.Tags*/ {
	marketParams := keeper.GetParams(ctx)

	chainID := ctx.ChainID()
	recordTime := keeper.GetOrderCleanTime(ctx)
	currTime := ctx.BlockHeader().Time.Unix()

	var needRemove bool
	if !strings.Contains(chainID, IntegrationNetSubString) {
		if time.Unix(recordTime, 0).UTC().Day() != time.Unix(currTime, 0).UTC().Day() {
			needRemove = true
		}
	} else {
		if time.Unix(recordTime, 0).Minute() != time.Unix(currTime, 0).Minute() {
			needRemove = true
		}
	}

	// if this is the first block of a new day, we clean the GTE order and there is no trade
	if needRemove {
		marketInfoList := keeper.GetAllMarketInfos(ctx)
		keeper.SetOrderCleanTime(ctx, currTime)
		removeExpiredOrder(ctx, keeper, marketInfoList, &marketParams)
		removeExpiredMarket(ctx, keeper, &marketParams)
		return //nil
	}

	markets := keeper.GetMarketsWithNewlyAddedOrder(ctx)
	if len(markets) == 0 {
		return
	}
	marketInfoList := make([]types.MarketInfo, len(markets))
	for idx, market := range markets {
		var err error
		marketInfoList[idx], err = keeper.GetMarketInfo(ctx, market)
		if err != nil {
			return //should not reach here in production
		}
	}
	currHeight := ctx.BlockHeight()
	ordersForUpdateList := make([]map[string]*types.Order, len(marketInfoList))
	newPrices := make([]sdk.Dec, len(marketInfoList))
	for idx, mi := range marketInfoList {
		// if a token is globally forbidden, exchange it is also impossible
		if keeper.IsTokenForbidden(ctx, mi.Stock) ||
			keeper.IsTokenForbidden(ctx, mi.Money) {
			continue
		}
		symbol := mi.GetSymbol()
		dataHash := ctx.BlockHeader().DataHash
		ratio := marketParams.MaxExecutedPriceChangeRatio
		oUpdate, newPrice := runMatch(ctx, mi.LastExecutedPrice, ratio, symbol, keeper, dataHash, currHeight)
		newPrices[idx] = newPrice
		ordersForUpdateList[idx] = oUpdate
	}
	for idx, mi := range marketInfoList {
		// ignore a market if there are no orders need further processing
		if len(ordersForUpdateList[idx]) == 0 {
			continue
		}
		bankxKeeper := keeper.GetBankxKeeper()
		orderKeeper := keepers.NewOrderKeeper(keeper.GetMarketKey(), mi.GetSymbol(), types.ModuleCdc)
		// update the order book
		for _, order := range ordersForUpdateList[idx] {
			orderKeeper.Update(ctx, order)
			if order.TimeInForce == types.IOC || order.LeftStock == 0 || notEnoughMoney(order) {
				removeOrder(ctx, orderKeeper, bankxKeeper, keeper, order, &marketParams)
				if keeper.IsSubScribed(types.Topic) {
					cancelOrderInfo := packageCancelOrderMsg(ctx, order, &marketParams, keeper)
					msgqueue.FillMsgs(ctx, types.CancelOrderInfoKey, cancelOrderInfo)
				}
			}
		}
		// if some orders dealt, update last executed price of this market
		if !newPrices[idx].IsZero() {
			mi.LastExecutedPrice = newPrices[idx]
			keeper.SetMarket(ctx, mi)
		}
	}
}

func packageCancelOrderMsg(ctx sdk.Context, order *types.Order,
	marketParams *Params, keeper types.ExpectedAuthXKeeper) types.CancelOrderInfo {
	return packageCancelOrderMsgWithDelReason(ctx, order, "", marketParams, keeper)
}

func packageCancelOrderMsgWithDelReason(ctx sdk.Context, order *types.Order, delReason string,
	marketParams *Params, keeper types.ExpectedAuthXKeeper) types.CancelOrderInfo {
	currentHeight := ctx.BlockHeight()
	usedFeatureFee := int64(0)
	if order.FrozenFeatureFee != 0 {
		usedFeatureFee = order.CalActualOrderFeatureFeeInt64(ctx, marketParams.GTEOrderLifetime)
	}
	msgInfo := types.CancelOrderInfo{
		OrderID:        order.OrderID(),
		TradingPair:    order.TradingPair,
		Side:           order.Side,
		Height:         currentHeight,
		Price:          order.Price,
		UsedCommission: order.CalActualOrderCommissionInt64(marketParams.FeeForZeroDeal),
		UsedFeatureFee: usedFeatureFee,
		LeftStock:      order.LeftStock,
		RemainAmount:   order.Freeze,
		DealStock:      order.DealStock,
		DealMoney:      order.DealMoney,
	}
	msgInfo.RebateRefereeAddr = keeper.GetRefereeAddr(ctx, order.Sender).String()
	if len(msgInfo.RebateRefereeAddr) != 0 {
		msgInfo.RebateAmount = getRebateAmountInOrder(ctx, keeper, msgInfo.UsedCommission, msgInfo.UsedFeatureFee)
	}
	msgInfo.DelReason = getCancelOrderReason(order, delReason)
	return msgInfo
}

func getRebateAmountInOrder(ctx sdk.Context, keeper types.ExpectedAuthXKeeper, commission, featureFee int64) int64 {
	rebateCommission := calRebateAmount(ctx, commission, keeper)
	rebateFeatureFee := calRebateAmount(ctx, featureFee, keeper)
	return rebateCommission + rebateFeatureFee
}

func getCancelOrderReason(order *types.Order, delReason string) string {
	if len(delReason) != 0 {
		return delReason
	}
	if order.TimeInForce == types.IOC {
		return types.CancelOrderByIocType
	}
	if order.LeftStock == 0 {
		return types.CancelOrderByAllFilled
	}
	if notEnoughMoney(order) {
		return types.CancelOrderByNoEnoughMoney
	}
	return types.CancelOrderByNotKnow
}
