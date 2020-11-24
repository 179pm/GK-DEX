package types

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/coinexchain/cet-sdk/modules/market"
	dex "github.com/coinexchain/cet-sdk/types"
)

// /////////////////////////////////////////////////////////

const MaxTradeAmount = int64(10000) * int64(10000) * int64(10000) * int64(10000) * 100 // Ten Billion

var _ sdk.Msg = MsgBancorInit{}
var _ sdk.Msg = MsgBancorTrade{}
var _ sdk.Msg = MsgBancorCancel{}

type MsgBancorInit struct {
	Owner              sdk.AccAddress `json:"owner"`
	Stock              string         `json:"stock"` // supply denom
	Money              string         `json:"money"` // paying denom
	InitPrice          string         `json:"init_price"`
	MaxSupply          sdk.Int        `json:"max_supply"`
	MaxPrice           string         `json:"max_price"`
	MaxMoney           sdk.Int        `json:"max_money"`
	StockPrecision     byte           `json:"stock_precision"`
	EarliestCancelTime int64          `json:"earliest_cancel_time"`
}

type MsgBancorCancel struct {
	Owner sdk.AccAddress `json:"owner"`
	Stock string         `json:"stock"`
	Money string         `json:"money"`
}

type MsgBancorTrade struct {
	Sender sdk.AccAddress `json:"sender"`
	Stock  string         `json:"stock"`
	Money  string         `json:"money"`
	//stock amount
	Amount int64 `json:"amount"`
	IsBuy  bool  `json:"is_buy"`
	//money up limit
	MoneyLimit int64 `json:"money_limit"`
}

func (msg MsgBancorInit) GetSymbol() string {
	return dex.GetSymbol(msg.Stock, msg.Money)
}
func (msg MsgBancorCancel) GetSymbol() string {
	return dex.GetSymbol(msg.Stock, msg.Money)
}
func (msg MsgBancorTrade) GetSymbol() string {
	return dex.GetSymbol(msg.Stock, msg.Money)
}

// --------------------------------------------------------
// sdk.Msg Implementation

func (msg MsgBancorInit) Route() string { return RouterKey }

func (msg MsgBancorInit) Type() string { return "bancor_init" }

func (msg MsgBancorInit) ValidateBasic() (err sdk.Error) {
	if len(msg.Owner) == 0 {
		return sdk.ErrInvalidAddress("missing owner address")
	}
	if len(msg.Stock) == 0 || len(msg.Money) == 0 {
		return ErrInvalidSymbol()
	}
	if !market.IsValidTradingPair([]string{msg.Stock, msg.Money}) {
		return ErrInvalidSymbol()
	}
	if !msg.MaxSupply.IsPositive() {
		return ErrNonPositiveSupply()
	}
	if msg.MaxSupply.GT(sdk.NewInt(MaxTradeAmount)) {
		return ErrMaxSupplyTooBig()
	}
	if msg.MaxMoney.IsNegative() {
		return ErrNegativeMaxMoney()
	}
	if msg.MaxMoney.GT(sdk.NewInt(MaxTradeAmount)) {
		return ErrMaxMoneyTooBIg()
	}
	maxPrice, err := sdk.NewDecFromStr(msg.MaxPrice)
	if err != nil {
		return ErrPriceFmt()
	}
	if !maxPrice.IsPositive() {
		return ErrNonPositivePrice()
	}
	initPrice, err := sdk.NewDecFromStr(msg.InitPrice)
	if err != nil {
		return ErrPriceFmt()
	}
	if initPrice.IsNegative() {
		return ErrNegativePrice()
	}

	ar, ok := CheckAR(msg, initPrice, maxPrice)
	if ar > MaxAR || ar < 0 || !ok {
		return ErrAlphaBreakLimit()
	}
	if ar == 0 {
		if err := checkMaxPrice(initPrice, maxPrice, msg.MaxSupply); err != nil {
			return err
		}
	}
	if !CheckStockPrecision(msg.MaxSupply, msg.StockPrecision) {
		return ErrStockSupplyPrecisionNotMatch()
	}
	if msg.EarliestCancelTime < 0 {
		return ErrEarliestCancelTimeIsNegative()
	}
	return nil
}

