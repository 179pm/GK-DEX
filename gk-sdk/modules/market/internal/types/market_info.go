package types

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	dex "github.com/coinexchain/cet-sdk/types"
)

type MarketInfo struct {
	Stock             string  `json:"stock"`
	Money             string  `json:"money"`
	PricePrecision    byte    `json:"price_precision"`
	LastExecutedPrice sdk.Dec `json:"last_executed_price"`
	OrderPrecision    byte    `json:"order_precision"`
	BuyFeeRate    sdk.Dec    `json:"buy_fee_rate"`
	SellFeeRate    sdk.Dec    `json:"sell_fee_rate"`
}

func GetGranularityOfOrder(orderPrecision byte) int64 {
	if orderPrecision > 8 {
		orderPrecision = 0
	}
	return int64(math.Pow10(int(orderPrecision)))
}

func (msg MarketInfo) GetSymbol() string {
	return dex.GetSymbol(msg.Stock, msg.Money)
}
