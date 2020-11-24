package keepers

import (
	"fmt"

	"github.com/coinexchain/cet-sdk/modules/bancorlite/internal/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	dex "github.com/coinexchain/cet-sdk/types"
)

type BancorInfo struct {
	Owner              sdk.AccAddress `json:"sender"`
	Stock              string         `json:"stock"`
	Money              string         `json:"money"`
	InitPrice          sdk.Dec        `json:"init_price"`
	MaxSupply          sdk.Int        `json:"max_supply"`
	StockPrecision     byte           `json:"stock_precision"`
	MaxPrice           sdk.Dec        `json:"max_price"`
	MaxMoney           sdk.Int        `json:"max_money"` // DEX2
	AR                 int64          `json:"ar"`        // DEX2
	Price              sdk.Dec        `json:"price"`
	StockInPool        sdk.Int        `json:"stock_in_pool"`
	MoneyInPool        sdk.Int        `json:"money_in_pool"`
	EarliestCancelTime int64          `json:"earliest_cancel_time"`
}

func (bi *BancorInfo) GetSymbol() string {
	return dex.GetSymbol(bi.Stock, bi.Money)
}

func (bi *BancorInfo) UpdateStockInPool(stockInPool sdk.Int) bool {
	if stockInPool.IsNegative() || stockInPool.GT(bi.MaxSupply) {
		return false
	}
	bi.StockInPool = stockInPool
	suppliedStock := bi.MaxSupply.Sub(bi.StockInPool)
	if bi.MaxMoney.IsZero() {
		bi.Price = bi.MaxPrice.Sub(bi.InitPrice).MulInt(suppliedStock).QuoInt(bi.MaxSupply).Add(bi.InitPrice)
		bi.MoneyInPool = bi.Price.Add(bi.InitPrice).MulInt(suppliedStock).QuoInt64(2).RoundInt()
	} else {
		// s = s/s_max * 1000, as of precision is 0.001
		factoredStock := suppliedStock.MulRaw(types.SupplyRatioSamples)
		s := factoredStock.Quo(bi.MaxSupply).Int64()
		if s > types.SupplyRatioSamples {
			return false
		}
		contrast := sdk.NewInt(s).Mul(bi.MaxSupply)
		// ratio = (s/s_max)^ar, ar = (p_max * s_max - m_max) / (m_max - p_init * s_max)
		ratio := types.TableLookup(bi.AR+types.ARSamples, s)
		// price_ratio = (s/s_max)^(ar)
		priceRatio := types.TableLookup(bi.AR, s)
		if factoredStock.GT(contrast) {
			if s > types.SupplyRatioSamples {
				return false
			}
			// ratio = (ratioNear - ratio) * (stock_now / s_max * 1000 - (s)) + ratio
			ratioNear := types.TableLookup(bi.AR+types.ARSamples, s+1)
			ratio = ratioNear.Sub(ratio).MulInt(factoredStock.Sub(sdk.NewInt(s).Mul(bi.MaxSupply))).
				Quo(sdk.NewDecFromInt(bi.MaxSupply)).Add(ratio)
			priceRatioNear := types.TableLookup(bi.AR, s+1)
			priceRatio = priceRatioNear.Sub(priceRatio).MulInt(factoredStock.Sub(sdk.NewInt(s).Mul(bi.MaxSupply))).
				Quo(sdk.NewDecFromInt(bi.MaxSupply)).Add(priceRatio)
		}

		// m_now = (m_max - s_max * price_max) * ratio + price_init * s_now
		bi.MoneyInPool = ratio.MulInt(bi.MaxMoney.Sub(bi.InitPrice.MulInt(bi.MaxSupply).TruncateInt())).
			Add(bi.InitPrice.MulInt(suppliedStock)).TruncateInt()
		// price = priceRatio * (maxPrice - initPrice) + initPrice
		bi.Price = priceRatio.MulTruncate(bi.MaxPrice.Sub(bi.InitPrice)).Add(bi.InitPrice)
	}
	return true
}