func checkMaxPrice(initPrice, maxPrice sdk.Dec, maxSupply sdk.Int) (err sdk.Error) {
	if initPrice.GT(maxPrice) {
		return ErrPriceConfiguration()
	}
	defer func() {
		if r := recover(); r != nil {
			err = ErrPriceTooBig()
		}
	}()

	if initPrice.Add(maxPrice).QuoInt64(2).MulInt(maxSupply).GT(sdk.NewDec(MaxTradeAmount)) {
		return ErrPriceTooBig()
	}

	return
}

func CheckStockPrecision(amount sdk.Int, precision byte) bool {
	if precision > 8 {
		precision = 0
	}
	if precision != 0 {
		mod := sdk.NewInt(int64(math.Pow10(int(precision))))
		if !amount.Mod(mod).IsZero() {
			return false
		}
	}
	return true
}

func CheckAR(msg MsgBancorInit, initPrice, maxPrice sdk.Dec) (int64, bool) {
	cwCalculate := func() (int64, bool) {
		defer func() {
			if r := recover(); r != nil {
			}
		}()
		if !msg.MaxMoney.IsZero() {
			ar := CalculateAR(msg, initPrice, maxPrice)
			if ar == 0 {
				return 0, false
			}
			return ar, true
		}
		return 0, true
	}
	return cwCalculate()
}

func CalculateAR(msg MsgBancorInit, initPrice, maxPrice sdk.Dec) int64 {
	if maxPrice.Equal(initPrice) {
		return 0
	}
	if sdk.NewDecFromInt(msg.MaxMoney).LTE(initPrice.MulInt(msg.MaxSupply)) {
		return 0
	}
	return maxPrice.MulInt(msg.MaxSupply).Sub(sdk.NewDecFromInt(msg.MaxMoney)).
		QuoTruncate(sdk.NewDecFromInt(msg.MaxMoney).Sub(initPrice.MulInt(msg.MaxSupply))).
		MulInt64(ARSamples).TruncateInt64()
}

func (msg MsgBancorInit) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgBancorInit) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Owner}
}

func (msg MsgBancorCancel) Route() string { return RouterKey }

func (msg MsgBancorCancel) Type() string { return "bancor_cancel" }

func (msg MsgBancorCancel) ValidateBasic() sdk.Error {
	if len(msg.Owner) == 0 {
		return sdk.ErrInvalidAddress("missing owner address")
	}
	if len(msg.Stock) == 0 || len(msg.Money) == 0 {
		return ErrInvalidSymbol()
	}
	return nil
}

func (msg MsgBancorCancel) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgBancorCancel) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Owner}
}

func (msg MsgBancorTrade) Route() string { return RouterKey }

func (msg MsgBancorTrade) Type() string { return "bancor_trade" }

func (msg MsgBancorTrade) ValidateBasic() sdk.Error {
	if len(msg.Sender) == 0 {
		return sdk.ErrInvalidAddress("missing sender address")
	}
	if len(msg.Stock) == 0 || len(msg.Money) == 0 || msg.Stock == "cet" {
		return ErrInvalidSymbol()
	}
	if !market.IsValidTradingPair([]string{msg.Stock, msg.Money}) {
		return ErrInvalidSymbol()
	}
	if msg.Amount <= 0 {
		return ErrNonPositiveAmount()
	}
	if msg.Amount > MaxTradeAmount {
		return ErrTradeAmountIsTooLarge()
	}
	return nil
}

func (msg MsgBancorTrade) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgBancorTrade) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// --------------------------------------------------------
// SetAccAddress

func (msg *MsgBancorInit) SetAccAddress(addr sdk.AccAddress) {
	msg.Owner = addr
}
func (msg *MsgBancorTrade) SetAccAddress(addr sdk.AccAddress) {
	msg.Sender = addr
}
func (msg *MsgBancorCancel) SetAccAddress(addr sdk.AccAddress) {
	msg.Owner = addr
}
