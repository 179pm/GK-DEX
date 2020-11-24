package rest

import (
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"

	"github.com/coinexchain/cet-sdk/modules/market/internal/types"
	"github.com/coinexchain/cosmos-utils/client/restutil"
)

// SendReq defines the properties of a send request's body.
type createMarketReq struct {
	BaseReq        rest.BaseReq `json:"base_req"`
	Stock          string       `json:"stock"`
	Money          string       `json:"money"`
	PricePrecision int          `json:"price_precision"`
	OrderPrecision int          `json:"order_precision,omitempty"`
	BuyFeeRate    sdk.Dec    `json:"buy_fee_rate"`
	SellFeeRate    sdk.Dec    `json:"sell_fee_rate"`
}

func (req *createMarketReq) New() restutil.RestReq {
	return new(createMarketReq)
}
func (req *createMarketReq) GetBaseReq() *rest.BaseReq {
	return &req.BaseReq
}
func (req *createMarketReq) GetMsg(r *http.Request, sender sdk.AccAddress) (sdk.Msg, error) {
	msg := types.NewMsgCreateTradingPair(req.Stock, req.Money, sender, byte(req.PricePrecision), 
	byte(req.OrderPrecision),req.BuyFeeRate, req.SellFeeRate)
	return msg, nil
}

type cancelMarketReq struct {
	BaseReq     rest.BaseReq `json:"base_req"`
	TradingPair string       `json:"trading_pair"`
	Time        int64        `json:"time"`
}

func (req *cancelMarketReq) New() restutil.RestReq {
	return new(cancelMarketReq)
}
func (req *cancelMarketReq) GetBaseReq() *rest.BaseReq {
	return &req.BaseReq
}
func (req *cancelMarketReq) GetMsg(r *http.Request, sender sdk.AccAddress) (sdk.Msg, error) {
	msg := types.MsgCancelTradingPair{
		Sender:        sender,
		TradingPair:   req.TradingPair,
		EffectiveTime: req.Time,
	}
	return msg, nil
}

type modifyPricePrecision struct {
	BaseReq        rest.BaseReq `json:"base_req"`
	TradingPair    string       `json:"trading_pair"`
	PricePrecision int          `json:"price_precision"`
}

func (req *modifyPricePrecision) New() restutil.RestReq {
	return new(modifyPricePrecision)
}
func (req *modifyPricePrecision) GetBaseReq() *rest.BaseReq {
	return &req.BaseReq
}
func (req *modifyPricePrecision) GetMsg(r *http.Request, sender sdk.AccAddress) (sdk.Msg, error) {
	msg := types.MsgModifyPricePrecision{
		Sender:         sender,
		TradingPair:    req.TradingPair,
		PricePrecision: byte(req.PricePrecision),
	}
	return msg, nil
}

func createMarketHandlerFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	var req createMarketReq
	return restutil.NewRestHandler(cdc, cliCtx, &req)
}

func cancelMarketHandlerFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	var req cancelMarketReq
	return restutil.NewRestHandler(cdc, cliCtx, &req)
}

func modifyTradingPairPricePrecision(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	var req modifyPricePrecision
	return restutil.NewRestHandler(cdc, cliCtx, &req)
}

type modifyFeeRate struct {
	BaseReq        rest.BaseReq `json:"base_req"`
	TradingPair    string       `json:"trading_pair"`
	BuyFeeRate sdk.Dec          `json:"buy_fee_rate"`
	SellFeeRate sdk.Dec          `json:"sell_fee_rate"`
}

func (req *modifyFeeRate) New() restutil.RestReq {
	return new(modifyFeeRate)
}
func (req *modifyFeeRate) GetBaseReq() *rest.BaseReq {
	return &req.BaseReq
}
func (req *modifyFeeRate) GetMsg(r *http.Request, sender sdk.AccAddress) (sdk.Msg, error) {
	msg := types.MsgModifyFeeRate{
		Sender:         sender,
		TradingPair: req.TradingPair,
		BuyFeeRate:    req.BuyFeeRate,
		SellFeeRate: req.SellFeeRate,
	}
	return msg, nil
}

func modifyFeeRateFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	var req modifyFeeRate
	return restutil.NewRestHandler(cdc, cliCtx, &req)
}