func (bi *BancorInfo) IsConsistent() bool {
	if bi.StockInPool.IsNegative() || bi.StockInPool.GT(bi.MaxSupply) {
		return false
	}
	suppliedStock := bi.MaxSupply.Sub(bi.StockInPool)
	if bi.InitPrice.Equal(bi.MaxPrice) {
		if !bi.MaxMoney.Equal(bi.InitPrice.MulInt(bi.MaxSupply).TruncateInt()) || bi.AR != 0 {
			return false
		}
		if !bi.InitPrice.MulInt(suppliedStock).Equal(sdk.NewDecFromInt(bi.MoneyInPool)) {
			return false
		}
		return true
	}
	if bi.MaxMoney.IsZero() {
		price := bi.MaxPrice.Sub(bi.InitPrice).MulInt(suppliedStock).QuoInt(bi.MaxSupply).Add(bi.InitPrice)
		moneyInPool := price.Add(bi.InitPrice).MulInt(suppliedStock).QuoInt64(2).RoundInt()
		return price.Equal(bi.Price) && moneyInPool.Equal(bi.MoneyInPool) && (bi.AR == 0)
	}

	if bi.MoneyInPool.IsNegative() || bi.MoneyInPool.
		GT(bi.MaxPrice.MulInt(bi.MaxSupply).TruncateInt()) || bi.MaxMoney.
		LT(bi.InitPrice.MulInt(bi.MaxSupply).TruncateInt()) {
		return false
	}
	biMsg := types.MsgBancorInit{
		MaxSupply: bi.MaxSupply,
		MaxMoney:  bi.MaxMoney,
	}
	ar, ok := types.CheckAR(biMsg, bi.InitPrice, bi.MaxPrice)
	if ok || ar != bi.AR {
		return false
	}
	biNew := *bi
	ok = biNew.UpdateStockInPool(biNew.StockInPool)
	if !ok {
		return false
	}
	return bi.MoneyInPool.Equal(biNew.MoneyInPool) && bi.Price.Equal(biNew.Price)
}

type BancorInfoDisplay struct {
	Owner              string `json:"owner"`
	Stock              string `json:"stock"`
	Money              string `json:"money"`
	InitPrice          string `json:"init_price"`
	MaxSupply          string `json:"max_supply"`
	StockPrecision     string `json:"stock_precision"`
	MaxPrice           string `json:"max_price"`
	MaxMoney           string `json:"max_money"`
	AR                 string `json:"ar"`
	CurrentPrice       string `json:"current_price"`
	StockInPool        string `json:"stock_in_pool"`
	MoneyInPool        string `json:"money_in_pool"`
	EarliestCancelTime int64  `json:"earliest_cancel_time"`
}

func NewBancorInfoDisplay(bi *BancorInfo) BancorInfoDisplay {
	price := sdk.ZeroDec()
	suppliedStock := bi.MaxSupply.Sub(bi.StockInPool)
	if bi.MaxMoney.IsPositive() {
		s := suppliedStock.MulRaw(types.SupplyRatioSamples).Quo(bi.MaxSupply).Int64()
		if s == types.SupplyRatioSamples {
			price = bi.MaxPrice
		} else if s == 0 && bi.MoneyInPool.IsZero() {
			price = bi.InitPrice
		} else {
			ratio := types.TableLookup(bi.AR+types.ARSamples, s)
			ratioNext := types.TableLookup(bi.AR+types.ARSamples, s+1)
			price = bi.InitPrice.Add(
				bi.MaxPrice.Sub(bi.InitPrice).MulInt64(types.ARSamples).MulInt64(types.ARSamples).
					QuoInt64(bi.AR + types.ARSamples).
					Mul(ratioNext.Sub(ratio)))
		}
	} else {
		price = bi.Price
	}
	return BancorInfoDisplay{
		Owner:              bi.Owner.String(),
		Stock:              bi.Stock,
		Money:              bi.Money,
		InitPrice:          bi.InitPrice.String(),
		MaxSupply:          bi.MaxSupply.String(),
		StockPrecision:     fmt.Sprintf("%d", bi.StockPrecision),
		MaxPrice:           bi.MaxPrice.String(),
		MaxMoney:           bi.MaxMoney.String(),
		AR:                 fmt.Sprintf("%d", bi.AR),
		CurrentPrice:       price.String(),
		StockInPool:        bi.StockInPool.String(),
		MoneyInPool:        bi.MoneyInPool.String(),
		EarliestCancelTime: bi.EarliestCancelTime,
	}
}
