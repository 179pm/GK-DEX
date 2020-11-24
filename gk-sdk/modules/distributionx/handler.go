package distributionx

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/coinexchain/cet-sdk/modules/distributionx/types"
	dex "github.com/coinexchain/cet-sdk/types"
)

func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case types.MsgDonateToCommunityPool:
			return handleMsgDonateToCommunityPool(ctx, k, msg)
		default:
			return dex.ErrUnknownRequest(ModuleName, msg)
		}
	}
}

func handleMsgDonateToCommunityPool(ctx sdk.Context, k Keeper, msg types.MsgDonateToCommunityPool) sdk.Result {
	err := k.DonateToCommunityPool(ctx, msg.FromAddr, msg.Amount)
	if err != nil {
		return err.Result()
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.FromAddr.String()),
		),
	})
	return sdk.Result{
		Events: ctx.EventManager().Events(),
	}
}
