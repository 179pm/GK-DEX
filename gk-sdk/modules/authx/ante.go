package authx

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	"github.com/cosmos/cosmos-sdk/x/supply"
	"github.com/tendermint/tendermint/types"

	dex "github.com/coinexchain/cet-sdk/types"
)

type AnteHelper interface {
	CheckMsg(ctx sdk.Context, msg sdk.Msg, memo string) sdk.Error
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func NewAnteHandler(ak auth.AccountKeeper, supplyKeeper authtypes.SupplyKeeper,
	axk AccountXKeeper, anteHelper AnteHelper) sdk.AnteHandler {

	ah := auth.NewAnteHandler(ak, supplyKeeper, auth.DefaultSigVerificationGasConsumer)
	return WrapAnteHandler(ah, axk, anteHelper)
}

func WrapAnteHandler(ah sdk.AnteHandler,
	axk AccountXKeeper, anteHelper AnteHelper) sdk.AnteHandler {

	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, res sdk.Result, abort bool) {
		// run auth.AnteHandler first
		newCtx, res, abort = ah(ctx, tx, simulate)
		if !res.IsOK() {
			return
		}

		// then, do additional check
		stdTx, _ := tx.(auth.StdTx)
		if err := doAdditionalCheck(ctx, stdTx, simulate, axk, anteHelper); err != nil {
			res = err.Result()
			abort = true
		}

		//finally, check msg type to call PreTotalSupply conditionally
		for _, msg := range stdTx.Msgs {
			switch msg := msg.(type) {
			case crisis.MsgVerifyInvariant:
				if msg.InvariantModuleName == supply.ModuleName {
					axk.PreTotalSupply(ctx)
				}
			}
		}

		return
	}
}

func doAdditionalCheck(ctx sdk.Context, tx auth.StdTx, simulate bool,
	axk AccountXKeeper, anteHelper AnteHelper) sdk.Error {

	if !simulate {
		if err := checkGasPrice(ctx, tx, axk); err != nil {
			return err
		}
	}

	memo := tx.Memo
	for _, msg := range tx.Msgs {
		if err := anteHelper.CheckMsg(ctx, msg, memo); err != nil {
			return err
		}
	}
	return nil
}

func checkGasPrice(ctx sdk.Context, tx auth.StdTx, axk AccountXKeeper) sdk.Error {
	if ctx.BlockHeader().Height == types.GenesisBlockHeight {
		// do not check gas price during the genesis block
		return nil
	}

	gasPrice := tx.Fee.GasPrices().AmountOf(dex.CET)
	minGasPrice := axk.GetParams(ctx).MinGasPriceLimit
	if gasPrice.LT(minGasPrice) {
		return ErrGasPriceTooLow(minGasPrice, gasPrice)
	}
	return nil
}
